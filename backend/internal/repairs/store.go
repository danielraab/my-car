package repairs

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound reports a repair, or a car to attach one to, that does not
// exist or belongs to another account.
var ErrNotFound = errors.New("repair not found")

// Repair is a stored repair. The amount is a decimal string exactly as it
// appears on the wire; an unrecorded description is empty.
type Repair struct {
	ID              string
	CarID           string
	Date            time.Time
	Station         string
	OdometerReading *int64
	Type            string
	Amount          string
	Description     string
	AttachmentCount int
}

// Input is a validated repair creation request.
type Input struct {
	CarID           string
	Date            time.Time
	Station         string
	OdometerReading *int64
	Type            string
	Amount          string
	Description     string
}

// NullableInt is a patch value for a nullable column: Set false leaves the
// column unchanged, Set true with a nil Value clears it.
type NullableInt struct {
	Set   bool
	Value *int64
}

// Patch is a validated partial update; nil pointers leave columns unchanged.
type Patch struct {
	Date                               *time.Time
	Station, Type, Amount, Description *string
	OdometerReading                    NullableInt
}

// Filter narrows a listing; nil members do not filter.
type Filter struct {
	CarID    *string
	From, To *time.Time
}

// Cursor is the keyset position of the last repair on a page.
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

const columns = `id, car_id, date, station, odometer_reading, type::text, amount::text, COALESCE(description, ''), (SELECT count(*) FROM attachments a WHERE a.repair_id = repairs.id)`

// owned restricts repairs to those on the account in $1's cars; repairs
// have no account column of their own.
const owned = `car_id IN (SELECT id FROM cars WHERE account_id = $1)`

func scanRepair(row pgx.Row) (Repair, error) {
	var r Repair
	err := row.Scan(&r.ID, &r.CarID, &r.Date, &r.Station, &r.OdometerReading, &r.Type, &r.Amount, &r.Description, &r.AttachmentCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Repair{}, ErrNotFound
	}
	return r, err
}

// List returns up to limit of the account's repairs matching the filter
// after the cursor, newest first (by date then ID, both descending), and
// the cursor of the next page if there is one.
func (s *Store) List(ctx context.Context, accountID string, f Filter, after *Cursor, limit int) ([]Repair, *Cursor, error) {
	var afterDate *time.Time
	var afterID *string
	if after != nil {
		afterDate, afterID = &after.Date, &after.ID
	}
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM repairs
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
	items := []Repair{}
	for rows.Next() {
		r, err := scanRepair(rows)
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

// Create inserts a repair on one of the account's cars; a car the account
// does not own inserts nothing and reports ErrNotFound.
func (s *Store) Create(ctx context.Context, accountID string, in Input) (Repair, error) {
	return scanRepair(s.pool.QueryRow(ctx, `INSERT INTO repairs (car_id, date, station, odometer_reading, type, amount, description)
SELECT id, $3, $4, $5, $6::text::repair_type, $7::text::numeric, $8 FROM cars WHERE account_id = $1 AND id = $2
RETURNING `+columns,
		accountID, in.CarID, in.Date, in.Station, in.OdometerReading, in.Type, in.Amount, in.Description))
}

func (s *Store) Get(ctx context.Context, accountID, repairID string) (Repair, error) {
	return scanRepair(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM repairs WHERE `+owned+` AND id = $2`, accountID, repairID))
}

// Update applies a patch in one statement: COALESCE keeps columns that were
// not supplied, and CASE on the Set flag lets the odometer reading be
// cleared as well as kept or replaced.
func (s *Store) Update(ctx context.Context, accountID, repairID string, p Patch) (Repair, error) {
	return scanRepair(s.pool.QueryRow(ctx, `UPDATE repairs SET
  date = COALESCE($3, date),
  station = COALESCE($4, station),
  type = COALESCE($5::text::repair_type, type),
  amount = COALESCE($6::text::numeric, amount),
  description = COALESCE($7, description),
  odometer_reading = CASE WHEN $8::boolean THEN $9 ELSE odometer_reading END
WHERE `+owned+` AND id = $2
RETURNING `+columns,
		accountID, repairID, p.Date, p.Station, p.Type, p.Amount, p.Description, p.OdometerReading.Set, p.OdometerReading.Value))
}

func (s *Store) Delete(ctx context.Context, accountID, repairID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM repairs WHERE `+owned+` AND id = $2`, accountID, repairID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Stations returns the distinct station names of the account's repairs in
// ascending order.
func (s *Store) Stations(ctx context.Context, accountID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT station FROM repairs WHERE `+owned+` ORDER BY station`, accountID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
