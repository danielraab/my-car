package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidCredential = errors.New("invalid or expired credential")

// Account is a user account with its profile and display settings. The
// display settings only label values; they never convert stored data.
type Account struct {
	ID, Email, FirstName, LastName     string
	Currency, VolumeUnit, DistanceUnit string
}

// Profile is the JSON representation of an account shared by the session
// and profile operations (OpenAPI `Profile`).
type Profile struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Currency     string `json:"currency"`
	VolumeUnit   string `json:"volumeUnit"`
	DistanceUnit string `json:"distanceUnit"`
}

// Profile returns the account's API representation.
func (a Account) Profile() Profile {
	return Profile{ID: a.ID, Email: a.Email, FirstName: a.FirstName, LastName: a.LastName, Currency: a.Currency, VolumeUnit: a.VolumeUnit, DistanceUnit: a.DistanceUnit}
}

// AccountColumns lists the accounts columns that Scan expects, in order.
const AccountColumns = "id,email,first_name,last_name,currency,volume_unit,distance_unit"

// ScanTargets returns the destinations for AccountColumns.
func (a *Account) ScanTargets() []any {
	return []any{&a.ID, &a.Email, &a.FirstName, &a.LastName, &a.Currency, &a.VolumeUnit, &a.DistanceUnit}
}

type Session struct {
	Account
	ExpiresAt time.Time
	// CreatedAt is when the session was established, i.e. the login time.
	CreatedAt time.Time
	// PasskeyID names the passkey that established the session, or "".
	PasskeyID string
}

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (s *Store) CreateMagicLink(ctx context.Context, digest []byte, email, returnTo string, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO magic_link_challenges (token_digest,email,return_to,expires_at) VALUES ($1,$2,$3,$4)`, digest, normalizeEmail(email), returnTo, expires)
	return err
}

func (s *Store) DeleteMagicLink(ctx context.Context, digest []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM magic_link_challenges WHERE token_digest=$1`, digest)
	return err
}

