package legacyimport

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"at.draab/my-car/internal/auth"
	"at.draab/my-car/internal/db"
)

const truncateAll = "TRUNCATE TABLE sessions, webauthn_challenges, webauthn_credentials, oidc_identities, oidc_login_attempts, magic_link_challenges, attachments, attachment_file_deletions, refuels, repairs, tickets, cars, accounts"

// freshDatabase returns a pool on an emptied, migrated database, holding the
// advisory lock the other data-clearing tests use.
func freshDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
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
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(7242026)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, truncateAll)
		conn.Exec(ctx, "SELECT pg_advisory_unlock(7242026)")
		conn.Release()
		pool.Close()
	})
	if _, err := pool.Exec(ctx, truncateAll); err != nil {
		t.Fatal(err)
	}
	return pool
}

func fixturePlan(t *testing.T) Plan {
	t.Helper()
	plan, problems := Map(parseFixture(t))
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	return plan
}

func rowCounts(t *testing.T, pool *pgxpool.Pool) Counts {
	t.Helper()
	var c Counts
	err := pool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM accounts), (SELECT count(*) FROM cars), (SELECT count(*) FROM refuels),
		(SELECT count(*) FROM repairs), (SELECT count(*) FROM tickets)`).
		Scan(&c.Accounts, &c.Cars, &c.Refuels, &c.Repairs, &c.Tickets)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestWriteImportsIntoFreshDatabase(t *testing.T) {
	pool := freshDatabase(t)
	ctx := context.Background()

	counts, err := Write(ctx, pool, fixturePlan(t))
	if err != nil {
		t.Fatal(err)
	}
	want := Counts{Accounts: 2, Cars: 3, Refuels: 3, Repairs: 2, Tickets: 2}
	if counts != want || rowCounts(t, pool) != want {
		t.Fatalf("counts = %+v, stored = %+v, want %+v", counts, rowCounts(t, pool), want)
	}

	var plate, registration, price *string
	var created string
	if err := pool.QueryRow(ctx, `SELECT c.license_plate, c.first_registration::text, c.purchase_price::text, c.created_at::text
		FROM cars c JOIN accounts a ON a.id = c.account_id WHERE a.email = 'grace@example.org'`).
		Scan(&plate, &registration, &price, &created); err != nil {
		t.Fatal(err)
	}
	if plate != nil || registration != nil || price != nil || created != "2024-02-02 17:22:12+00" {
		t.Fatalf("ka = %v %v %v %s", plate, registration, price, created)
	}
	var misowned int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM refuels r JOIN cars c ON c.id = r.car_id JOIN accounts a ON a.id = c.account_id
		WHERE (a.email = 'ada@example.org') <> (c.name IN ('Golf', 'Octavia'))`).Scan(&misowned); err != nil || misowned != 0 {
		t.Fatalf("refuels on the wrong account: %d, %v", misowned, err)
	}
	var amount, liters string
	if err := pool.QueryRow(ctx, `SELECT amount::text, quantity::text FROM refuels WHERE station = 'O''Neill''s Fuel'`).Scan(&amount, &liters); err != nil || amount != "72.31" || liters != "45.2" {
		t.Fatalf("refuel decimals = %s %s, %v", amount, liters, err)
	}

	// Logging in with the legacy address in any case reaches the imported account.
	account, err := auth.NewStore(pool).ResolveOIDCIdentity(ctx, "https://issuer.example", "subject-1", "ADA@example.ORG")
	if err != nil {
		t.Fatal(err)
	}
	var cars int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cars WHERE account_id = $1`, account.ID).Scan(&cars); err != nil || cars != 2 || account.FirstName != "Ada" {
		t.Fatalf("logged-in account %+v owns %d cars, %v", account, cars, err)
	}
}

func TestWriteRefusesDatabaseThatIsNotFresh(t *testing.T) {
	pool := freshDatabase(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO accounts (email) VALUES ('early@example.org')`); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(ctx, pool, fixturePlan(t)); !errors.Is(err, ErrNotFresh) {
		t.Fatalf("err = %v, want ErrNotFresh", err)
	}
	if got := rowCounts(t, pool); got != (Counts{Accounts: 1}) {
		t.Fatalf("stored = %+v", got)
	}
}

func TestWriteSecondRunIsRefused(t *testing.T) {
	pool := freshDatabase(t)
	ctx := context.Background()
	if _, err := Write(ctx, pool, fixturePlan(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := Write(ctx, pool, fixturePlan(t)); !errors.Is(err, ErrNotFresh) {
		t.Fatalf("err = %v, want ErrNotFresh", err)
	}
	if got := rowCounts(t, pool); got != (Counts{Accounts: 2, Cars: 3, Refuels: 3, Repairs: 2, Tickets: 2}) {
		t.Fatalf("stored after second run = %+v", got)
	}
}

func TestWriteFailureLeavesNothing(t *testing.T) {
	pool := freshDatabase(t)
	plan := fixturePlan(t)
	// The last row violates a foreign key after everything else was inserted.
	plan.Tickets[1].CarID = newUUID()
	if _, err := Write(context.Background(), pool, plan); err == nil {
		t.Fatal("write succeeded, want a foreign-key failure")
	}
	if got := rowCounts(t, pool); got != (Counts{}) {
		t.Fatalf("stored after failed write = %+v", got)
	}
}
