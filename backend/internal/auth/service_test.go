package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type fakeStore struct {
	PasskeyRepository
	magicDigest        []byte
	email, magicReturn string
	magicUsed, deleted bool
	stateDigest        []byte
	attempt            OIDCAttempt
	stateUsed          bool
	account            Account
	sessionDigest      []byte
	expires            time.Time
	revoked            bool
}

func (f *fakeStore) CreateMagicLink(_ context.Context, d []byte, e, r string, _ time.Time) error {
	f.magicDigest = append([]byte(nil), d...)
	f.email = e
	f.magicReturn = r
	return nil
}
func (f *fakeStore) DeleteMagicLink(_ context.Context, d []byte) error {
	if bytes.Equal(d, f.magicDigest) {
		f.deleted = true
	}
	return nil
}
func (f *fakeStore) ConsumeMagicLinkAndCreateSession(_ context.Context, d, session []byte, _ time.Time, expires time.Time) (MagicLinkLogin, error) {
	if f.magicUsed || !bytes.Equal(d, f.magicDigest) {
		return MagicLinkLogin{}, ErrInvalidCredential
	}
	f.magicUsed = true
	f.sessionDigest = append([]byte(nil), session...)
	f.expires = expires
	return MagicLinkLogin{Account: f.account, ReturnTo: f.magicReturn}, nil
}
func (f *fakeStore) CreateOIDCAttempt(_ context.Context, d []byte, n, v, r string, _ time.Time) error {
	f.stateDigest = append([]byte(nil), d...)
	f.attempt = OIDCAttempt{Nonce: n, PKCEVerifier: v, ReturnTo: r}
	return nil
}
func (f *fakeStore) ConsumeOIDCAttempt(_ context.Context, d []byte, _ time.Time) (OIDCAttempt, error) {
	if f.stateUsed || !bytes.Equal(d, f.stateDigest) {
		return OIDCAttempt{}, ErrInvalidCredential
	}
	f.stateUsed = true
	return f.attempt, nil
}
func (f *fakeStore) ResolveOIDCIdentity(_ context.Context, _, _, email string) (Account, error) {
	f.account.Email = normalizeEmail(email)
	return f.account, nil
}
func (f *fakeStore) CreateSession(_ context.Context, d []byte, _ string, e time.Time) error {
	f.sessionDigest = append([]byte(nil), d...)
	f.expires = e
	return nil
}
func (f *fakeStore) Session(_ context.Context, d []byte, n time.Time) (Session, error) {
	if f.revoked || !bytes.Equal(d, f.sessionDigest) || !f.expires.After(n) {
		return Session{}, ErrInvalidCredential
	}
	return Session{Account: f.account, ExpiresAt: f.expires}, nil
}
func (f *fakeStore) RevokeSession(_ context.Context, d []byte, _ time.Time) error {
	if bytes.Equal(d, f.sessionDigest) {
		f.revoked = true
	}
	return nil
}

type fakeMailer struct {
	to, link string
	err      error
}

func (f *fakeMailer) SendMagicLink(_ context.Context, to, link string) error {
	f.to, f.link = to, link
	return f.err
}

type fakeOIDC struct {
	identity               VerifiedIdentity
	err                    error
	state, nonce, verifier string
}

func (f *fakeOIDC) AuthorizationURL(s, n, v string) string {
	f.state, f.nonce, f.verifier = s, n, v
	return "https://id.example/authorize?state=" + url.QueryEscape(s)
}
func (f *fakeOIDC) Verify(context.Context, string, string, string) (VerifiedIdentity, error) {
	return f.identity, f.err
}

func newTestService() (*Service, *fakeStore, *fakeMailer, *fakeOIDC) {
	st := &fakeStore{account: Account{ID: "account-id", Email: "driver@example.com", Currency: "CHF", VolumeUnit: "gallon", DistanceUnit: "mile"}}
	m := &fakeMailer{}
	o := &fakeOIDC{identity: VerifiedIdentity{Issuer: "https://id.example", Subject: "sub", Email: "driver@example.com"}}
	s := NewService(st, m, o, "https://cars.example")
	s.now = func() time.Time { return time.Unix(1700000000, 0) }
	return s, st, m, o
}
func serve(s *Service, method, target, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestMagicLinkRequestConsumptionAndReplay(t *testing.T) {
	s, st, m, _ := newTestService()
	w := serve(s, "POST", "/api/v1/auth/magic-links", `{"email":" Driver@Example.COM ","returnTo":"/cars"}`, nil)
	if w.Code != 202 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if m.to != "driver@example.com" || st.magicReturn != "/cars" {
		t.Fatalf("mail=%q return=%q", m.to, st.magicReturn)
	}
	u, _ := url.Parse(m.link)
	token := u.Path[strings.LastIndex(u.Path, "/")+1:]
	if bytes.Contains(st.magicDigest, []byte(token)) {
		t.Fatal("raw token persisted")
	}
	w = serve(s, "GET", "/api/v1/auth/magic-links/"+token, "", nil)
	if w.Code != 302 || w.Header().Get("Location") != "/cars" {
		t.Fatalf("consume status=%d location=%q", w.Code, w.Header().Get("Location"))
	}
	c := w.Result().Cookies()[0]
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie=%#v", c)
	}
	if w = serve(s, "GET", "/api/v1/auth/magic-links/"+token, "", nil); w.Code != 400 {
		t.Fatalf("replay status=%d", w.Code)
	}
}

