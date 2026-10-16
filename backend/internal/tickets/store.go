package tickets

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound reports a ticket, or a car to attach one to, that does not
// exist or belongs to another account.
var ErrNotFound = errors.New("ticket not found")

// Ticket is a stored ticket. The amount is a decimal string exactly as it
// appears on the wire; an unrecorded description is empty.
type Ticket struct {
	ID              string
	CarID           string
	Date            time.Time
	Type            string
	Location        string
	Amount          string
	Description     string
	AttachmentCount int
}

// Input is a validated ticket creation request.
type Input struct {
	CarID       string
	Date        time.Time
	Type        string
	Location    string
	Amount      string
	Description string
}

// Patch is a validated partial update; nil pointers leave columns unchanged.
type Patch struct {
	Date                                *time.Time
	Type, Location, Amount, Description *string
}

// Filter narrows a listing; nil members do not filter.
type Filter struct {
	CarID    *string
	From, To *time.Time
}

// Cursor is the keyset position of the last ticket on a page.
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

const columns = `id, car_id, date, type::text, location, amount::text, COALESCE(description, ''), (SELECT count(*) FROM attachments a WHERE a.ticket_id = tickets.id)`

// owned restricts tickets to those on the account in $1's cars; tickets
// have no account column of their own.
const owned = `car_id IN (SELECT id FROM cars WHERE account_id = $1)`

func scanTicket(row pgx.Row) (Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.CarID, &t.Date, &t.Type, &t.Location, &t.Amount, &t.Description, &t.AttachmentCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ticket{}, ErrNotFound
	}
	return t, err
}

// List returns up to limit of the account's tickets matching the filter
// after the cursor, newest first (by date then ID, both descending), and
// the cursor of the next page if there is one.
func (s *Store) List(ctx context.Context, accountID string, f Filter, after *Cursor, limit int) ([]Ticket, *Cursor, error) {
	var afterDate *time.Time
	var afterID *string
	if after != nil {
		afterDate, afterID = &after.Date, &after.ID
	}
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM tickets
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
	items := []Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, t)
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

// Create inserts a ticket on one of the account's cars; a car the account
// does not own inserts nothing and reports ErrNotFound.
func (s *Store) Create(ctx context.Context, accountID string, in Input) (Ticket, error) {
	return scanTicket(s.pool.QueryRow(ctx, `INSERT INTO tickets (car_id, date, type, location, amount, description)
SELECT id, $3, $4::text::ticket_type, $5, $6::text::numeric, $7 FROM cars WHERE account_id = $1 AND id = $2
RETURNING `+columns,
		accountID, in.CarID, in.Date, in.Type, in.Location, in.Amount, in.Description))
}

func (s *Store) Get(ctx context.Context, accountID, ticketID string) (Ticket, error) {
	return scanTicket(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM tickets WHERE `+owned+` AND id = $2`, accountID, ticketID))
}

// Update applies a patch in one statement; COALESCE keeps columns that were
// not supplied.
func (s *Store) Update(ctx context.Context, accountID, ticketID string, p Patch) (Ticket, error) {
	return scanTicket(s.pool.QueryRow(ctx, `UPDATE tickets SET
  date = COALESCE($3, date),
  type = COALESCE($4::text::ticket_type, type),
  location = COALESCE($5, location),
  amount = COALESCE($6::text::numeric, amount),
  description = COALESCE($7, description)
WHERE `+owned+` AND id = $2
RETURNING `+columns,
		accountID, ticketID, p.Date, p.Type, p.Location, p.Amount, p.Description))
}

func (s *Store) Delete(ctx context.Context, accountID, ticketID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM tickets WHERE `+owned+` AND id = $2`, accountID, ticketID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Locations returns the distinct locations of the account's tickets in
// ascending order.
func (s *Store) Locations(ctx context.Context, accountID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT location FROM tickets WHERE `+owned+` ORDER BY location`, accountID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
