package attachments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxFilesPerEntry is the number of attachments one entry can hold.
const MaxFilesPerEntry = 10

var (
	// ErrNotFound reports an entry or attachment that does not exist or
	// belongs to another account.
	ErrNotFound = errors.New("attachment or entry not found")
	// ErrLimitReached reports an entry that already holds MaxFilesPerEntry
	// attachments.
	ErrLimitReached = errors.New("attachment limit reached")
)

// OwnerKind is the type of entry an attachment belongs to.
type OwnerKind int

const (
	OwnerCar OwnerKind = iota
	OwnerRefuel
	OwnerRepair
	OwnerTicket
)

// ownerColumns names each kind's foreign key column in attachments.
var ownerColumns = map[OwnerKind]string{OwnerCar: "car_id", OwnerRefuel: "refuel_id", OwnerRepair: "repair_id", OwnerTicket: "ticket_id"}

// ownerQueries select an entry by ID and owning account ($1 entry, $2
// account), returning the account ID; "FOR UPDATE OF e" is appended to lock it.
var ownerQueries = map[OwnerKind]string{
	OwnerCar:    `SELECT e.account_id FROM cars e WHERE e.id = $1 AND e.account_id = $2`,
	OwnerRefuel: `SELECT c.account_id FROM refuels e JOIN cars c ON c.id = e.car_id WHERE e.id = $1 AND c.account_id = $2`,
	OwnerRepair: `SELECT c.account_id FROM repairs e JOIN cars c ON c.id = e.car_id WHERE e.id = $1 AND c.account_id = $2`,
	OwnerTicket: `SELECT c.account_id FROM tickets e JOIN cars c ON c.id = e.car_id WHERE e.id = $1 AND c.account_id = $2`,
}

// Owner identifies the entry an attachment belongs to.
type Owner struct {
	Kind OwnerKind
	ID   string
}

// Attachment is a stored attachment's metadata.
type Attachment struct {
	ID          string
	Filename    string
	ContentType string
	Size        int64
	StorageKey  string
	CreatedAt   time.Time
}

// NewAttachment is a validated upload ready to be stored.
type NewAttachment struct {
	Filename    string
	ContentType string
	Size        int64
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

const columns = `id, filename, content_type, size_bytes, storage_key, created_at`

func scanAttachment(row pgx.Row) (Attachment, error) {
	var a Attachment
	err := row.Scan(&a.ID, &a.Filename, &a.ContentType, &a.Size, &a.StorageKey, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	return a, err
}

// OwnerExists reports whether the account owns the entry, as ErrNotFound.
func (s *Store) OwnerExists(ctx context.Context, accountID string, owner Owner) error {
	var id string
	err := s.pool.QueryRow(ctx, ownerQueries[owner.Kind], owner.ID, accountID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// List returns the entry's attachments, oldest first.
func (s *Store) List(ctx context.Context, accountID string, owner Owner) ([]Attachment, error) {
	if err := s.OwnerExists(ctx, accountID, owner); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+columns+` FROM attachments WHERE account_id = $1 AND `+ownerColumns[owner.Kind]+` = $2 ORDER BY created_at, id`, accountID, owner.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Attachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// Create stores an attachment in one transaction: it locks the owning entry
// (so concurrent uploads cannot exceed MaxFilesPerEntry), checks the count,
// calls put with the new storage key, and inserts the record. The account is
// taken from the entry, never from the request.
//
// orphan is the key of a file put stored that is confirmed to have no record
// and must be removed by the caller; it is empty when nothing needs removing,
// including when the outcome is unknown, so a record never loses its file.
func (s *Store) Create(ctx context.Context, accountID string, owner Owner, in NewAttachment, put func(key string) error) (a Attachment, orphan string, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Attachment{}, "", err
	}
	defer tx.Rollback(ctx)
	var ownerAccount string
	err = tx.QueryRow(ctx, ownerQueries[owner.Kind]+` FOR UPDATE OF e`, owner.ID, accountID).Scan(&ownerAccount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, "", ErrNotFound
	}
	if err != nil {
		return Attachment{}, "", err
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM attachments WHERE `+ownerColumns[owner.Kind]+` = $1`, owner.ID).Scan(&count); err != nil {
		return Attachment{}, "", err
	}
	if count >= MaxFilesPerEntry {
		return Attachment{}, "", ErrLimitReached
	}
	id := uuid.NewString()
	key := storageKey(ownerAccount, id)
	if err := put(key); err != nil {
		// A failed put may still have left content behind.
		return Attachment{}, key, err
	}
	a, err = scanAttachment(tx.QueryRow(ctx, `INSERT INTO attachments (id, account_id, `+ownerColumns[owner.Kind]+`, storage_key, filename, content_type, size_bytes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING `+columns,
		id, ownerAccount, owner.ID, key, in.Filename, in.ContentType, in.Size))
	if err != nil {
		return Attachment{}, key, err
	}
	if err := tx.Commit(ctx); err != nil {
		var exists bool
		if checkErr := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM attachments WHERE storage_key = $1)`, key).Scan(&exists); checkErr != nil {
			return Attachment{}, "", err
		}
		if exists {
			return a, "", nil
		}
		return Attachment{}, key, err
	}
	return a, "", nil
}

// Get returns one of the account's attachments.
func (s *Store) Get(ctx context.Context, accountID, attachmentID string) (Attachment, error) {
	return scanAttachment(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM attachments WHERE id = $1 AND account_id = $2`, attachmentID, accountID))
}

// Delete removes one of the account's attachments; a database trigger queues
// its stored file for removal (see PendingFileDeletions).
func (s *Store) Delete(ctx context.Context, accountID, attachmentID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM attachments WHERE id = $1 AND account_id = $2`, attachmentID, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PendingFileDeletions returns up to limit storage keys whose record is gone,
// oldest first.
func (s *Store) PendingFileDeletions(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT storage_key FROM attachment_file_deletions ORDER BY queued_at, storage_key LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// FileDeleted removes a storage key from the deletion queue once its file is
// gone.
func (s *Store) FileDeleted(ctx context.Context, key string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM attachment_file_deletions WHERE storage_key = $1`, key)
	return err
}
