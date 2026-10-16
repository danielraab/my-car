package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// testDatabaseURL returns DATABASE_URL, skipping the test if it isn't set.
// CI provisions a Postgres service and sets it; a plain `go test ./...`
// without one skips rather than failing.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping test that requires a live Postgres database")
	}
	return url
}

func TestNewPool_Ping(t *testing.T) {
	url := testDatabaseURL(t)
	ctx := context.Background()

	pool, err := NewPool(ctx, url)
	if err != nil {
		t.Fatalf("NewPool() error: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping() error: %v", err)
	}
}

func TestNewPool_Unreachable(t *testing.T) {
	ctx := context.Background()

	_, err := NewPool(ctx, "postgres://nobody:nobody@127.0.0.1:1/nonexistent?connect_timeout=1")
	if err == nil {
		t.Fatal("NewPool() with an unreachable database: expected error, got nil")
	}
}

func TestRunMigrations(t *testing.T) {
	url := testDatabaseURL(t)

	if err := RunMigrations(url); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	// Running again against an already up-to-date database must not fail.
	if err := RunMigrations(url); err != nil {
		t.Fatalf("RunMigrations() second call error: %v", err)
	}

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}
	defer sqlDB.Close()

	var version int
	var dirty bool
	if err := sqlDB.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if version != 10 {
		t.Errorf("schema_migrations.version = %d, want 10", version)
	}
	if dirty {
		t.Errorf("schema_migrations.dirty = true, want false")
	}
}

func TestDomainSchema(t *testing.T) {
	url := testDatabaseURL(t)
	if err := RunMigrations(url); err != nil {
		t.Fatalf("RunMigrations() error: %v", err)
	}

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}
	defer sqlDB.Close()
	lock, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("acquire schema-test lock connection: %v", err)
	}
	if _, err = lock.ExecContext(context.Background(), "SELECT pg_advisory_lock(7242026)"); err != nil {
		lock.Close()
		t.Fatalf("acquire schema-test lock: %v", err)
	}
	defer func() {
		_, _ = lock.ExecContext(context.Background(), "SELECT pg_advisory_unlock(7242026)")
		_ = lock.Close()
	}()

	if _, err := sqlDB.Exec("TRUNCATE accounts, attachment_file_deletions CASCADE"); err != nil {
		t.Fatalf("truncate domain tables: %v", err)
	}
	defer func() {
		if _, err := sqlDB.Exec("TRUNCATE accounts, attachment_file_deletions CASCADE"); err != nil {
			t.Errorf("cleanup domain tables: %v", err)
		}
	}()

	assertSchemaObjects(t, sqlDB)
	accountID := insertAccount(t, sqlDB, "driver@example.com")
	assertAccountDisplaySettings(t, sqlDB, accountID)
	assertRejected(t, sqlDB, "INSERT INTO oidc_login_attempts (state_digest, nonce, pkce_verifier, return_to, expires_at) VALUES (decode('00','hex'), 'nonce', 'verifier', '/', now() + interval '1 minute')")
	assertRejected(t, sqlDB, "INSERT INTO magic_link_challenges (token_digest, email, return_to, expires_at) VALUES (decode('00','hex'), 'other@example.com', '/', now() + interval '1 minute')")
	assertRejected(t, sqlDB, "INSERT INTO accounts (email) VALUES ('DRIVER@example.com')")
	assertRejected(t, sqlDB, "INSERT INTO accounts (email) VALUES ('driver@example.com')")

	carID := insertCar(t, sqlDB, accountID)
	assertRejected(t, sqlDB, "INSERT INTO cars (account_id, type, make, name, fuel, first_registration, license_plate) VALUES ($1, ' ', 'Ford', 'Focus', 'gasoline', '2020-01-01', 'AB-123')", accountID)
	assertOptionalRegistration(t, sqlDB, accountID)
	assertDeregistrationAndInformation(t, sqlDB, accountID)
	assertPasskeySchema(t, sqlDB, accountID)

	assertNullableOdometerAndNumericChecks(t, sqlDB, carID)
	assertUpdatedAtTrigger(t, sqlDB, carID)
	assertAttachments(t, sqlDB, accountID, carID)
	assertCarDeleteCascades(t, sqlDB, carID)
}

