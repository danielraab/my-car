package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/httpserver"
	"golang.org/x/time/rate"
)

// Endpoint groups of the public authentication routes; each has its own
// per-client limit.
const (
	groupMagicLinkRequest = "magic_link_request"
	groupLoginRedirect    = "login_redirect"
	groupPasskeyLogin     = "passkey_login"
)

const (
	codeRateLimited      = "rate_limited"
	rateLimitedLoginPath = "/auth/login?error=rate_limited"

	// recipientLimit magic-link emails per address within recipientWindow.
	recipientLimit  = 3
	recipientWindow = 15 * time.Minute

	limiterSweepInterval = time.Minute
)

// clientLimits are the per-client token buckets of each endpoint group.
var clientLimits = map[string]struct {
	every time.Duration
	burst int
}{
	groupMagicLinkRequest: {every: 2 * time.Minute, burst: 3},  // 5 per 10 minutes
	groupLoginRedirect:    {every: 2 * time.Second, burst: 30}, // 30 per minute
	groupPasskeyLogin:     {every: 2 * time.Second, burst: 30}, // 30 per minute
}

type limitMode int

const (
	limitJSON limitMode = iota
	limitRedirect
)

// rateLimits holds the authentication rate-limiting state of a Service.
type rateLimits struct {
	perIP      bool
	trusted    []netip.Prefix
	clients    map[string]*keyedLimiter
	recipients *windowLimiter
}

func newRateLimits() *rateLimits {
	clients := make(map[string]*keyedLimiter, len(clientLimits))
	for group, l := range clientLimits {
		clients[group] = newKeyedLimiter(rate.Every(l.every), l.burst)
	}
	return &rateLimits{perIP: true, clients: clients, recipients: newWindowLimiter(recipientLimit, recipientWindow)}
}

// ConfigureRateLimiting sets the reverse proxies whose X-Forwarded-For header
// is trusted and whether public authentication routes are limited per client
// address. It must be called before RegisterRoutes. The per-recipient
// magic-link limit applies either way.
func (s *Service) ConfigureRateLimiting(trustedProxies []netip.Prefix, perIP bool) {
	s.limits.trusted = trustedProxies
	s.limits.perIP = perIP
	if !perIP {
		slog.Info("per-IP rate limiting disabled; only the per-recipient magic-link limit applies")
	}
}

// limit wraps a public authentication handler with the per-client limit of
// its endpoint group.
func (s *Service) limit(group string, mode limitMode, next http.HandlerFunc) http.Handler {
	if !s.limits.perIP {
		return next
	}
	limiter := s.limits.clients[group]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientKey(clientAddr(r, s.limits.trusted))
		ok, retryAfter := limiter.allow(key, s.now())
		if ok {
			next(w, r)
			return
		}
		httpserver.Logger(r.Context()).Info("rate limit exceeded", "limit", "client", "group", group, "key", key, "path", r.Pattern)
		if mode == limitRedirect {
			http.Redirect(w, r, rateLimitedLoginPath, http.StatusSeeOther)
			return
		}
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(max(retryAfter, time.Second).Seconds()))))
		apierror.Write(w, http.StatusTooManyRequests, codeRateLimited, "too many requests; try again later", nil)
	})
}

// allowMagicLinkFor applies the per-recipient limit to a normalized email,
// logging a hit without the address itself.
func (s *Service) allowMagicLinkFor(email string, r *http.Request) bool {
	sum := sha256.Sum256([]byte(email))
	key := hex.EncodeToString(sum[:])
	if s.limits.recipients.allow(key, s.now()) {
		return true
	}
	httpserver.Logger(r.Context()).Info("rate limit exceeded", "limit", "recipient", "group", groupMagicLinkRequest, "key", key[:12], "path", r.Pattern)
	return false
}

// clientAddr resolves the client address of a request. X-Forwarded-For is
// believed only when the connection comes from a trusted proxy: its entries
// are walked from the right, skipping trusted proxies, and the first other
// address is the client. A malformed entry ends the walk at the connection's
// address, since everything left of it is client-controlled.
func clientAddr(r *http.Request, trusted []netip.Prefix) netip.Addr {
	remote := parseAddr(r.RemoteAddr)
	if !remote.IsValid() || !isTrusted(remote, trusted) {
		return remote
	}
	var hops []string
	for _, header := range r.Header.Values("X-Forwarded-For") {
		hops = append(hops, strings.Split(header, ",")...)
	}
	for i := len(hops) - 1; i >= 0; i-- {
		hop := parseAddr(strings.TrimSpace(hops[i]))
		if !hop.IsValid() {
			return remote
		}
		if !isTrusted(hop, trusted) {
			return hop
		}
	}
	return remote
}

// parseAddr accepts an address with or without a port.
func parseAddr(value string) netip.Addr {
	if addrPort, err := netip.ParseAddrPort(value); err == nil {
		return addrPort.Addr().Unmap()
	}
	if addr, err := netip.ParseAddr(value); err == nil {
		return addr.Unmap()
	}
	return netip.Addr{}
}

func isTrusted(addr netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// clientKey groups IPv4 clients by address and IPv6 clients by /64 prefix.
func clientKey(addr netip.Addr) string {
	if !addr.IsValid() {
		return "unknown"
	}
	if addr.Is6() {
		prefix, _ := addr.Prefix(64)
		return prefix.String()
	}
	return addr.String()
}

// keyedLimiter keeps one token bucket per key. Keys idle long enough for
// their bucket to refill completely are dropped, so memory follows the
// recently active clients.
type keyedLimiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	idle      time.Duration
	entries   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	limiter *rate.Limiter
	seen    time.Time
}

func newKeyedLimiter(limit rate.Limit, burst int) *keyedLimiter {
	idle := time.Duration(float64(burst) / float64(limit) * float64(time.Second))
	return &keyedLimiter{limit: limit, burst: burst, idle: idle, entries: map[string]*bucket{}}
}

// allow takes a token for key, or reports how long until one is available.
func (l *keyedLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)
	b, ok := l.entries[key]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.entries[key] = b
	}
	b.seen = now
	reservation := b.limiter.ReserveN(now, 1)
	if delay := reservation.DelayFrom(now); delay > 0 {
		reservation.CancelAt(now)
		return false, delay
	}
	return true, 0
}

func (l *keyedLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < limiterSweepInterval {
		return
	}
	l.lastSweep = now
	for key, b := range l.entries {
		if now.Sub(b.seen) > l.idle {
			delete(l.entries, key)
		}
	}
}

// windowLimiter allows at most max events per key within a sliding window.
type windowLimiter struct {
	mu        sync.Mutex
	max       int
	window    time.Duration
	events    map[string][]time.Time
	lastSweep time.Time
}

func newWindowLimiter(max int, window time.Duration) *windowLimiter {
	return &windowLimiter{max: max, window: window, events: map[string][]time.Time{}}
}

func (l *windowLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) >= limiterSweepInterval {
		l.lastSweep = now
		for k, events := range l.events {
			if recent := l.recent(events, now); len(recent) == 0 {
				delete(l.events, k)
			} else {
				l.events[k] = recent
			}
		}
	}
	events := l.recent(l.events[key], now)
	if len(events) >= l.max {
		l.events[key] = events
		return false
	}
	l.events[key] = append(events, now)
	return true
}

// recent drops events that have left the window.
func (l *windowLimiter) recent(events []time.Time, now time.Time) []time.Time {
	kept := events[:0]
	for _, at := range events {
		if now.Sub(at) < l.window {
			kept = append(kept, at)
		}
	}
	return kept
}
