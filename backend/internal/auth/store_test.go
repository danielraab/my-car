package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"at.draab/my-car/internal/db"
)

func testStore(t *testing.T) (*Store, func()) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test")
	}
	if err := db.RunMigrations(url); err != nil {
		t.Fatal(err)
	}
	pool, err := db.NewPool(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := pool.Acquire(context.Background())
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	if _, err = lock.Exec(context.Background(), `SELECT pg_advisory_lock(7242026)`); err != nil {
		lock.Release()
		pool.Close()
		t.Fatal(err)
	}
	return NewStore(pool), func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM magic_link_challenges WHERE email LIKE 'auth-test-%'`)
		pool.Exec(ctx, `DELETE FROM sessions WHERE account_id IN (SELECT id FROM accounts WHERE email LIKE 'auth-test-%')`)
		pool.Exec(ctx, `DELETE FROM webauthn_challenges WHERE account_id IN (SELECT id FROM accounts WHERE email LIKE 'auth-test-%') OR session_data->>'test' = 'auth-test'`)
		pool.Exec(ctx, `DELETE FROM webauthn_credentials WHERE account_id IN (SELECT id FROM accounts WHERE email LIKE 'auth-test-%')`)
		pool.Exec(ctx, `DELETE FROM oidc_identities WHERE account_id IN (SELECT id FROM accounts WHERE email LIKE 'auth-test-%')`)
		pool.Exec(ctx, `DELETE FROM accounts WHERE email LIKE 'auth-test-%'`)
		lock.Exec(ctx, `SELECT pg_advisory_unlock(7242026)`)
		lock.Release()
		pool.Close()
	}
}

func uniqueTestEmail() string {
	return "auth-test-" + time.Now().Format("150405.000000000") + "@example.com"
}

func TestStoreMagicLinkReplayAndIdentityConvergence(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now()
	email := uniqueTestEmail()
	magic := Digest("magic-" + email)
	if err := s.CreateMagicLink(ctx, magic, " "+strings.ToUpper(email)+" ", "/", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	sessionDigest := Digest("magic-session-" + email)
	a, err := s.ConsumeMagicLinkAndCreateSession(ctx, magic, sessionDigest, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if a.Email != email || a.Currency != "EUR" || a.VolumeUnit != "liter" || a.DistanceUnit != "kilometer" {
		t.Fatalf("account=%+v", a)
	}
	session, err := s.Session(ctx, sessionDigest, now)
	if err != nil {
		t.Fatalf("atomic session: %v", err)
	}
	if session.Currency != "EUR" || session.VolumeUnit != "liter" || session.DistanceUnit != "kilometer" {
		t.Fatalf("session account=%+v", session.Account)
	}
	if _, err = s.ConsumeMagicLink(ctx, magic, now); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("replay error=%v", err)
	}
	oidc, err := s.ResolveOIDCIdentity(ctx, "https://issuer.example", email, strings.ToUpper(email))
	if err != nil {
		t.Fatal(err)
	}
	if oidc.ID != a.ID {
		t.Fatalf("OIDC account %s != magic account %s", oidc.ID, a.ID)
	}
}

func TestStoreOIDCReplaySessionsAndPrune(t *testing.T) {
	s, cleanup := testStore(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Now()
	email := uniqueTestEmail()
	state := Digest("state-" + email)
	if err := s.CreateOIDCAttempt(ctx, state, "nonce", "verifier", "/cars", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	a, err := s.ConsumeOIDCAttempt(ctx, state, now)
	if err != nil {
		t.Fatal(err)
	}
	if a.ReturnTo != "/cars" {
		t.Fatalf("returnTo=%q", a.ReturnTo)
	}
	if _, err = s.ConsumeOIDCAttempt(ctx, state, now); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("replay error=%v", err)
	}
	account, err := s.ResolveOIDCIdentity(ctx, "https://issuer.example", email, email)
	if err != nil {
		t.Fatal(err)
	}
	session := Digest("session-" + email)
	if err = s.CreateSession(ctx, session, account.ID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, session, now); err != nil {
		t.Fatal(err)
	}
	if err = s.Prune(ctx, now); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, session, now); err != nil {
		t.Fatal("pruned active session")
	}
	if err = s.RevokeSession(ctx, session, now); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Session(ctx, session, now); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("revoked session error=%v", err)
	}
}