func assertSchemaObjects(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	for _, table := range []string{"accounts", "cars", "refuels", "repairs", "tickets", "oidc_identities", "oidc_login_attempts", "magic_link_challenges", "sessions", "webauthn_credentials", "webauthn_challenges", "attachments", "attachment_file_deletions"} {
		var exists bool
		if err := sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)", table).Scan(&exists); err != nil {
			t.Fatalf("check %s table: %v", table, err)
		}
		if !exists {
			t.Errorf("%s table does not exist", table)
		}
	}

	for enumName, wantValues := range map[string]int{"vehicle_fuel": 4, "refuel_fuel": 6, "repair_type": 4, "ticket_type": 3} {
		var got int
		if err := sqlDB.QueryRow("SELECT count(*) FROM pg_enum JOIN pg_type ON pg_type.oid = pg_enum.enumtypid WHERE pg_type.typname = $1", enumName).Scan(&got); err != nil {
			t.Fatalf("count %s values: %v", enumName, err)
		}
		if got != wantValues {
			t.Errorf("%s enum has %d values, want %d", enumName, got, wantValues)
		}
	}

	for _, index := range []string{"cars_account_created_at_id_idx", "refuels_car_date_id_idx", "repairs_car_date_id_idx", "tickets_car_date_id_idx", "oidc_login_attempts_expiry_idx", "magic_link_challenges_expiry_idx", "sessions_account_expiry_idx", "sessions_expiry_idx", "webauthn_credentials_account_created_at_id_idx", "webauthn_challenges_expiry_idx", "sessions_credential_idx", "attachments_car_idx", "attachments_refuel_idx", "attachments_repair_idx", "attachments_ticket_idx"} {
		var exists bool
		if err := sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1)", index).Scan(&exists); err != nil {
			t.Fatalf("check %s index: %v", index, err)
		}
		if !exists {
			t.Errorf("%s does not exist", index)
		}
	}
}

func assertPasskeySchema(t *testing.T, sqlDB *sql.DB, accountID string) {
	t.Helper()
	insert := "INSERT INTO webauthn_credentials (credential_id, account_id, public_key, aaguid, user_verified, backup_eligible, backup_state, name) VALUES ($1, $2, decode('01','hex'), $3, true, $4, $5, $6) RETURNING id"
	var credentialID string
	if err := sqlDB.QueryRow(insert, []byte("cred-1"), accountID, make([]byte, 16), true, true, "Laptop").Scan(&credentialID); err != nil {
		t.Fatalf("insert passkey: %v", err)
	}
	assertRejected(t, sqlDB, insert, []byte("cred-1"), accountID, nil, false, false, "Duplicate")
	assertRejected(t, sqlDB, insert, []byte("cred-2"), accountID, nil, false, false, "  ")
	assertRejected(t, sqlDB, insert, []byte("cred-3"), accountID, nil, false, true, "Backed up but not eligible")
	assertRejected(t, sqlDB, insert, []byte("cred-4"), accountID, []byte("short"), false, false, "Bad AAGUID")

	challenge := "INSERT INTO webauthn_challenges (ceremony_digest, kind, account_id, session_data, expires_at) VALUES ($1, $2, $3, '{}', now() + interval '5 minutes')"
	digest := make([]byte, 32)
	assertRejected(t, sqlDB, challenge, digest, "login", accountID)
	assertRejected(t, sqlDB, challenge, digest, "registration", nil)
	assertRejected(t, sqlDB, challenge, digest[:8], "login", nil)
	if _, err := sqlDB.Exec(challenge, digest, "login", nil); err != nil {
		t.Fatalf("insert login challenge: %v", err)
	}

	sessionDigest := make([]byte, 32)
	sessionDigest[0] = 1
	if _, err := sqlDB.Exec("INSERT INTO sessions (token_digest, account_id, expires_at, credential_id) VALUES ($1, $2, now() + interval '1 hour', $3)", sessionDigest, accountID, credentialID); err != nil {
		t.Fatalf("insert passkey session: %v", err)
	}
	if _, err := sqlDB.Exec("DELETE FROM webauthn_credentials WHERE id = $1", credentialID); err != nil {
		t.Fatalf("delete passkey: %v", err)
	}
	var remaining sql.NullString
	if err := sqlDB.QueryRow("SELECT credential_id FROM sessions WHERE token_digest = $1", sessionDigest).Scan(&remaining); err != nil {
		t.Fatalf("read session after passkey deletion: %v", err)
	}
	if remaining.Valid {
		t.Errorf("session credential_id = %q after deletion, want NULL", remaining.String)
	}
	if _, err := sqlDB.Exec("DELETE FROM sessions WHERE token_digest = $1", sessionDigest); err != nil {
		t.Fatalf("cleanup passkey session: %v", err)
	}
}

