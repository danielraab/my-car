package refuels

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound reports a refuel, or a car to attach one to, that does not
// exist or belongs to another account.
var ErrNotFound = errors.New("refuel not found")

// ErrFuelMismatch reports a subtype that does not fit the car's fuel.
var ErrFuelMismatch = errors.New("refuel subtype does not fit the car's fuel")

// Refuel is a stored refuel. The amount is a decimal string exactly as it
// appears on the wire; an unrecorded description is empty.
type Refuel struct {
	ID              string
	CarID           string
	Date            time.Time
	Station         string
	OdometerReading *int64
	Fuel            string
	Quantity        string
	Amount          string
	AttachmentCount int
}

// Input is a validated refuel creation request.
type Input struct {
	CarID           string
	Date            time.Time
	Station         string
	OdometerReading *int64
	Fuel            string
	Quantity        string
	Amount          string
}

// NullableInt is a patch value for a nullable column: Set false leaves the
// column unchanged, Set true with a nil Value clears it.
type NullableInt struct {
	Set   bool
	Value *int64
}

// Patch is a validated partial update; nil pointers leave columns unchanged.
type Patch struct {
	Date                            *time.Time
	Station, Fuel, Quantity, Amount *string
	OdometerReading                 NullableInt
}

// Filter narrows a listing; nil members do not filter.
type Filter struct {
	CarID    *string
	From, To *time.Time
}

// Cursor is the keyset position of the last refuel on a page.
type Cursor struct {
	Date time.Time
	ID   string
}

func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.Date.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
}

// DecodeCursor parses a cursor produced by Encode.
func DecodeCursor(s string) (Cursor, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, false
	}
	ts, id, ok := strings.Cut(string(raw), "|")
	if !ok || !isUUID(id) {
		return Cursor{}, false
	}
	date, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return Cursor{}, false
	}
	return Cursor{Date: date, ID: id}, true
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const columns = `id, car_id, date, station, odometer_reading, fuel::text, quantity::text, amount::text, (SELECT count(*) FROM attachments a WHERE a.refuel_id = refuels.id)`

// owned restricts refuels to those on the account in $1's cars; refuels
// have no account column of their own.
const owned = `car_id IN (SELECT id FROM cars WHERE account_id = $1)`

func scanRefuel(row pgx.Row) (Refuel, error) {
	var r Refuel
	err := row.Scan(&r.ID, &r.CarID, &r.Date, &r.Station, &r.OdometerReading, &r.Fuel, &r.Quantity, &r.Amount, &r.AttachmentCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Refuel{}, ErrNotFound
	}
	return r, err
}

// List returns up to limit of the account's refuels matching the filter
// after the cursor, newest first (by date then ID, both descending), and
// the cursor of the next page if there is one.
func (s *Store) List(ctx context.Context, accountID string, f Filter, after *Cursor, limit int) ([]Refuel, *Cursor, error) {
	var afterDate *time.Time
	var afterID *string
	if after != nil {
		afterDate, afterID = &after.Date, &after.ID
	}
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM refuels
WHERE `+owned+`
  AND ($2::uuid IS NULL OR car_id = $2::uuid)
  AND ($3::timestamptz IS NULL OR date >= $3::timestamptz)
  AND ($4::timestamptz IS NULL OR date < $4::timestamptz)
  AND ($5::timestamptz IS NULL OR (date, id) < ($5::timestamptz, $6::uuid))
ORDER BY date DESC, id DESC
LIMIT $7`, accountID, f.CarID, f.From, f.To, afterDate, afterID, limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []Refuel{}
	for rows.Next() {
		r, err := scanRefuel(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(items) <= limit {
		return items, nil, nil
	}
	items = items[:limit]
	last := items[limit-1]
	return items, &Cursor{Date: last.Date, ID: last.ID}, nil
}

// Create inserts a refuel on one of the account's cars; a car the account
// does not own inserts nothing and reports ErrNotFound, and a subtype that
// does not fit the car's fuel reports ErrFuelMismatch. The car row is
// share-locked, so a concurrent fuel change cannot slip in between the
// check and the insert.
func (s *Store) Create(ctx context.Context, accountID string, in Input) (Refuel, error) {
	var created Refuel
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := checkCarFuel(ctx, tx, `SELECT fuel::text FROM cars WHERE account_id = $1 AND id = $2 FOR SHARE`, accountID, in.CarID, in.Fuel); err != nil {
			return err
		}
		var err error
		created, err = scanRefuel(tx.QueryRow(ctx, `INSERT INTO refuels (car_id, date, station, odometer_reading, fuel, quantity, amount)
VALUES ($1, $2, $3, $4, $5::text::refuel_fuel, $6::text::numeric, $7::text::numeric)
RETURNING `+columns,
			in.CarID, in.Date, in.Station, in.OdometerReading, in.Fuel, in.Quantity, in.Amount))
		return err
	})
	return created, err
}

// checkCarFuel reads a car's fuel with query (parameters accountID and id)
// and reports ErrNotFound when there is no such car, or ErrFuelMismatch
// when subtype does not fit it.
func checkCarFuel(ctx context.Context, tx pgx.Tx, query, accountID, id, subtype string) error {
	var carFuel string
	if err := tx.QueryRow(ctx, query, accountID, id).Scan(&carFuel); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if !FitsCarFuel(carFuel, subtype) {
		return ErrFuelMismatch
	}
	return nil
}

func (s *Store) Get(ctx context.Context, accountID, refuelID string) (Refuel, error) {
	return scanRefuel(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM refuels WHERE `+owned+` AND id = $2`, accountID, refuelID))
}

