package cars

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound reports a car that does not exist or belongs to another account.
var ErrNotFound = errors.New("car not found")

// ErrDeregistrationBeforeFirstRegistration reports a write rejected by the
// database because the deregistration date precedes the first registration.
var ErrDeregistrationBeforeFirstRegistration = errors.New("deregistration date before first registration")

const dateOrderConstraint = "cars_deregistration_after_first_registration"

// Car is a stored car. Dates are RFC 3339 full-dates and the price is a
// decimal string, exactly as they appear on the wire.
type Car struct {
	ID                 string
	Type               string
	Make               string
	Name               string
	Fuel               string
	FirstRegistration  *string
	LicensePlate       *string
	FIN                *string
	IsActive           bool
	PurchaseDate       *string
	PurchasePrice      *string
	DeregistrationDate *string
	Information        *string
	AttachmentCount    int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Input is a validated car creation request.
type Input struct {
	Type, Make, Name, Fuel                                            string
	FirstRegistration, LicensePlate, FIN, PurchaseDate, PurchasePrice *string
	DeregistrationDate, Information                                   *string
	IsActive                                                          bool
}

// Nullable is a patch value for a nullable column: Set false leaves the
// column unchanged, Set true with a nil Value clears it.
type Nullable struct {
	Set   bool
	Value *string
}

// Patch is a validated partial update; nil pointers leave columns unchanged.
type Patch struct {
	Type, Make, Name, Fuel                                            *string
	IsActive                                                          *bool
	FirstRegistration, LicensePlate, FIN, PurchaseDate, PurchasePrice Nullable
	DeregistrationDate, Information                                   Nullable
}

// Cursor is the keyset position of the last car on a page.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID))
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
	createdAt, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return Cursor{}, false
	}
	return Cursor{CreatedAt: createdAt, ID: id}, true
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const columns = `id, type, make, name, fuel::text, to_char(first_registration, 'YYYY-MM-DD'), license_plate, fin, is_active, to_char(purchase_date, 'YYYY-MM-DD'), purchase_price::text, to_char(deregistration_date, 'YYYY-MM-DD'), information, (SELECT count(*) FROM attachments a WHERE a.car_id = cars.id), created_at, updated_at`

func scanCar(row pgx.Row) (Car, error) {
	var c Car
	err := row.Scan(&c.ID, &c.Type, &c.Make, &c.Name, &c.Fuel, &c.FirstRegistration, &c.LicensePlate, &c.FIN, &c.IsActive, &c.PurchaseDate, &c.PurchasePrice, &c.DeregistrationDate, &c.Information, &c.AttachmentCount, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Car{}, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == dateOrderConstraint {
		return Car{}, ErrDeregistrationBeforeFirstRegistration
	}
	return c, err
}

// List returns up to limit of the account's cars after the cursor, ordered
// by creation time then ID, and the cursor of the next page if there is one.
func (s *Store) List(ctx context.Context, accountID string, after *Cursor, limit int) ([]Car, *Cursor, error) {
	var rows pgx.Rows
	var err error
	if after == nil {
		rows, err = s.pool.Query(ctx, `SELECT `+columns+` FROM cars WHERE account_id = $1 ORDER BY created_at, id LIMIT $2`, accountID, limit+1)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT `+columns+` FROM cars WHERE account_id = $1 AND (created_at, id) > ($2, $3::uuid) ORDER BY created_at, id LIMIT $4`, accountID, after.CreatedAt, after.ID, limit+1)
	}
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []Car{}
	for rows.Next() {
		c, err := scanCar(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(items) <= limit {
		return items, nil, nil
	}
	items = items[:limit]
	last := items[limit-1]
	return items, &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
}

func (s *Store) Create(ctx context.Context, accountID string, in Input) (Car, error) {
	return scanCar(s.pool.QueryRow(ctx, `INSERT INTO cars (account_id, type, make, name, fuel, first_registration, license_plate, fin, is_active, purchase_date, purchase_price, deregistration_date, information)
VALUES ($1, $2, $3, $4, $5::text::vehicle_fuel, $6::text::date, $7, $8, $9, $10::text::date, $11::text::numeric, $12::text::date, $13)
RETURNING `+columns,
		accountID, in.Type, in.Make, in.Name, in.Fuel, in.FirstRegistration, in.LicensePlate, in.FIN, in.IsActive, in.PurchaseDate, in.PurchasePrice, in.DeregistrationDate, in.Information))
}

func (s *Store) Get(ctx context.Context, accountID, carID string) (Car, error) {
	return scanCar(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM cars WHERE account_id = $1 AND id = $2`, accountID, carID))
}

// Update applies a patch in one statement: COALESCE keeps non-nullable
// columns that were not supplied, and CASE on each Set flag lets a nullable
// column be cleared as well as kept or replaced.
func (s *Store) Update(ctx context.Context, accountID, carID string, p Patch) (Car, error) {
	return scanCar(s.pool.QueryRow(ctx, `UPDATE cars SET
  type = COALESCE($3, type),
  make = COALESCE($4, make),
  name = COALESCE($5, name),
  fuel = COALESCE($6::text::vehicle_fuel, fuel),
  is_active = COALESCE($7, is_active),
  first_registration = CASE WHEN $8::boolean THEN $9::text::date ELSE first_registration END,
  license_plate = CASE WHEN $10::boolean THEN $11 ELSE license_plate END,
  fin = CASE WHEN $12::boolean THEN $13 ELSE fin END,
  purchase_date = CASE WHEN $14::boolean THEN $15::text::date ELSE purchase_date END,
  purchase_price = CASE WHEN $16::boolean THEN $17::text::numeric ELSE purchase_price END,
  deregistration_date = CASE WHEN $18::boolean THEN $19::text::date ELSE deregistration_date END,
  information = CASE WHEN $20::boolean THEN $21 ELSE information END
WHERE account_id = $1 AND id = $2
RETURNING `+columns,
		accountID, carID, p.Type, p.Make, p.Name, p.Fuel, p.IsActive,
		p.FirstRegistration.Set, p.FirstRegistration.Value, p.LicensePlate.Set, p.LicensePlate.Value,
		p.FIN.Set, p.FIN.Value, p.PurchaseDate.Set, p.PurchaseDate.Value, p.PurchasePrice.Set, p.PurchasePrice.Value,
		p.DeregistrationDate.Set, p.DeregistrationDate.Value, p.Information.Set, p.Information.Value))
}

func (s *Store) Delete(ctx context.Context, accountID, carID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM cars WHERE account_id = $1 AND id = $2`, accountID, carID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
