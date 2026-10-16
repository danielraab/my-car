package seed

import (
	"context"
	"os"
	"testing"
	"time"

	"at.draab/my-car/internal/db"
)

func TestRun(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	if err := db.RunMigrations(url); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(7242026)"); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock(7242026)")
	defer pool.Exec(ctx, "TRUNCATE TABLE sessions, webauthn_challenges, webauthn_credentials, oidc_identities, oidc_login_attempts, magic_link_challenges, attachments, attachment_file_deletions, refuels, repairs, tickets, cars, accounts")
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE sessions, webauthn_challenges, webauthn_credentials, oidc_identities, oidc_login_attempts, magic_link_challenges, attachments, attachment_file_deletions, refuels, repairs, tickets, cars, accounts"); err != nil {
		t.Fatal(err)
	}
	var oldID string
	if err := pool.QueryRow(ctx, "INSERT INTO accounts (email) VALUES ('old@example.org') RETURNING id").Scan(&oldID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO sessions (token_digest, account_id, expires_at) VALUES (decode(repeat('ab', 32), 'hex'), $1, now() + interval '1 day')", oldID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO webauthn_credentials (credential_id, account_id, public_key, user_verified, backup_eligible, backup_state, name) VALUES ('\\x01', $1, '\\x01', true, false, false, 'Old key')", oldID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "WITH car AS (INSERT INTO cars (account_id, type, make, name, fuel) VALUES ($1, 'Car', 'VW', 'Golf', 'diesel') RETURNING id) INSERT INTO attachments (id, account_id, car_id, storage_key, filename, content_type, size_bytes) SELECT gen_random_uuid(), $1, id, 'old-key', 'old.pdf', 'application/pdf', 1 FROM car", oldID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := Run(ctx, pool, []string{"one@example.org", "two@example.org"}, now); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"accounts": 2, "cars": 10, "refuels": 1000, "repairs": 150, "tickets": 180, "sessions": 0, "oidc_identities": 0, "oidc_login_attempts": 0, "magic_link_challenges": 0, "webauthn_credentials": 0, "webauthn_challenges": 0, "attachments": 0, "attachment_file_deletions": 1} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != want {
			t.Errorf("%s count = %d, error %v; want %d", table, count, err, want)
		}
	}
	var oldAccounts, wrongCarCounts int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM accounts WHERE email = 'old@example.org'").Scan(&oldAccounts); err != nil || oldAccounts != 0 {
		t.Errorf("old account remains: %d, %v", oldAccounts, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM (SELECT account_id FROM cars GROUP BY account_id HAVING count(*) <> 5) c").Scan(&wrongCarCounts); err != nil || wrongCarCounts != 0 {
		t.Errorf("accounts with incorrect car counts: %d, %v", wrongCarCounts, err)
	}
	var electricAccounts, mismatched, charges int
	if err := pool.QueryRow(ctx, "SELECT count(DISTINCT account_id) FROM cars WHERE fuel = 'electric'").Scan(&electricAccounts); err != nil || electricAccounts != 2 {
		t.Errorf("accounts with an electric car: %d, %v; want 2", electricAccounts, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE (c.fuel = 'electric') <> (r.fuel IN ('home', 'ac', 'dc') OR (r.fuel = 'other' AND c.fuel = 'electric'))),
		count(*) FILTER (WHERE c.fuel = 'electric')
		FROM refuels r JOIN cars c ON c.id = r.car_id`).Scan(&mismatched, &charges); err != nil || mismatched != 0 || charges == 0 {
		t.Errorf("refuels with a subtype not fitting the car: %d, charges: %d, %v", mismatched, charges, err)
	}
	for _, table := range []string{"refuels", "repairs", "tickets"} {
		rows, err := pool.Query(ctx, "SELECT a.email, count(*), min(e.date), max(e.date) FROM "+table+" e JOIN cars c ON c.id=e.car_id JOIN accounts a ON a.id=c.account_id GROUP BY a.email")
		if err != nil {
			t.Fatal(err)
		}
		seen := 0
		for rows.Next() {
			var email string
			var count int
			var earliest, latest time.Time
			if err := rows.Scan(&email, &count, &earliest, &latest); err != nil {
				t.Fatal(err)
			}
			want := map[string]int{"refuels": 500, "repairs": 75, "tickets": 90}[table]
			if count != want || earliest.Before(now.AddDate(-5, 0, 0)) || latest.After(now) || latest.Sub(earliest) < 4*365*24*time.Hour {
				t.Errorf("%s %s: %d records across %s–%s", table, email, count, earliest, latest)
			}
			seen++
		}
		rows.Close()
		if err := rows.Err(); err != nil || seen != 2 {
			t.Errorf("%s: %d accounts, error %v", table, seen, err)
		}
	}
	// A duplicate causes an insert error after TRUNCATE; the old seed must survive.
	if err := Run(ctx, pool, []string{"duplicate@example.org", "duplicate@example.org"}, now); err == nil {
		t.Fatal("expected duplicate insert to fail")
	}
	var accounts, migrations int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM accounts").Scan(&accounts); err != nil || accounts != 2 {
		t.Fatalf("rollback lost accounts: %d, %v", accounts, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&migrations); err != nil || migrations != 1 {
		t.Fatalf("migration state lost: %d, %v", migrations, err)
	}
}