// CreateOIDCAttempt saves the one-time server-side state required to complete
// an authorization-code login. stateDigest is deliberately the only state
// representation retained by the database.
func (s *Store) CreateOIDCAttempt(ctx context.Context, stateDigest []byte, nonce, verifier, returnTo string, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO oidc_login_attempts (state_digest,nonce,pkce_verifier,return_to,expires_at) VALUES ($1,$2,$3,$4,$5)`, stateDigest, nonce, verifier, returnTo, expires)
	return err
}

type OIDCAttempt struct{ Nonce, PKCEVerifier, ReturnTo string }

func (s *Store) ConsumeOIDCAttempt(ctx context.Context, stateDigest []byte, now time.Time) (OIDCAttempt, error) {
	var out OIDCAttempt
	err := s.pool.QueryRow(ctx, `UPDATE oidc_login_attempts SET consumed_at=$2 WHERE state_digest=$1 AND consumed_at IS NULL AND expires_at>$2 RETURNING nonce,pkce_verifier,return_to`, stateDigest, now).Scan(&out.Nonce, &out.PKCEVerifier, &out.ReturnTo)
	if errors.Is(err, pgx.ErrNoRows) {
		return OIDCAttempt{}, ErrInvalidCredential
	}
	return out, err
}

func (s *Store) ResolveOIDCIdentity(ctx context.Context, issuer, subject, email string) (Account, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, err
	}
	defer tx.Rollback(ctx)
	var existing string
	err = tx.QueryRow(ctx, `SELECT account_id FROM oidc_identities WHERE issuer=$1 AND subject=$2`, issuer, subject).Scan(&existing)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Account{}, err
	}
	var a Account
	err = tx.QueryRow(ctx, `INSERT INTO accounts (email) VALUES ($1) ON CONFLICT (email) DO UPDATE SET email=EXCLUDED.email RETURNING `+AccountColumns, normalizeEmail(email)).Scan(a.ScanTargets()...)
	if err != nil {
		return Account{}, err
	}
	if existing != "" && existing != a.ID {
		return Account{}, errors.New("OIDC identity belongs to another account")
	}
	if existing == "" {
		if _, err = tx.Exec(ctx, `INSERT INTO oidc_identities (issuer,subject,account_id) VALUES ($1,$2,$3)`, issuer, subject, a.ID); err != nil {
			return Account{}, err
		}
	}
	return a, tx.Commit(ctx)
}

type MagicLinkLogin struct {
	Account
	ReturnTo string
}

func (s *Store) ConsumeMagicLink(ctx context.Context, digest []byte, now time.Time) (MagicLinkLogin, error) {
	return s.consumeMagicLink(ctx, digest, nil, now, time.Time{})
}

func (s *Store) ConsumeMagicLinkAndCreateSession(ctx context.Context, digest, sessionDigest []byte, now, sessionExpires time.Time) (MagicLinkLogin, error) {
	return s.consumeMagicLink(ctx, digest, sessionDigest, now, sessionExpires)
}

func (s *Store) consumeMagicLink(ctx context.Context, digest, sessionDigest []byte, now, sessionExpires time.Time) (MagicLinkLogin, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return MagicLinkLogin{}, err
	}
	defer tx.Rollback(ctx)
	var email, returnTo string
	err = tx.QueryRow(ctx, `UPDATE magic_link_challenges SET consumed_at=$2 WHERE token_digest=$1 AND consumed_at IS NULL AND expires_at>$2 RETURNING email,return_to`, digest, now).Scan(&email, &returnTo)
	if errors.Is(err, pgx.ErrNoRows) {
		return MagicLinkLogin{}, ErrInvalidCredential
	}
	if err != nil {
		return MagicLinkLogin{}, err
	}
	var a Account
	err = tx.QueryRow(ctx, `INSERT INTO accounts (email) VALUES ($1) ON CONFLICT (email) DO UPDATE SET email=EXCLUDED.email RETURNING `+AccountColumns, email).Scan(a.ScanTargets()...)
	if err != nil {
		return MagicLinkLogin{}, err
	}
	if sessionDigest != nil {
		if _, err = tx.Exec(ctx, `INSERT INTO sessions (token_digest,account_id,expires_at) VALUES ($1,$2,$3)`, sessionDigest, a.ID, sessionExpires); err != nil {
			return MagicLinkLogin{}, err
		}
	}
	return MagicLinkLogin{Account: a, ReturnTo: returnTo}, tx.Commit(ctx)
}

func (s *Store) CreateSession(ctx context.Context, digest []byte, accountID string, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO sessions (token_digest,account_id,expires_at) VALUES ($1,$2,$3)`, digest, accountID, expires)
	return err
}
func (s *Store) Session(ctx context.Context, digest []byte, now time.Time) (Session, error) {
	var out Session
	err := s.pool.QueryRow(ctx, `SELECT a.id,a.email,a.first_name,a.last_name,a.currency,a.volume_unit,a.distance_unit,s.expires_at,s.created_at,coalesce(s.credential_id::text,'') FROM sessions s JOIN accounts a ON a.id=s.account_id WHERE s.token_digest=$1 AND s.revoked_at IS NULL AND s.expires_at>$2`, digest, now).Scan(append(out.Account.ScanTargets(), &out.ExpiresAt, &out.CreatedAt, &out.PasskeyID)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrInvalidCredential
	}
	return out, err
}
func (s *Store) RevokeSession(ctx context.Context, digest []byte, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=$2 WHERE token_digest=$1 AND revoked_at IS NULL`, digest, now)
	return err
}

func (s *Store) Prune(ctx context.Context, now time.Time) error {
	for _, q := range []string{`DELETE FROM oidc_login_attempts WHERE expires_at <= $1`, `DELETE FROM magic_link_challenges WHERE expires_at <= $1`, `DELETE FROM webauthn_challenges WHERE expires_at <= $1`, `DELETE FROM sessions WHERE expires_at <= $1 OR revoked_at IS NOT NULL`} {
		if _, err := s.pool.Exec(ctx, q, now); err != nil {
			return err
		}
	}
	return nil
}
