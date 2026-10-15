package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"golang.org/x/oauth2"

	"at.draab/my-car/internal/httpserver"
)

// sessionLifetime is how long a browser session stays valid after login.
const sessionLifetime = 30 * 24 * time.Hour

type Repository interface {
	CreateMagicLink(context.Context, []byte, string, string, time.Time) error
	DeleteMagicLink(context.Context, []byte) error
	ConsumeMagicLinkAndCreateSession(context.Context, []byte, []byte, time.Time, time.Time) (MagicLinkLogin, error)
	CreateOIDCAttempt(context.Context, []byte, string, string, string, time.Time) error
	ConsumeOIDCAttempt(context.Context, []byte, time.Time) (OIDCAttempt, error)
	ResolveOIDCIdentity(context.Context, string, string, string) (Account, error)
	CreateSession(context.Context, []byte, string, time.Time) error
	Session(context.Context, []byte, time.Time) (Session, error)
	RevokeSession(context.Context, []byte, time.Time) error
	PasskeyRepository
}

type Service struct {
	store   Repository
	mailer  Mailer
	oidc    OIDCProvider
	rp      *webauthn.WebAuthn
	limits  *rateLimits
	baseURL string
	now     func() time.Time
}

type sessionContextKey struct{}
type authenticatedSession struct {
	Session
	digest []byte
}

// NewService wires the authentication routes. Passkeys are always enabled,
// with the relying party derived from baseURL, which must be an http(s)
// origin as validated by configuration loading; anything else panics.
func NewService(store Repository, mailer Mailer, oidc OIDCProvider, baseURL string) *Service {
	rp, err := NewRelyingParty(baseURL)
	if err != nil {
		panic(err)
	}
	return &Service{store: store, mailer: mailer, oidc: oidc, rp: rp, limits: newRateLimits(), baseURL: strings.TrimRight(baseURL, "/"), now: time.Now}
}

func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/methods", s.getAuthenticationMethods)
	mux.Handle("POST /api/v1/auth/magic-links", s.limit(groupMagicLinkRequest, limitJSON, s.requestMagicLink))
	mux.Handle("GET /api/v1/auth/magic-links/{token}", s.limit(groupLoginRedirect, limitRedirect, s.consumeMagicLink))
	if s.oidc != nil {
		mux.Handle("GET /api/v1/auth/oidc/start", s.limit(groupLoginRedirect, limitRedirect, s.startOIDC))
		mux.Handle("GET /api/v1/auth/oidc/callback", s.limit(groupLoginRedirect, limitRedirect, s.callbackOIDC))
	}
	s.registerPasskeyRoutes(mux)
	mux.Handle("GET /api/v1/session", s.RequireSession(http.HandlerFunc(s.getSession)))
	mux.Handle("DELETE /api/v1/session", s.RequireSession(http.HandlerFunc(s.logout)))
}

func (s *Service) getAuthenticationMethods(w http.ResponseWriter, _ *http.Request) {
	methods := []string{"magic_link", "passkey"}
	if s.oidc != nil {
		methods = append(methods, "oidc")
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]string{"methods": methods})
}

func validEmail(raw string) (string, bool) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil || parsed.Address != strings.TrimSpace(raw) {
		return "", false
	}
	parts := strings.Split(parsed.Address, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return normalizeEmail(parsed.Address), true
}
func validReturnTo(raw string) (string, bool) {
	if raw == "" {
		return "/", true
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") || strings.Contains(raw, "\\") {
		return "", false
	}
	return raw, true
}
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": http.StatusText(status), "message": message})
}

