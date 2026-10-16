package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrPasskeyNotFound  = errors.New("passkey not found")
	ErrDuplicatePasskey = errors.New("passkey already registered")
)

// Ceremony kinds stored with a passkey challenge.
const (
	ceremonyRegistration = "registration"
	ceremonyLogin        = "login"
)

// Passkey is a stored WebAuthn credential and its display metadata.
type Passkey struct {
	ID, AccountID     string
	CredentialID      []byte
	PublicKey         []byte
	AAGUID            []byte
	SignCount         uint32
	Transports        []string
	AttestationFormat string
	UserVerified      bool
	BackupEligible    bool
	BackupState       bool
	Name              string
	CreatedAt         time.Time
	LastUsedAt        *time.Time
}

// PasskeyRepository persists passkey ceremonies and credentials.
type PasskeyRepository interface {
	CreatePasskeyChallenge(ctx context.Context, digest []byte, kind, accountID string, sessionData []byte, expires time.Time) error
	ConsumePasskeyChallenge(ctx context.Context, digest []byte, kind, accountID string, now time.Time) ([]byte, error)
	ListPasskeys(ctx context.Context, accountID string) ([]Passkey, error)
	AccountPasskeys(ctx context.Context, accountID string) (Account, []Passkey, error)
	CreatePasskey(ctx context.Context, p Passkey) (Passkey, error)
	RenamePasskey(ctx context.Context, accountID, passkeyID, name string) (Passkey, error)
	DeletePasskey(ctx context.Context, accountID, passkeyID string, now time.Time) error
	CompletePasskeyLogin(ctx context.Context, credentialID []byte, signCount uint32, backupState bool, now time.Time, sessionDigest []byte, sessionExpires time.Time) error
}

const passkeyColumns = `id,account_id,credential_id,public_key,aaguid,sign_count,transports,attestation_format,user_verified,backup_eligible,backup_state,name,created_at,last_used_at`

func scanPasskey(row pgx.Row) (Passkey, error) {
	var p Passkey
	var signCount int64
	err := row.Scan(&p.ID, &p.AccountID, &p.CredentialID, &p.PublicKey, &p.AAGUID, &signCount, &p.Transports, &p.AttestationFormat, &p.UserVerified, &p.BackupEligible, &p.BackupState, &p.Name, &p.CreatedAt, &p.LastUsedAt)
	p.SignCount = uint32(signCount)
	return p, err
}

// nullableAccount maps the empty account ID of a login ceremony to NULL.
func nullableAccount(accountID string) *string {
	if accountID == "" {
		return nil
	}
	return &accountID
}

// CreatePasskeyChallenge stores the server-side state of a ceremony under the
// digest of its correlation token. Registration ceremonies are bound to
// accountID; login ceremonies pass "".
func (s *Store) CreatePasskeyChallenge(ctx context.Context, digest []byte, kind, accountID string, sessionData []byte, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO webauthn_challenges (ceremony_digest,kind,account_id,session_data,expires_at) VALUES ($1,$2,$3,$4,$5)`, digest, kind, nullableAccount(accountID), sessionData, expires)
	return err
}

// ConsumePasskeyChallenge marks an unexpired, unused ceremony of the given kind
// and account as used and returns its state.
func (s *Store) ConsumePasskeyChallenge(ctx context.Context, digest []byte, kind, accountID string, now time.Time) ([]byte, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, `UPDATE webauthn_challenges SET consumed_at=$4 WHERE ceremony_digest=$1 AND kind=$2 AND account_id IS NOT DISTINCT FROM $3 AND consumed_at IS NULL AND expires_at>$4 RETURNING session_data`, digest, kind, nullableAccount(accountID), now).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredential
	}
	return data, err
}

func (s *Store) ListPasskeys(ctx context.Context, accountID string) ([]Passkey, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+passkeyColumns+` FROM webauthn_credentials WHERE account_id=$1 ORDER BY created_at, id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Passkey{}
	for rows.Next() {
		p, err := scanPasskey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AccountPasskeys loads an account and its passkeys for a login assertion.
func (s *Store) AccountPasskeys(ctx context.Context, accountID string) (Account, []Passkey, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `SELECT `+AccountColumns+` FROM accounts WHERE id=$1`, accountID).Scan(a.ScanTargets()...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, nil, ErrInvalidCredential
	}
	if err != nil {
		return Account{}, nil, err
	}
	passkeys, err := s.ListPasskeys(ctx, accountID)
	return a, passkeys, err
}

func (s *Store) CreatePasskey(ctx context.Context, p Passkey) (Passkey, error) {
	if p.Transports == nil {
		p.Transports = []string{}
	}
	created, err := scanPasskey(s.pool.QueryRow(ctx, `INSERT INTO webauthn_credentials (account_id,credential_id,public_key,aaguid,sign_count,transports,attestation_format,user_verified,backup_eligible,backup_state,name) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+passkeyColumns,
		p.AccountID, p.CredentialID, p.PublicKey, p.AAGUID, int64(p.SignCount), p.Transports, p.AttestationFormat, p.UserVerified, p.BackupEligible, p.BackupState, p.Name))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Passkey{}, ErrDuplicatePasskey
	}
	return created, err
}

func (s *Store) RenamePasskey(ctx context.Context, accountID, passkeyID, name string) (Passkey, error) {
	p, err := scanPasskey(s.pool.QueryRow(ctx, `UPDATE webauthn_credentials SET name=$3 WHERE account_id=$1 AND id=$2 RETURNING `+passkeyColumns, accountID, passkeyID, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return Passkey{}, ErrPasskeyNotFound
	}
	return p, err
}

// DeletePasskey revokes every session established with the passkey and then
// deletes it, atomically.
func (s *Store) DeletePasskey(ctx context.Context, accountID, passkeyID string, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE sessions SET revoked_at=$3 WHERE credential_id=$2 AND account_id=$1 AND revoked_at IS NULL`, accountID, passkeyID, now); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM webauthn_credentials WHERE account_id=$1 AND id=$2`, accountID, passkeyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPasskeyNotFound
	}
	return tx.Commit(ctx)
}

// CompletePasskeyLogin records a verified assertion's counter, backup state
// and use time, and creates a session referencing the passkey, atomically.
func (s *Store) CompletePasskeyLogin(ctx context.Context, credentialID []byte, signCount uint32, backupState bool, now time.Time, sessionDigest []byte, sessionExpires time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var passkeyID, accountID string
	err = tx.QueryRow(ctx, `UPDATE webauthn_credentials SET sign_count=$2, backup_state=$3, last_used_at=$4 WHERE credential_id=$1 RETURNING id,account_id`, credentialID, int64(signCount), backupState, now).Scan(&passkeyID, &accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidCredential
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO sessions (token_digest,account_id,expires_at,credential_id) VALUES ($1,$2,$3,$4)`, sessionDigest, accountID, sessionExpires, passkeyID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