func TestAuthenticationMethodsReflectOIDCAvailability(t *testing.T) {
	s, _, _, _ := newTestService()
	w := serve(s, "GET", "/api/v1/auth/methods", "", nil)
	if w.Code != http.StatusOK || w.Body.String() != "{\"methods\":[\"magic_link\",\"passkey\",\"oidc\"]}\n" {
		t.Fatalf("enabled response=%d %q", w.Code, w.Body.String())
	}

	s.oidc = nil
	w = serve(s, "GET", "/api/v1/auth/methods", "", nil)
	if w.Code != http.StatusOK || w.Body.String() != "{\"methods\":[\"magic_link\",\"passkey\"]}\n" {
		t.Fatalf("disabled response=%d %q", w.Code, w.Body.String())
	}
	for _, target := range []string{"/api/v1/auth/oidc/start", "/api/v1/auth/oidc/callback"} {
		if w = serve(s, "GET", target, "", nil); w.Code != http.StatusNotFound {
			t.Fatalf("disabled route %s status=%d", target, w.Code)
		}
	}
}
func TestMagicLinkRequestDoesNotEnumerateAndCleansSendFailure(t *testing.T) {
	s, st, m, _ := newTestService()
	for _, email := range []string{"new@example.com", "existing@example.com"} {
		w := serve(s, "POST", "/api/v1/auth/magic-links", `{"email":"`+email+`"}`, nil)
		if w.Code != 202 || w.Body.Len() != 0 {
			t.Fatalf("%s response=%d %q", email, w.Code, w.Body.String())
		}
	}
	m.err = errors.New("SMTP down")
	w := serve(s, "POST", "/api/v1/auth/magic-links", `{"email":"driver@example.com"}`, nil)
	if w.Code != 500 || !st.deleted {
		t.Fatalf("status=%d deleted=%v", w.Code, st.deleted)
	}
}
func TestSessionAndLogout(t *testing.T) {
	s, st, _, _ := newTestService()
	st.sessionDigest = Digest("session")
	st.expires = s.now().Add(time.Hour)
	cookie := &http.Cookie{Name: SessionCookieName, Value: "session"}
	w := serve(s, "GET", "/api/v1/session", "", cookie)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"email":"driver@example.com"`) || !strings.Contains(w.Body.String(), `"currency":"CHF","volumeUnit":"gallon","distanceUnit":"mile"`) {
		t.Fatalf("session=%d %s", w.Code, w.Body.String())
	}
	w = serve(s, "DELETE", "/api/v1/session", "", cookie)
	if w.Code != 204 || !st.revoked || w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatalf("logout=%d revoked=%v", w.Code, st.revoked)
	}
	if got := w.Header().Get("Clear-Site-Data"); got != `"cache"` {
		t.Fatalf("Clear-Site-Data=%q", got)
	}
	if w = serve(s, "GET", "/api/v1/session", "", cookie); w.Code != 401 {
		t.Fatalf("revoked status=%d", w.Code)
	}
	if w = serve(s, "DELETE", "/api/v1/session", "", cookie); w.Code != 401 || w.Header().Get("Clear-Site-Data") != "" {
		t.Fatalf("logout without session=%d Clear-Site-Data=%q", w.Code, w.Header().Get("Clear-Site-Data"))
	}
}
func TestOIDCStartCallbackAndReplay(t *testing.T) {
	s, st, _, o := newTestService()
	w := serve(s, "GET", "/api/v1/auth/oidc/start?returnTo=%2Fcars", "", nil)
	if w.Code != 302 || o.state == "" || o.nonce == "" || o.verifier == "" || st.attempt.ReturnTo != "/cars" {
		t.Fatalf("start=%d %#v", w.Code, o)
	}
	w = serve(s, "GET", "/api/v1/auth/oidc/callback?code=code&state="+url.QueryEscape(o.state), "", nil)
	if w.Code != 302 || w.Header().Get("Location") != "/cars" {
		t.Fatalf("callback=%d %s", w.Code, w.Body.String())
	}
	if w = serve(s, "GET", "/api/v1/auth/oidc/callback?code=code&state="+url.QueryEscape(o.state), "", nil); w.Code != 400 {
		t.Fatalf("replay=%d", w.Code)
	}
}
func TestOIDCRejectsExternalReturnAndInvalidIdentity(t *testing.T) {
	s, _, _, o := newTestService()
	if w := serve(s, "GET", "/api/v1/auth/oidc/start?returnTo=https://evil.example", "", nil); w.Code != 400 {
		t.Fatalf("external return=%d", w.Code)
	}
	if w := serve(s, "GET", "/api/v1/auth/oidc/start?returnTo=%2F%2Fevil.example", "", nil); w.Code != 400 {
		t.Fatalf("protocol-relative return=%d", w.Code)
	}
	w := serve(s, "GET", "/api/v1/auth/oidc/start", "", nil)
	o.err = errors.New("invalid claims")
	w = serve(s, "GET", "/api/v1/auth/oidc/callback?code=code&state="+url.QueryEscape(o.state), "", nil)
	if w.Code != 401 {
		t.Fatalf("invalid identity=%d", w.Code)
	}
}
func TestOIDCClientAuthorizationURLUsesNonceAndPKCE(t *testing.T) {
	c := &OIDCClient{oauth: oauth2.Config{ClientID: "client", RedirectURL: "https://cars.example/api/v1/auth/oidc/callback", Endpoint: oauth2.Endpoint{AuthURL: "https://id.example/authorize"}}}
	u, _ := url.Parse(c.AuthorizationURL("state", "nonce", "verifier-verifier-verifier-verifier-verifier"))
	q := u.Query()
	if q.Get("state") != "state" || q.Get("nonce") != "nonce" || q.Get("code_challenge") != "" && q.Get("code_challenge_method") != "S256" {
		t.Fatalf("query=%v", q)
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("missing PKCE challenge")
	}
}
