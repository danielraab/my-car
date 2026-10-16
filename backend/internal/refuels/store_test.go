package refuels

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
	email := "refuels-test-" + time.Now().Format("150405.000000000") + "@example.com"
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

func validInput(carID string, day int, station string) Input {
	return Input{CarID: carID, Date: base.AddDate(0, 0, day), Station: station, Fuel: "normal", Quantity: "40", Amount: "120.50"}
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
	var created []Refuel
	for i, c := range []string{car, secondCar, car} {
		// Created out of date order to show listing is by date, not insertion.
		r, err := s.Create(ctx, account, validInput(c, 2-i, "Station"))
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, r)
	}
	byDate := []Refuel{created[0], created[1], created[2]} // newest first

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
		if _, err := s.Create(ctx, tieAccount, validInput(tieCar, 0, "Station")); err != nil {
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

	in := validInput(car, 0, "Garage")
	in.OdometerReading = ptr(int64(123456))
	r, err := s.Create(ctx, account, in)
	if err != nil {
		t.Fatal(err)
	}
	if r.CarID != car || !r.Date.Equal(base) || r.Station != "Garage" || *r.OdometerReading != 123456 ||
		r.Fuel != "normal" || r.Amount != "120.50" || r.Quantity != "40" {
		t.Fatalf("created = %+v", r)
	}

	bare, err := s.Create(ctx, account, validInput(car, 1, "Garage"))
	if err != nil || bare.OdometerReading != nil || bare.Quantity != "40" {
		t.Fatalf("created without optionals = %+v, %v", bare, err)
	}

	got, err := s.Get(ctx, account, r.ID)
	if err != nil || got.ID != r.ID || got.Amount != "120.50" {
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

func TestStoreUpdateTriState(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	other, _ := testCar(t, pool)

	in := validInput(car, 0, "Garage")
	in.OdometerReading = ptr(int64(1000))
	r, err := s.Create(ctx, account, in)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.Update(ctx, account, r.ID, Patch{
		Amount:          ptr("0"),
		OdometerReading: NullableInt{Set: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != "0" || updated.OdometerReading != nil || updated.Quantity != "40" ||
		updated.Station != "Garage" || updated.Fuel != "normal" || !updated.Date.Equal(base) {
		t.Fatalf("updated = %+v", updated)
	}

	updated, err = s.Update(ctx, account, r.ID, Patch{
		Quantity:        ptr("50"),
		Fuel:            ptr("special"),
		OdometerReading: NullableInt{Set: true, Value: ptr(int64(2000))},
	})
	if err != nil || updated.Quantity != "50" || updated.Fuel != "special" || *updated.OdometerReading != 2000 {
		t.Fatalf("second update = %+v, %v", updated, err)
	}

	if _, err := s.Update(ctx, other, r.ID, Patch{Station: ptr("stolen")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update as other account err = %v", err)
	}
}

func TestStoreStationsAreDistinctSortedAndScoped(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	other, otherCar := testCar(t, pool)

	empty, err := s.Stations(ctx, account)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("stations with no refuels = %#v, %v", empty, err)
	}
	for i, station := range []string{"Zeta", "Alpha", "Zeta"} {
		if _, err := s.Create(ctx, account, validInput(car, i, station)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Create(ctx, other, validInput(otherCar, 0, "Other")); err != nil {
		t.Fatal(err)
	}
	stations, err := s.Stations(ctx, account)
	if err != nil || len(stations) != 2 || stations[0] != "Alpha" || stations[1] != "Zeta" {
		t.Fatalf("stations = %v, %v", stations, err)
	}
}

func TestStoreChartReturnsPredecessorsBeforeFrom(t *testing.T) {
	s, pool := testStore(t)
	ctx := context.Background()
	account, car := testCar(t, pool)
	secondCar := addCar(t, pool, account)
	other, otherCar := testCar(t, pool)
	create := func(accountID, carID string, day int, odometer *int64) Refuel {
		t.Helper()
		in := validInput(carID, day, "Station")
		in.OdometerReading = odometer
		r, err := s.Create(ctx, accountID, in)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	create(account, car, 0, ptr(int64(1000)))
	// The latest refuel before the range has no reading, so the one before
	// it is the predecessor.
	create(account, car, 1, nil)
	inRange := create(account, car, 3, ptr(int64(1500)))
	create(account, secondCar, 0, ptr(int64(7000)))
	create(other, otherCar, 1, ptr(int64(9000)))

	from := base.AddDate(0, 0, 2)
	items, previous, err := s.Chart(ctx, account, Filter{From: &from})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != inRange.ID {
		t.Fatalf("items = %v", items)
	}
	if len(previous) != 2 || previous[car] != 1000 || previous[secondCar] != 7000 {
		t.Fatalf("previous = %v", previous)
	}

	_, previous, err = s.Chart(ctx, account, Filter{CarID: &secondCar, From: &from})
	if err != nil || len(previous) != 1 || previous[secondCar] != 7000 {
		t.Fatalf("car-filtered previous = %v, %v", previous, err)
	}
	_, previous, err = s.Chart(ctx, account, Filter{})
	if err != nil || len(previous) != 0 {
		t.Fatalf("unbounded previous = %v, %v", previous, err)
	}
}

func TestStoreSubtypeMustFitCarFuel(t *testing.T) {
	store, pool := testStore(t)
	ctx := context.Background()
	accountID, dieselID := testCar(t, pool)
	var electricID string
	if err := pool.QueryRow(ctx, `INSERT INTO cars (account_id, type, make, name, fuel, first_registration, license_plate)
VALUES ($1, 'Hatchback', 'VW', 'ID.3', 'electric', '2022-01-01', 'W-3') RETURNING id`, accountID).Scan(&electricID); err != nil {
		t.Fatal(err)
	}

	charge := validInput(electricID, 0, "Ionity")
	charge.Fuel = "dc"
	created, err := store.Create(ctx, accountID, charge)
	if err != nil || created.Fuel != "dc" {
		t.Fatalf("create dc charge = %+v, %v", created, err)
	}
	for _, fuel := range []string{"normal", "special"} {
		bad := validInput(electricID, 1, "Shell")
		bad.Fuel = fuel
		if _, err := store.Create(ctx, accountID, bad); !errors.Is(err, ErrFuelMismatch) {
			t.Errorf("create %s on electric car: err = %v, want ErrFuelMismatch", fuel, err)
		}
	}
	for _, carID := range []string{electricID, dieselID} {
		other := validInput(carID, 2, "Somewhere")
		other.Fuel = "other"
		if _, err := store.Create(ctx, accountID, other); err != nil {
			t.Errorf("create other on %s: %v", carID, err)
		}
	}

	refuel, err := store.Create(ctx, accountID, validInput(dieselID, 3, "OMV"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(ctx, accountID, refuel.ID, Patch{Fuel: ptr("home")}); !errors.Is(err, ErrFuelMismatch) {
		t.Fatalf("update diesel refuel to home: err = %v, want ErrFuelMismatch", err)
	}
	unchanged, err := store.Get(ctx, accountID, refuel.ID)
	if err != nil || unchanged.Fuel != "normal" {
		t.Fatalf("refuel after rejected update = %+v, %v", unchanged, err)
	}
	if _, err := store.Update(ctx, accountID, created.ID, Patch{Fuel: ptr("ac")}); err != nil {
		t.Fatalf("update charge to ac: %v", err)
	}

	// Another account's car or refuel is still reported as missing.
	otherAccount, otherCar := testCar(t, pool)
	if _, err := store.Create(ctx, accountID, validInput(otherCar, 4, "X")); !errors.Is(err, ErrNotFound) {
		t.Errorf("create on foreign car: err = %v, want ErrNotFound", err)
	}
	if _, err := store.Update(ctx, otherAccount, refuel.ID, Patch{Fuel: ptr("special")}); !errors.Is(err, ErrNotFound) {
		t.Errorf("update foreign refuel: err = %v, want ErrNotFound", err)
	}
}