func insertAccount(t *testing.T, sqlDB *sql.DB, email string) string {
	t.Helper()
	var id string
	if err := sqlDB.QueryRow("INSERT INTO accounts (email) VALUES ($1) RETURNING id", email).Scan(&id); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	return id
}

func assertAccountDisplaySettings(t *testing.T, sqlDB *sql.DB, accountID string) {
	t.Helper()
	var currency, volumeUnit, distanceUnit string
	if err := sqlDB.QueryRow("SELECT currency, volume_unit, distance_unit FROM accounts WHERE id = $1", accountID).Scan(&currency, &volumeUnit, &distanceUnit); err != nil {
		t.Fatalf("read account display settings: %v", err)
	}
	if currency != "EUR" || volumeUnit != "liter" || distanceUnit != "kilometer" {
		t.Errorf("display settings = %s/%s/%s, want EUR/liter/kilometer", currency, volumeUnit, distanceUnit)
	}
	if _, err := sqlDB.Exec("UPDATE accounts SET currency = 'USD', volume_unit = 'gallon', distance_unit = 'mile' WHERE id = $1", accountID); err != nil {
		t.Errorf("set valid display settings: %v", err)
	}
	assertRejected(t, sqlDB, "UPDATE accounts SET currency = 'usd' WHERE id = $1", accountID)
	assertRejected(t, sqlDB, "UPDATE accounts SET currency = 'EURO' WHERE id = $1", accountID)
	assertRejected(t, sqlDB, "UPDATE accounts SET volume_unit = 'barrel' WHERE id = $1", accountID)
	assertRejected(t, sqlDB, "UPDATE accounts SET distance_unit = 'furlong' WHERE id = $1", accountID)
	if _, err := sqlDB.Exec("UPDATE accounts SET currency = DEFAULT, volume_unit = DEFAULT, distance_unit = DEFAULT WHERE id = $1", accountID); err != nil {
		t.Errorf("reset display settings: %v", err)
	}
}

func insertCar(t *testing.T, sqlDB *sql.DB, accountID string) string {
	t.Helper()
	var id string
	if err := sqlDB.QueryRow("INSERT INTO cars (account_id, type, make, name, fuel, first_registration, license_plate) VALUES ($1, 'Car', 'Ford', 'Focus', 'gasoline', '2020-01-01', 'AB-123') RETURNING id", accountID).Scan(&id); err != nil {
		t.Fatalf("insert car: %v", err)
	}
	return id
}

func assertOptionalRegistration(t *testing.T, sqlDB *sql.DB, accountID string) {
	t.Helper()
	var registration sql.NullTime
	var plate sql.NullString
	if err := sqlDB.QueryRow("INSERT INTO cars (account_id, type, make, name, fuel) VALUES ($1, 'Car', 'Ford', 'Ka', 'gasoline') RETURNING first_registration, license_plate", accountID).Scan(&registration, &plate); err != nil {
		t.Fatalf("insert car without registration details: %v", err)
	}
	if registration.Valid || plate.Valid {
		t.Errorf("first_registration = %v, license_plate = %v, want both NULL", registration, plate)
	}
	assertRejected(t, sqlDB, "INSERT INTO cars (account_id, type, make, name, fuel, license_plate) VALUES ($1, 'Car', 'Ford', 'Ka', 'gasoline', '  ')", accountID)
}

