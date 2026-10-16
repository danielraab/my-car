package stats

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"at.draab/my-car/internal/db"
)

func testStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test")
	}
	if err := db.RunMigrations(url); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Serialize with other packages' database tests, one of which truncates
	// the domain tables while holding this lock.
	lock, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Exec(ctx, `SELECT pg_advisory_lock(7242026)`); err != nil {
		lock.Release()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		lock.Exec(ctx, `SELECT pg_advisory_unlock(7242026)`)
		lock.Release()
	})
	return NewStore(pool), pool
}

// testCar creates an account owning one car and returns both IDs.
func testCar(t *testing.T, pool *pgxpool.Pool) (accountID, carID string) {
	t.Helper()
	ctx := context.Background()
	email := "stats-test-" + time.Now().Format("150405.000000000") + "@example.com"
	if err := pool.QueryRow(ctx, `INSERT INTO accounts (email) VALUES ($1) RETURNING id`, email).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	carID = addCar(t, pool, accountID)
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM cars WHERE account_id = $1`, accountID)
		pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, accountID)
	})
	return accountID, carID
}

func addCar(t *testing.T, pool *pgxpool.Pool, accountID string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `INSERT INTO cars (account_id, type, make, name, fuel, first_registration, license_plate)
VALUES ($1, 'Car', 'VW', 'Golf', 'diesel', '2019-03-01', 'W-1') RETURNING id`, accountID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func addRefuel(t *testing.T, pool *pgxpool.Pool, carID string, date time.Time, amount string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO refuels (car_id, date, station, fuel, quantity, amount)
VALUES ($1, $2, 'Shell', 'normal', 40, $3)`, carID, date, amount); err != nil {
		t.Fatal(err)
	}
}

func addRepair(t *testing.T, pool *pgxpool.Pool, carID string, date time.Time, amount string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO repairs (car_id, date, station, type, amount)
VALUES ($1, $2, 'Garage', 'service', $3)`, carID, date, amount); err != nil {
		t.Fatal(err)
	}
}

func addTicket(t *testing.T, pool *pgxpool.Pool, carID string, date time.Time, amount string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO tickets (car_id, date, type, location, amount)
VALUES ($1, $2, 'parking', 'Vienna', $3)`, carID, date, amount); err != nil {
		t.Fatal(err)
	}
}

var (
	from = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to   = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
)

func TestStoreExpenses(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	secondCar := addCar(t, pool, account)
	other, otherCar := testCar(t, pool)

	// Inserted out of date order to show rows come back by date, then kind.
	addTicket(t, pool, car, time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC), "30")
	addRepair(t, pool, secondCar, time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC), "120.50")
	addRefuel(t, pool, car, from, "61.234")
	addTicket(t, pool, car, from, "15.00")
	addRefuel(t, pool, car, to, "99.00")
	addRefuel(t, pool, car, from.Add(-time.Microsecond), "1.00")
	addRefuel(t, pool, otherCar, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), "44.00")

	got, err := s.Expenses(ctx, account, Filter{From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}
	want := []Expense{
		{Date: from, Kind: "refuel", Amount: "61.234"},
		{Date: from, Kind: "ticket", Amount: "15.00"},
		{Date: time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC), Kind: "repair", Amount: "120.50"},
		{Date: time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC), Kind: "ticket", Amount: "30"},
	}
	if len(got) != len(want) {
		t.Fatalf("expenses = %+v", got)
	}
	for i := range want {
		if !got[i].Date.Equal(want[i].Date) || got[i].Kind != want[i].Kind || got[i].Amount != want[i].Amount {
			t.Fatalf("expense %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	got, err = s.Expenses(ctx, account, Filter{CarID: &secondCar, From: from, To: to})
	if err != nil || len(got) != 1 || got[0].Kind != "repair" {
		t.Fatalf("car filter = %+v, %v", got, err)
	}
	got, err = s.Expenses(ctx, account, Filter{CarID: &otherCar, From: from, To: to})
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("another account's car = %+v, %v", got, err)
	}
	got, err = s.Expenses(ctx, other, Filter{From: from, To: to})
	if err != nil || len(got) != 1 || got[0].Amount != "44.00" {
		t.Fatalf("other account = %+v, %v", got, err)
	}
	got, err = s.Expenses(ctx, account, Filter{From: to, To: from})
	if err != nil || len(got) != 0 {
		t.Fatalf("inverted range = %+v, %v", got, err)
	}
}
