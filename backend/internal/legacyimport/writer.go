package legacyimport

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFresh reports a target database that already holds application data;
// the import runs once, into an empty database.
var ErrNotFresh = errors.New("database is not fresh")

// Counts is the number of rows written per table.
type Counts struct {
	Accounts, Cars, Refuels, Repairs, Tickets int
}

// guardedTables must all be empty before the import writes anything.
var guardedTables = []string{"accounts", "cars", "refuels", "repairs", "tickets"}

// Write inserts the plan in one transaction, after checking under a lock
// that the target tables are empty. A failure leaves the database unchanged.
func Write(ctx context.Context, pool *pgxpool.Pool, plan Plan) (Counts, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Counts{}, fmt.Errorf("begin import: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `LOCK TABLE accounts, cars, refuels, repairs, tickets IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return Counts{}, fmt.Errorf("lock target tables: %w", err)
	}
	for _, table := range guardedTables {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+`)`).Scan(&exists); err != nil {
			return Counts{}, fmt.Errorf("check %s: %w", table, err)
		}
		if exists {
			return Counts{}, fmt.Errorf("%w: %s has rows", ErrNotFresh, table)
		}
	}

	// Parents are queued before children, and a batch runs in order.
	batch := &pgx.Batch{}
	for _, a := range plan.Accounts {
		batch.Queue(`INSERT INTO accounts (id, email, first_name, last_name, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			a.ID, a.Email, a.FirstName, a.LastName, a.CreatedAt, a.UpdatedAt)
	}
	for _, c := range plan.Cars {
		batch.Queue(`INSERT INTO cars (id, account_id, type, make, name, fuel, first_registration, license_plate, fin, is_active, purchase_date, purchase_price, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6::text::vehicle_fuel, $7::text::date, $8, $9, $10, $11::text::date, $12::text::numeric, $13, $14)`,
			c.ID, c.AccountID, c.Type, c.Make, c.Name, c.Fuel, c.FirstRegistration, c.LicensePlate, c.FIN, c.IsActive, c.PurchaseDate, c.PurchasePrice, c.CreatedAt, c.UpdatedAt)
	}
	for _, r := range plan.Refuels {
		batch.Queue(`INSERT INTO refuels (id, car_id, date, station, odometer_reading, fuel, quantity, amount, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5::text::bigint, $6::text::refuel_fuel, $7::text::numeric, $8::text::numeric, $9, $10)`,
			r.ID, r.CarID, r.Date, r.Station, r.OdometerReading, r.Fuel, r.Quantity, r.Amount, r.CreatedAt, r.UpdatedAt)
	}
	for _, r := range plan.Repairs {
		batch.Queue(`INSERT INTO repairs (id, car_id, date, station, odometer_reading, type, amount, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5::text::bigint, $6::text::repair_type, $7::text::numeric, $8, $9, $10)`,
			r.ID, r.CarID, r.Date, r.Station, r.OdometerReading, r.Type, r.Amount, r.Description, r.CreatedAt, r.UpdatedAt)
	}
	for _, t := range plan.Tickets {
		batch.Queue(`INSERT INTO tickets (id, car_id, date, type, location, amount, description, created_at, updated_at)
VALUES ($1, $2, $3, $4::text::ticket_type, $5, $6::text::numeric, $7, $8, $9)`,
			t.ID, t.CarID, t.Date, t.Type, t.Location, t.Amount, t.Description, t.CreatedAt, t.UpdatedAt)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return Counts{}, fmt.Errorf("insert imported rows: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Counts{}, fmt.Errorf("commit import: %w", err)
	}
	return Counts{
		Accounts: len(plan.Accounts), Cars: len(plan.Cars), Refuels: len(plan.Refuels),
		Repairs: len(plan.Repairs), Tickets: len(plan.Tickets),
	}, nil
}