func (s *Service) requestMagicLink(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		ReturnTo string `json:"returnTo"`
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(&input) != nil {
		writeError(w, 400, "invalid request")
		return
	}
	email, ok := validEmail(input.Email)
	if !ok {
		writeError(w, 400, "invalid email")
		return
	}
	returnTo, ok := validReturnTo(input.ReturnTo)
	if !ok {
		writeError(w, 400, "invalid return target")
		return
	}
	if !s.allowMagicLinkFor(email, r) {
		// Same answer as a delivered link, so the limit reveals nothing.
		w.WriteHeader(http.StatusAccepted)
		return
	}
	token, digest, err := NewCredential()
	if err != nil {
		writeError(w, 500, "authentication unavailable")
		return
	}
	if err = s.store.CreateMagicLink(r.Context(), digest, email, returnTo, s.now().Add(15*time.Minute)); err != nil {
		httpserver.Logger(r.Context()).Error("create magic-link challenge", "error", err)
		writeError(w, 500, "authentication unavailable")
		return
	}
	link := s.baseURL + "/api/v1/auth/magic-links/" + url.PathEscape(token)
	if err = s.mailer.SendMagicLink(r.Context(), email, link); err != nil {
		httpserver.Logger(r.Context()).Error("deliver magic-link email", "error", err)
		_ = s.store.DeleteMagicLink(r.Context(), digest)
		writeError(w, 500, "authentication unavailable")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
func (s *Service) consumeMagicLink(w http.ResponseWriter, r *http.Request) {
	sessionToken, sessionDigest, err := NewCredential()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication unavailable")
		return
	}
	expires := s.now().Add(sessionLifetime)
	login, err := s.store.ConsumeMagicLinkAndCreateSession(r.Context(), Digest(r.PathValue("token")), sessionDigest, s.now(), expires)
	if err != nil {
		writeError(w, 400, "invalid or expired magic link")
		return
	}
	SetSessionCookie(w, sessionToken, expires)
	http.Redirect(w, r, login.ReturnTo, http.StatusFound)
}
func (s *Service) establishSession(w http.ResponseWriter, ctx context.Context, a Account) error {
	token, digest, err := NewCredential()
	if err != nil {
		return err
	}
	expires := s.now().Add(sessionLifetime)
	if err = s.store.CreateSession(ctx, digest, a.ID, expires); err != nil {
		return err
	}
	SetSessionCookie(w, token, expires)
	return nil
}

func (s *Service) startOIDC(w http.ResponseWriter, r *http.Request) {
	returnTo, ok := validReturnTo(r.URL.Query().Get("returnTo"))
	if !ok {
		writeError(w, 400, "invalid return target")
		return
	}
	state, stateDigest, err := NewCredential()
	if err != nil {
		writeError(w, 500, "authentication unavailable")
		return
	}
	nonce, _, err := NewCredential()
	if err != nil {
		writeError(w, 500, "authentication unavailable")
		return
	}
	verifier := oauth2.GenerateVerifier()
	if err = s.store.CreateOIDCAttempt(r.Context(), stateDigest, nonce, verifier, returnTo, s.now().Add(10*time.Minute)); err != nil {
		writeError(w, 500, "authentication unavailable")
		return
	}
	http.Redirect(w, r, s.oidc.AuthorizationURL(state, nonce, verifier), http.StatusFound)
}
func (s *Service) callbackOIDC(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("code") == "" || r.URL.Query().Get("state") == "" {
		writeError(w, 400, "invalid callback")
		return
	}
	attempt, err := s.store.ConsumeOIDCAttempt(r.Context(), Digest(r.URL.Query().Get("state")), s.now())
	if err != nil {
		writeError(w, 400, "invalid or expired callback")
		return
	}
	identity, err := s.oidc.Verify(r.Context(), r.URL.Query().Get("code"), attempt.PKCEVerifier, attempt.Nonce)
	if err != nil {
		httpserver.Logger(r.Context()).Warn("verify OIDC identity", "error", err)
		writeError(w, 401, "identity could not be verified")
		return
	}
	if normalized, ok := validEmail(identity.Email); !ok {
		writeError(w, http.StatusUnauthorized, "identity email is invalid")
		return
	} else {
		identity.Email = normalized
	}
	account, err := s.store.ResolveOIDCIdentity(r.Context(), identity.Issuer, identity.Subject, identity.Email)
	if err != nil {
		writeError(w, 401, "identity could not be associated")
		return
	}
	if err = s.establishSession(w, r.Context(), account); err != nil {
		writeError(w, 500, "authentication unavailable")
		return
	}
	http.Redirect(w, r, attempt.ReturnTo, http.StatusFound)
}

func sessionToken(r *http.Request) (string, error) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c.Value == "" {
		return "", ErrInvalidCredential
	}
	return c.Value, nil
}

// RequireSession authenticates the opaque cookie and adds its account to the
// request context for protected API handlers.
func (s *Service) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := sessionToken(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		session, err := s.store.Session(r.Context(), Digest(token), s.now())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey{}, authenticatedSession{Session: session, digest: Digest(token)})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithAccount returns a context carrying account as the authenticated caller,
// as RequireSession does. It lets handlers outside this package be exercised
// without a stored session.
func WithAccount(ctx context.Context, account Account) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, authenticatedSession{Session: Session{Account: account}})
}

func AccountFromContext(ctx context.Context) (Account, bool) {
	authenticated, ok := ctx.Value(sessionContextKey{}).(authenticatedSession)
	return authenticated.Account, ok
}
func (s *Service) getSession(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value(sessionContextKey{}).(authenticatedSession).Session
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"profile": map[string]string{"id": session.ID, "email": session.Email, "firstName": session.FirstName, "lastName": session.LastName}})
}
func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	authenticated := r.Context().Value(sessionContextKey{}).(authenticatedSession)
	if err := s.store.RevokeSession(r.Context(), authenticated.digest, s.now()); err != nil {
		writeError(w, 500, "logout failed")
		return
	}
	ClearSessionCookie(w)
	// Drops privately cached responses, such as attachment contents, from the
	// browser. Cookies are cleared above and local storage holds no secrets.
	w.Header().Set("Clear-Site-Data", `"cache"`)
	w.WriteHeader(http.StatusNoContent)
}