func assertDeregistrationAndInformation(t *testing.T, sqlDB *sql.DB, accountID string) {
	t.Helper()
	var deregistration sql.NullTime
	var information sql.NullString
	if err := sqlDB.QueryRow("INSERT INTO cars (account_id, type, make, name, fuel) VALUES ($1, 'Car', 'Ford', 'Puma', 'gasoline') RETURNING deregistration_date, information", accountID).Scan(&deregistration, &information); err != nil {
		t.Fatalf("insert car without deregistration and information: %v", err)
	}
	if deregistration.Valid || information.Valid {
		t.Errorf("deregistration_date = %v, information = %v, want both NULL", deregistration, information)
	}
	if err := sqlDB.QueryRow("INSERT INTO cars (account_id, type, make, name, fuel, first_registration, deregistration_date, information) VALUES ($1, 'Car', 'Ford', 'Puma', 'gasoline', '2020-01-01', '2020-01-01', 'Sold\nto a friend') RETURNING deregistration_date, information", accountID).Scan(&deregistration, &information); err != nil {
		t.Fatalf("insert car with deregistration and information: %v", err)
	}
	if got := deregistration.Time.Format("2006-01-02"); !deregistration.Valid || got != "2020-01-01" {
		t.Errorf("deregistration_date = %v, want 2020-01-01", deregistration)
	}
	if !information.Valid || information.String != "Sold\nto a friend" {
		t.Errorf("information = %v, want the stored text", information)
	}
	assertRejected(t, sqlDB, "INSERT INTO cars (account_id, type, make, name, fuel, information) VALUES ($1, 'Car', 'Ford', 'Puma', 'gasoline', ' \n ')", accountID)
	assertRejected(t, sqlDB, "INSERT INTO cars (account_id, type, make, name, fuel, first_registration, deregistration_date) VALUES ($1, 'Car', 'Ford', 'Puma', 'gasoline', '2020-01-02', '2020-01-01')", accountID)
}

func assertNullableOdometerAndNumericChecks(t *testing.T, sqlDB *sql.DB, carID string) {
	t.Helper()
	var odometer sql.NullInt64
	if err := sqlDB.QueryRow("INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'Station', 'normal', 20.5, 50.2) RETURNING odometer_reading", carID).Scan(&odometer); err != nil {
		t.Fatalf("insert refuel without odometer: %v", err)
	}
	if odometer.Valid {
		t.Error("refuel odometer is present, want NULL")
	}
	assertRejected(t, sqlDB, "INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'Station', 'normal', 0, 50.2)", carID)
	for _, fuel := range []string{"home", "ac", "dc"} {
		if _, err := sqlDB.Exec("INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'Wallbox', $2, 40, 14)", carID, fuel); err != nil {
			t.Errorf("insert refuel with charging subtype %s: %v", fuel, err)
		}
	}
	assertRejected(t, sqlDB, "INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'Station', 'premium', 20, 30)", carID)
	assertRejected(t, sqlDB, "INSERT INTO repairs (car_id, date, station, type, amount) VALUES ($1, now(), 'Garage', 'service', -1)", carID)
}

func assertUpdatedAtTrigger(t *testing.T, sqlDB *sql.DB, carID string) {
	t.Helper()
	var before, after time.Time
	if err := sqlDB.QueryRow("SELECT updated_at FROM cars WHERE id = $1", carID).Scan(&before); err != nil {
		t.Fatalf("read car updated_at before update: %v", err)
	}
	if _, err := sqlDB.Exec("SELECT pg_sleep(0.01)"); err != nil {
		t.Fatalf("sleep before update: %v", err)
	}
	if _, err := sqlDB.Exec("UPDATE cars SET name = 'Fiesta' WHERE id = $1", carID); err != nil {
		t.Fatalf("update car: %v", err)
	}
	if err := sqlDB.QueryRow("SELECT updated_at FROM cars WHERE id = $1", carID).Scan(&after); err != nil {
		t.Fatalf("read car updated_at after update: %v", err)
	}
	if !after.After(before) {
		t.Errorf("updated_at = %s after update, want value after %s", after, before)
	}
}

