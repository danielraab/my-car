package tickets

import (
	"context"
	"errors"
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
	email := "tickets-test-" + time.Now().Format("150405.000000000") + "@example.com"
	if err := pool.QueryRow(ctx, `INSERT INTO accounts (email) VALUES ($1) RETURNING id`, email).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO cars (account_id, type, make, name, fuel, first_registration, license_plate)
VALUES ($1, 'Hatchback', 'VW', 'Golf', 'diesel', '2019-03-01', 'W-1') RETURNING id`, accountID).Scan(&carID); err != nil {
		t.Fatal(err)
	}
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
VALUES ($1, 'Van', 'Ford', 'Transit', 'diesel', '2020-01-01', 'W-2') RETURNING id`, accountID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func ptr[T any](v T) *T { return &v }

var base = time.Date(2026, 1, 10, 9, 30, 0, 0, time.UTC)

func validInput(carID string, day int, location string) Input {
	return Input{CarID: carID, Date: base.AddDate(0, 0, day), Type: "parking", Location: location, Amount: "36.00"}
}

func TestStoreListPaginatesAndFilters(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	secondCar := addCar(t, pool, account)
	other, otherCar := testCar(t, pool)
	if _, err := s.Create(ctx, other, validInput(otherCar, 0, "not mine")); err != nil {
		t.Fatal(err)
	}
	var created []Ticket
	for i, c := range []string{car, secondCar, car} {
		// Created out of date order to show listing is by date, not insertion.
		r, err := s.Create(ctx, account, validInput(c, 2-i, "Vienna"))
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, r)
	}
	byDate := []Ticket{created[0], created[1], created[2]} // newest first

	page, next, err := s.List(ctx, account, Filter{}, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != byDate[0].ID || page[1].ID != byDate[1].ID || next == nil {
		t.Fatalf("first page = %v, next = %v", page, next)
	}
	decoded, ok := DecodeCursor(next.Encode())
	if !ok || !decoded.Date.Equal(next.Date) || decoded.ID != next.ID {
		t.Fatalf("cursor round trip = %v, %v", decoded, ok)
	}
	page, next, err = s.List(ctx, account, Filter{}, &decoded, 2)
	if err != nil || len(page) != 1 || page[0].ID != byDate[2].ID || next != nil {
		t.Fatalf("last page = %v, next = %v, err = %v", page, next, err)
	}

	page, _, err = s.List(ctx, account, Filter{CarID: &car}, nil, 10)
	if err != nil || len(page) != 2 || page[0].CarID != car || page[1].CarID != car {
		t.Fatalf("car filter = %v, %v", page, err)
	}
	page, _, err = s.List(ctx, account, Filter{CarID: &otherCar}, nil, 10)
	if err != nil || len(page) != 0 {
		t.Fatalf("another account's car filter = %v, %v", page, err)
	}

	from, to := base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)
	page, _, err = s.List(ctx, account, Filter{From: &from, To: &to}, nil, 10)
	if err != nil || len(page) != 1 || page[0].ID != created[1].ID {
		t.Fatalf("date range (from inclusive, to exclusive) = %v, %v", page, err)
	}

	// Same date: ties break by descending ID.
	tieAccount, tieCar := testCar(t, pool)
	for range 2 {
		if _, err := s.Create(ctx, tieAccount, validInput(tieCar, 0, "Vienna")); err != nil {
			t.Fatal(err)
		}
	}
	page, _, err = s.List(ctx, tieAccount, Filter{}, nil, 10)
	if err != nil || len(page) != 2 || page[0].ID < page[1].ID {
		t.Fatalf("same-date tie = %v, %v", page, err)
	}
}

func TestStoreCreateGetDelete(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	other, otherCar := testCar(t, pool)

	if _, err := s.Create(ctx, account, validInput(otherCar, 0, "x")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("create on another account's car err = %v", err)
	}

	in := validInput(car, 0, "Graz")
	in.Description = "Short-term parking zone"
	r, err := s.Create(ctx, account, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.CarID != car || !r.Date.Equal(base) || r.Location != "Graz" ||
		r.Type != "parking" || r.Amount != "36.00" || r.Description != "Short-term parking zone" {
		t.Fatalf("created = %+v", r)
	}

	bare, err := s.Create(ctx, account, validInput(car, 1, "Graz"))
	if err != nil || bare.Description != "" {
		t.Fatalf("created without description = %+v, %v", bare, err)
	}

	got, err := s.Get(ctx, account, r.ID)
	if err != nil || got.ID != r.ID || got.Amount != "36.00" {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if _, err := s.Get(ctx, other, r.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get as other account err = %v", err)
	}
	if err := s.Delete(ctx, other, r.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete as other account err = %v", err)
	}
	if err := s.Delete(ctx, account, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, account, r.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete err = %v", err)
	}
}

func TestStoreUpdate(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	other, _ := testCar(t, pool)

	in := validInput(car, 0, "Linz")
	in.Description = "Radar"
	r, err := s.Create(ctx, account, in)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.Update(ctx, account, r.ID, Patch{Amount: ptr("0"), Type: ptr("velocity")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != "0" || updated.Type != "velocity" || updated.Description != "Radar" ||
		updated.Location != "Linz" || !updated.Date.Equal(base) {
		t.Fatalf("updated = %+v", updated)
	}

	later := base.AddDate(0, 1, 0)
	updated, err = s.Update(ctx, account, r.ID, Patch{Description: ptr(""), Location: ptr("Wels"), Date: &later})
	if err != nil || updated.Description != "" || updated.Location != "Wels" || !updated.Date.Equal(later) ||
		updated.Amount != "0" || updated.Type != "velocity" {
		t.Fatalf("second update = %+v, %v", updated, err)
	}

	if _, err := s.Update(ctx, other, r.ID, Patch{Location: ptr("stolen")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update as other account err = %v", err)
	}
}

func TestStoreLocationsAreDistinctSortedAndScoped(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	other, otherCar := testCar(t, pool)

	empty, err := s.Locations(ctx, account)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("locations with no tickets = %#v, %v", empty, err)
	}
	for i, location := range []string{"Zeta", "Alpha", "Zeta"} {
		if _, err := s.Create(ctx, account, validInput(car, i, location)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Create(ctx, other, validInput(otherCar, 0, "Other")); err != nil {
		t.Fatal(err)
	}
	locations, err := s.Locations(ctx, account)
	if err != nil || len(locations) != 2 || locations[0] != "Alpha" || locations[1] != "Zeta" {
		t.Fatalf("locations = %v, %v", locations, err)
	}
}