// Update applies a patch in one statement: COALESCE keeps columns that were
// not supplied, and CASE on the Set flag lets the odometer reading be
// cleared as well as kept or replaced. A supplied subtype is first checked
// against the refuel's car's fuel, with the car row share-locked as in
// Create.
func (s *Store) Update(ctx context.Context, accountID, refuelID string, p Patch) (Refuel, error) {
	var updated Refuel
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if p.Fuel != nil {
			if err := checkCarFuel(ctx, tx, `SELECT c.fuel::text FROM cars c JOIN refuels r ON r.car_id = c.id
WHERE c.account_id = $1 AND r.id = $2 FOR SHARE OF c`, accountID, refuelID, *p.Fuel); err != nil {
				return err
			}
		}
		var err error
		updated, err = scanRefuel(tx.QueryRow(ctx, `UPDATE refuels SET
  date = COALESCE($3, date),
  station = COALESCE($4, station),
  fuel = COALESCE($5::text::refuel_fuel, fuel),
  quantity = COALESCE($6::text::numeric, quantity),
  amount = COALESCE($7::text::numeric, amount),
  odometer_reading = CASE WHEN $8::boolean THEN $9 ELSE odometer_reading END
WHERE `+owned+` AND id = $2
RETURNING `+columns,
			accountID, refuelID, p.Date, p.Station, p.Fuel, p.Quantity, p.Amount, p.OdometerReading.Set, p.OdometerReading.Value))
		return err
	})
	return updated, err
}

// Chart returns every matching refuel in deterministic order, and for each
// car the last odometer reading dated before f.From, so the first refuel in
// the range can be measured against its actual predecessor.
func (s *Store) Chart(ctx context.Context, accountID string, f Filter) ([]Refuel, map[string]int64, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM refuels WHERE `+owned+`
  AND ($2::uuid IS NULL OR car_id = $2::uuid)
  AND ($3::timestamptz IS NULL OR date >= $3)
  AND ($4::timestamptz IS NULL OR date < $4)
ORDER BY date, id`, accountID, f.CarID, f.From, f.To)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []Refuel{}
	for rows.Next() {
		item, err := scanRefuel(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	previous := map[string]int64{}
	if f.From == nil {
		return items, previous, nil
	}
	rows, err = s.pool.Query(ctx, `SELECT DISTINCT ON (car_id) car_id, odometer_reading FROM refuels
WHERE `+owned+`
  AND ($2::uuid IS NULL OR car_id = $2::uuid)
  AND date < $3 AND odometer_reading IS NOT NULL
ORDER BY car_id, date DESC, id DESC`, accountID, f.CarID, f.From)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var carID string
		var reading int64
		if err := rows.Scan(&carID, &reading); err != nil {
			return nil, nil, err
		}
		previous[carID] = reading
	}
	return items, previous, rows.Err()
}
func (s *Store) Delete(ctx context.Context, accountID, refuelID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM refuels WHERE `+owned+` AND id = $2`, accountID, refuelID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Stations returns the distinct station names of the account's refuels in
// ascending order.
func (s *Store) Stations(ctx context.Context, accountID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT station FROM refuels WHERE `+owned+` ORDER BY station`, accountID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
