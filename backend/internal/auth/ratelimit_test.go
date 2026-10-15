package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestClientAddrTrustsForwardedForOnlyFromTrustedProxies(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("172.30.0.0/24"), netip.MustParsePrefix("2001:db8:ffff::/48")}
	for _, tc := range []struct {
		name, remote string
		forwarded    []string
		trusted      []netip.Prefix
		want         string
	}{
		{"no trusted proxies ignores the header", "198.51.100.7:4000", []string{"203.0.113.9"}, nil, "198.51.100.7"},
		{"untrusted peer ignores the header", "198.51.100.7:4000", []string{"203.0.113.9"}, trusted, "198.51.100.7"},
		{"trusted proxy", "172.30.0.2:4000", []string{"198.51.100.1, 203.0.113.9"}, trusted, "203.0.113.9"},
		{"spoofed leftmost entry", "172.30.0.2:4000", []string{"192.0.2.66, 203.0.113.9"}, trusted, "203.0.113.9"},
		{"chained trusted proxies", "172.30.0.2:4000", []string{"203.0.113.9, 172.30.0.7"}, trusted, "203.0.113.9"},
		{"repeated headers", "172.30.0.2:4000", []string{"192.0.2.66", "203.0.113.9"}, trusted, "203.0.113.9"},
		{"malformed entry stops the walk", "172.30.0.2:4000", []string{"203.0.113.9, not-an-ip"}, trusted, "172.30.0.2"},
		{"only trusted entries", "172.30.0.2:4000", []string{"172.30.0.9"}, trusted, "172.30.0.2"},
		{"no header from trusted proxy", "172.30.0.2:4000", nil, trusted, "172.30.0.2"},
		{"entry with port", "172.30.0.2:4000", []string{"203.0.113.9:5555"}, trusted, "203.0.113.9"},
		{"IPv4-mapped peer", "[::ffff:172.30.0.2]:4000", []string{"203.0.113.9"}, trusted, "203.0.113.9"},
		{"IPv6 proxy and client", "[2001:db8:ffff::1]:4000", []string{"2001:db8:1:2::a"}, trusted, "2001:db8:1:2::a"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tc.remote
		for _, value := range tc.forwarded {
			r.Header.Add("X-Forwarded-For", value)
		}
		if got := clientAddr(r, tc.trusted); got != netip.MustParseAddr(tc.want) {
			t.Errorf("%s: client = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestClientKeyGroupsIPv6ByPrefix(t *testing.T) {
	a, b := clientKey(netip.MustParseAddr("2001:db8:1:2::a")), clientKey(netip.MustParseAddr("2001:db8:1:2:ffff::b"))
	if a != b || a != "2001:db8:1:2::/64" {
		t.Fatalf("IPv6 keys %q and %q, want the same /64", a, b)
	}
	if other := clientKey(netip.MustParseAddr("2001:db8:1:3::a")); other == a {
		t.Fatalf("different /64 shares key %q", other)
	}
	if got := clientKey(netip.MustParseAddr("203.0.113.9")); got != "203.0.113.9" {
		t.Fatalf("IPv4 key %q", got)
	}
}

func TestKeyedLimiterBurstRefillAndEviction(t *testing.T) {
	l := newKeyedLimiter(rate.Every(2*time.Minute), 3)
	now := time.Unix(1700000000, 0)
	for i := 0; i < 3; i++ {
		if ok, _ := l.allow("a", now); !ok {
			t.Fatalf("burst request %d rejected", i+1)
		}
	}
	ok, retry := l.allow("a", now)
	if ok || retry != 2*time.Minute {
		t.Fatalf("over burst ok=%v retry=%s, want rejection with 2m", ok, retry)
	}
	if ok, _ = l.allow("b", now); !ok {
		t.Fatal("independent key rejected")
	}
	if ok, _ = l.allow("a", now.Add(2*time.Minute)); !ok {
		t.Fatal("refilled token rejected")
	}
	l.allow("c", now.Add(3*time.Minute))
	if ok, _ = l.allow("c", now.Add(20*time.Minute)); !ok {
		t.Fatal("request after idle period rejected")
	}
	if _, kept := l.entries["a"]; kept {
		t.Fatal("idle key a was not evicted")
	}
	if len(l.entries) != 1 {
		t.Fatalf("entries=%d, want only the active key", len(l.entries))
	}
}

func TestWindowLimiterSlidingWindow(t *testing.T) {
	l := newWindowLimiter(3, 15*time.Minute)
	now := time.Unix(1700000000, 0)
	for i := 0; i < 3; i++ {
		if !l.allow("r", now.Add(time.Duration(i)*5*time.Minute)) {
			t.Fatalf("event %d rejected", i+1)
		}
	}
	if l.allow("r", now.Add(14*time.Minute)) {
		t.Fatal("fourth event within 15 minutes allowed")
	}
	if !l.allow("r", now.Add(15*time.Minute)) {
		t.Fatal("event after the first left the window rejected")
	}
	if l.allow("r", now.Add(16*time.Minute)) {
		t.Fatal("window not full again after re-admission")
	}
	l.allow("other", now.Add(2*time.Hour))
	if _, kept := l.events["r"]; kept {
		t.Fatal("expired key was not swept")
	}
}

// captureLogs routes slog output into a buffer for the duration of a test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

type countingMailer struct{ sent []string }

func (m *countingMailer) SendMagicLink(_ context.Context, to, _ string) error {
	m.sent = append(m.sent, to)
	return nil
}

func serveFrom(s *Service, remote, method, target, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.RemoteAddr = remote
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func magicLinkBody(email string) string {
	return `{"email":"` + email + `"}`
}

func TestMagicLinkRequestsAreLimitedPerClient(t *testing.T) {
	logs := captureLogs(t)
	s, _, _, _ := newTestService()
	mailer := &countingMailer{}
	s.mailer = mailer
	for i := 0; i < 3; i++ {
		if w := serveFrom(s, "198.51.100.7:4000", "POST", "/api/v1/auth/magic-links", magicLinkBody("user"+strconv.Itoa(i)+"@example.com")); w.Code != http.StatusAccepted {
			t.Fatalf("request %d=%d", i+1, w.Code)
		}
	}
	w := serveFrom(s, "198.51.100.7:4000", "POST", "/api/v1/auth/magic-links", magicLinkBody("user4@example.com"))
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "120" {
		t.Fatalf("fourth request=%d retry-after=%q", w.Code, w.Header().Get("Retry-After"))
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != codeRateLimited {
		t.Fatalf("body=%s", w.Body.String())
	}
	if len(mailer.sent) != 3 {
		t.Fatalf("sent %d emails, want 3", len(mailer.sent))
	}
	if entry := logs.String(); !strings.Contains(entry, "level=INFO") || !strings.Contains(entry, "limit=client") || !strings.Contains(entry, "group=magic_link_request") || !strings.Contains(entry, "key=198.51.100.7") {
		t.Fatalf("log=%q", entry)
	}

	if w = serveFrom(s, "203.0.113.9:4000", "POST", "/api/v1/auth/magic-links", magicLinkBody("other@example.com")); w.Code != http.StatusAccepted {
		t.Fatalf("other client=%d", w.Code)
	}
	if w = serveFrom(s, "198.51.100.7:4000", "GET", "/api/v1/auth/magic-links/unknown", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("other group from limited client=%d, want the handler's own 400", w.Code)
	}
}

func TestBrowserNavigationIsRedirectedWhenLimited(t *testing.T) {
	logs := captureLogs(t)
	s, st, _, o := newTestService()
	for i := 0; i < 30; i++ {
		if w := serveFrom(s, "198.51.100.7:4000", "GET", "/api/v1/auth/oidc/start", ""); w.Code != http.StatusFound {
			t.Fatalf("request %d=%d", i+1, w.Code)
		}
	}
	o.state, st.stateDigest = "", nil
	w := serveFrom(s, "198.51.100.7:4000", "GET", "/api/v1/auth/oidc/start", "")
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/auth/login?error=rate_limited" {
		t.Fatalf("limited start=%d location=%q", w.Code, w.Header().Get("Location"))
	}
	if o.state != "" || st.stateDigest != nil {
		t.Fatal("limited start created an OIDC attempt")
	}
	for _, target := range []string{"/api/v1/auth/oidc/callback?code=c&state=s", "/api/v1/auth/magic-links/token"} {
		if w = serveFrom(s, "198.51.100.7:4000", "GET", target, ""); w.Code != http.StatusSeeOther {
			t.Fatalf("%s=%d, want the shared group's redirect", target, w.Code)
		}
	}
	if !strings.Contains(logs.String(), "group=login_redirect") {
		t.Fatalf("log=%q", logs.String())
	}
	for i := 0; i < 50; i++ {
		if w = serveFrom(s, "198.51.100.7:4000", "GET", "/api/v1/session", ""); w.Code != http.StatusUnauthorized {
			t.Fatalf("authenticated route=%d", w.Code)
		}
	}
}

func TestPerIPLimitingCanBeDisabled(t *testing.T) {
	logs := captureLogs(t)
	s, _, _, _ := newTestService()
	s.ConfigureRateLimiting(nil, false)
	if !strings.Contains(logs.String(), "level=INFO") || !strings.Contains(logs.String(), "per-IP rate limiting disabled") {
		t.Fatalf("startup log=%q", logs.String())
	}
	for i := 0; i < 10; i++ {
		if w := serveFrom(s, "198.51.100.7:4000", "POST", "/api/v1/auth/magic-links", magicLinkBody("user"+strconv.Itoa(i)+"@example.com")); w.Code != http.StatusAccepted {
			t.Fatalf("request %d=%d", i+1, w.Code)
		}
	}
}

func TestMagicLinkRecipientLimitIsSilent(t *testing.T) {
	for _, perIP := range []bool{true, false} {
		logs := captureLogs(t)
		s, st, _, _ := newTestService()
		s.ConfigureRateLimiting(nil, perIP)
		mailer := &countingMailer{}
		s.mailer = mailer
		var last *httptest.ResponseRecorder
		for i := 0; i < 4; i++ {
			st.magicDigest = nil
			last = serveFrom(s, "198.51.100."+strconv.Itoa(i+1)+":4000", "POST", "/api/v1/auth/magic-links", magicLinkBody(" Victim@Example.com "))
			if last.Code != http.StatusAccepted || last.Body.Len() != 0 {
				t.Fatalf("perIP=%v request %d=%d %q", perIP, i+1, last.Code, last.Body.String())
			}
		}
		if len(mailer.sent) != 3 || st.magicDigest != nil {
			t.Fatalf("perIP=%v sent=%d challenge created=%v", perIP, len(mailer.sent), st.magicDigest != nil)
		}
		entry := logs.String()
		if !strings.Contains(entry, "limit=recipient") || strings.Contains(strings.ToLower(entry), "victim") {
			t.Fatalf("perIP=%v log=%q", perIP, entry)
		}
	}
}

func TestRateLimitLogOmitsURLSecrets(t *testing.T) {
	for _, tc := range []struct {
		name, target, route string
		secrets             []string
	}{
		{"magic-link consumption", "/api/v1/auth/magic-links/s3cret-t0ken", "GET /api/v1/auth/magic-links/{token}", []string{"s3cret-t0ken"}},
		{"OIDC callback", "/api/v1/auth/oidc/callback?code=s3cret-c0de&state=s3cret-st4te", "GET /api/v1/auth/oidc/callback", []string{"s3cret-c0de", "s3cret-st4te"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _, _ := newTestService()
			for i := 0; i < 30; i++ {
				serveFrom(s, "198.51.100.7:4000", "GET", "/api/v1/auth/oidc/start", "")
			}
			logs := captureLogs(t)
			if w := serveFrom(s, "198.51.100.7:4000", "GET", tc.target, ""); w.Code != http.StatusSeeOther {
				t.Fatalf("throttled request=%d", w.Code)
			}
			entry := logs.String()
			if !strings.Contains(entry, "rate limit exceeded") || !strings.Contains(entry, tc.route) {
				t.Fatalf("log=%q, want the route %q", entry, tc.route)
			}
			for _, secret := range tc.secrets {
				if strings.Contains(entry, secret) {
					t.Fatalf("log=%q contains %q", entry, secret)
				}
			}
		})
	}
}