func assertCarDeleteCascades(t *testing.T, sqlDB *sql.DB, carID string) {
	t.Helper()
	for _, statement := range []string{
		"INSERT INTO repairs (car_id, date, station, type, amount) VALUES ($1, now(), 'Garage', 'service', 10)",
		"INSERT INTO tickets (car_id, date, type, location, amount) VALUES ($1, now(), 'parking', 'Vienna', 20)",
	} {
		if _, err := sqlDB.Exec(statement, carID); err != nil {
			t.Fatalf("insert dependent expense: %v", err)
		}
	}
	if _, err := sqlDB.Exec("DELETE FROM cars WHERE id = $1", carID); err != nil {
		t.Fatalf("delete car: %v", err)
	}
	for _, table := range []string{"refuels", "repairs", "tickets"} {
		var count int
		if err := sqlDB.QueryRow("SELECT count(*) FROM "+table+" WHERE car_id = $1", carID).Scan(&count); err != nil {
			t.Fatalf("count %s after car deletion: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s has %d rows after car deletion, want 0", table, count)
		}
	}
	assertQueued(t, sqlDB, "k-car", true)
	assertQueued(t, sqlDB, "k-refuel", true)
}

// assertAttachments checks the exactly-one-owner rule and that deleting an
// attachment, directly or through a cascade from its car's expense, queues
// its stored file for removal. It leaves one attachment on carID and one on
// a refuel of carID for assertCarDeleteCascades.
func assertAttachments(t *testing.T, sqlDB *sql.DB, accountID, carID string) {
	t.Helper()
	var refuelID, ticketID string
	if err := sqlDB.QueryRow("INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'Station', 'normal', 10, 20) RETURNING id", carID).Scan(&refuelID); err != nil {
		t.Fatalf("insert refuel: %v", err)
	}
	if err := sqlDB.QueryRow("INSERT INTO tickets (car_id, date, type, location, amount) VALUES ($1, now(), 'parking', 'Linz', 5) RETURNING id", carID).Scan(&ticketID); err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	insert := "INSERT INTO attachments (id, account_id, car_id, refuel_id, ticket_id, storage_key, filename, content_type, size_bytes) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, 'receipt.jpg', $6, 10)"
	assertRejected(t, sqlDB, insert, accountID, nil, nil, nil, "k-none", "image/jpeg")
	assertRejected(t, sqlDB, insert, accountID, nil, refuelID, ticketID, "k-two", "image/jpeg")
	assertRejected(t, sqlDB, insert, accountID, carID, nil, nil, "k-svg", "image/svg+xml")
	for key, owner := range map[string][3]any{"k-car": {carID, nil, nil}, "k-refuel": {nil, refuelID, nil}, "k-ticket": {nil, nil, ticketID}} {
		if _, err := sqlDB.Exec(insert, accountID, owner[0], owner[1], owner[2], key, "image/jpeg"); err != nil {
			t.Fatalf("insert attachment %s: %v", key, err)
		}
	}
	assertRejected(t, sqlDB, insert, accountID, carID, nil, nil, "k-car", "image/jpeg")

	if _, err := sqlDB.Exec("DELETE FROM tickets WHERE id = $1", ticketID); err != nil {
		t.Fatalf("delete ticket: %v", err)
	}
	assertQueued(t, sqlDB, "k-ticket", true)
	assertQueued(t, sqlDB, "k-car", false)
}

func assertQueued(t *testing.T, sqlDB *sql.DB, key string, want bool) {
	t.Helper()
	var queued bool
	if err := sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM attachment_file_deletions WHERE storage_key = $1)", key).Scan(&queued); err != nil {
		t.Fatalf("check file deletion queue: %v", err)
	}
	if queued != want {
		t.Errorf("storage key %s queued for deletion = %v, want %v", key, queued, want)
	}
}

func assertRejected(t *testing.T, sqlDB *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := sqlDB.Exec(statement, args...); err == nil {
		t.Errorf("statement succeeded, want constraint failure: %s", statement)
	}
}
