package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgx5migrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// TestElectricChargingMigration exercises 000010 up and down on a database of
// its own, so stepping the schema back cannot disturb other packages' tests.
func TestElectricChargingMigration(t *testing.T) {
	base := testDatabaseURL(t)
	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}
	defer admin.Close()

	name := fmt.Sprintf("electric_charging_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create scratch database: %v", err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE " + name + " WITH (FORCE)"); err != nil {
			t.Errorf("drop scratch database: %v", err)
		}
	}()
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	parsed.Path = "/" + name
	scratchURL := parsed.String()

	sqlDB, err := sql.Open("pgx", scratchURL)
	if err != nil {
		t.Fatalf("open scratch database: %v", err)
	}
	defer sqlDB.Close()
	m := scratchMigrator(t, sqlDB)

	if err := m.Migrate(9); err != nil {
		t.Fatalf("migrate to 9: %v", err)
	}
	accountID := insertAccount(t, sqlDB, "driver@example.com")
	carID := insertCar(t, sqlDB, accountID)
	if _, err := sqlDB.Exec("INSERT INTO refuels (car_id, date, station, fuel, liters, amount) VALUES ($1, now(), 'Station', 'special', 42.5, 70)", carID); err != nil {
		t.Fatalf("insert refuel at version 9: %v", err)
	}

	if err := m.Migrate(10); err != nil {
		t.Fatalf("migrate to 10: %v", err)
	}
	var fuel, quantity string
	if err := sqlDB.QueryRow("SELECT fuel::text, quantity::text FROM refuels WHERE car_id = $1", carID).Scan(&fuel, &quantity); err != nil {
		t.Fatalf("read refuel at version 10: %v", err)
	}
	if fuel != "special" || quantity != "42.5" {
		t.Errorf("refuel after up = (%s, %s), want (special, 42.5)", fuel, quantity)
	}

	// Rolling back is refused while a charging subtype is stored.
	if _, err := sqlDB.Exec("INSERT INTO refuels (car_id, date, station, fuel, quantity, amount) VALUES ($1, now(), 'Wallbox', 'home', 40, 12)", carID); err != nil {
		t.Fatalf("insert charge: %v", err)
	}
	if err := m.Migrate(9); err == nil {
		t.Fatal("migrate down with charges stored: expected error, got nil")
	}
	if err := m.Force(10); err != nil {
		t.Fatalf("clear dirty state: %v", err)
	}
	if _, err := sqlDB.Exec("DELETE FROM refuels WHERE fuel = 'home'"); err != nil {
		t.Fatalf("delete charge: %v", err)
	}

	if err := m.Migrate(9); err != nil {
		t.Fatalf("migrate down to 9: %v", err)
	}
	if err := sqlDB.QueryRow("SELECT fuel::text, liters::text FROM refuels WHERE car_id = $1", carID).Scan(&fuel, &quantity); err != nil {
		t.Fatalf("read refuel after down: %v", err)
	}
	if fuel != "special" || quantity != "42.5" {
		t.Errorf("refuel after down = (%s, %s), want (special, 42.5)", fuel, quantity)
	}
	assertRejected(t, sqlDB, "INSERT INTO refuels (car_id, date, station, fuel, liters, amount) VALUES ($1, now(), 'Wallbox', 'dc', 40, 12)", carID)
}

func scratchMigrator(t *testing.T, sqlDB *sql.DB) *migrate.Migrate {
	t.Helper()
	driver, err := pgx5migrate.WithInstance(sqlDB, &pgx5migrate.Config{})
	if err != nil {
		t.Fatalf("create migration driver: %v", err)
	}
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("load embedded migrations: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	return m
}
