package httpserver

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"at.draab/my-car/internal/config"
)

// captureLogs routes slog output into a buffer for the duration of a test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

// testHandler is the real mux plus routes exercising the access log: a
// path-parameter route standing in for the magic-link one, an error route, a
// rate-limited route and a panicking route.
func testHandler(mode config.RequestLogMode) http.Handler {
	mux := NewMux(fakePinger{}, nil, testStaticFS(), func(mux *http.ServeMux) {
		mux.HandleFunc("GET /api/v1/auth/magic-links/{token}", func(w http.ResponseWriter, r *http.Request) {
			Logger(r.Context()).Error("consume failed", "err", "boom")
			w.WriteHeader(http.StatusInternalServerError)
		})
		mux.HandleFunc("GET /api/v1/ok", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})
		mux.HandleFunc("GET /api/v1/limited", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})
		mux.HandleFunc("GET /api/v1/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("handler exploded")
		})
	})
	return Wrap(mux, mode)
}

func serve(h http.Handler, target string, header map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range header {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestWrap_RequestIDOnEveryResponse(t *testing.T) {
	captureLogs(t)
	for _, mode := range []config.RequestLogMode{config.RequestLogAll, config.RequestLogOff} {
		h := testHandler(mode)
		seen := map[string]bool{}
		for _, target := range []string{"/api/v1/ok", "/api/does-not-exist", "/assets/app.js", "/cars/abc"} {
			id := serve(h, target, nil).Header().Get("X-Request-Id")
			if len(id) != 16 || seen[id] {
				t.Errorf("%s %s: X-Request-Id = %q, want a new 16-character ID", mode, target, id)
			}
			seen[id] = true
		}
	}
}

func TestWrap_ClientRequestIDIgnored(t *testing.T) {
	logs := captureLogs(t)
	w := serve(testHandler(config.RequestLogAll), "/api/v1/ok", map[string]string{"X-Request-Id": "abc"})
	if id := w.Header().Get("X-Request-Id"); id == "abc" || id == "" {
		t.Fatalf("X-Request-Id = %q, want a generated ID", id)
	}
	if strings.Contains(logs.String(), "request_id=abc") {
		t.Fatalf("log = %q uses the client's ID", logs.String())
	}
}

func TestLogger_DefaultOutsideRequest(t *testing.T) {
	if Logger(context.Background()) != slog.Default() {
		t.Fatal("Logger outside a request is not the default logger")
	}
}

func TestWrap_AccessEntryFields(t *testing.T) {
	logs := captureLogs(t)
	w := serve(testHandler(config.RequestLogAll), "/api/v1/ok?q=s3cret", map[string]string{"Cookie": "session=s3cret-cookie"})
	entry := logs.String()
	for _, want := range []string{"level=INFO", "msg=request", "request_id=" + w.Header().Get("X-Request-Id"), "method=GET", `route="GET /api/v1/ok"`, "status=200", "duration="} {
		if !strings.Contains(entry, want) {
			t.Errorf("log = %q, want %q", entry, want)
		}
	}
	if strings.Contains(entry, "s3cret") || strings.Contains(entry, "path=") {
		t.Errorf("log = %q contains the query, cookie or an /api/ path", entry)
	}
}

func TestWrap_MagicLinkTokenNotLogged(t *testing.T) {
	logs := captureLogs(t)
	w := serve(testHandler(config.RequestLogAll), "/api/v1/auth/magic-links/s3cret-t0ken", nil)
	entry := logs.String()
	if !strings.Contains(entry, `route="GET /api/v1/auth/magic-links/{token}"`) || strings.Contains(entry, "s3cret-t0ken") {
		t.Fatalf("log = %q, want the route template without the token", entry)
	}
	// The handler's error entry and the access entry share the response's ID.
	id := "request_id=" + w.Header().Get("X-Request-Id")
	lines := strings.Split(strings.TrimSpace(entry), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "consume failed") || !strings.Contains(lines[1], "status=500") {
		t.Fatalf("log = %q, want the handler error then the access entry", entry)
	}
	for _, line := range lines {
		if !strings.Contains(line, id) {
			t.Errorf("line %q lacks %s", line, id)
		}
	}
}

func TestWrap_FrontendPathLogged(t *testing.T) {
	logs := captureLogs(t)
	serve(testHandler(config.RequestLogAll), "/cars/abc/refuels", nil)
	if entry := logs.String(); !strings.Contains(entry, "path=/cars/abc/refuels") || !strings.Contains(entry, `route=/`) {
		t.Fatalf("log = %q, want the frontend path and the fallback route", entry)
	}
}

func TestWrap_ErrorsOnlyMode(t *testing.T) {
	logs := captureLogs(t)
	h := testHandler(config.RequestLogErrors)
	serve(h, "/api/v1/ok", nil)
	serve(h, "/api/does-not-exist", nil)
	serve(h, "/api/v1/limited", nil)
	entry := logs.String()
	if strings.Contains(entry, "status=200") || !strings.Contains(entry, "status=404") || !strings.Contains(entry, "status=429") {
		t.Fatalf("log = %q, want only the 404 and 429 entries", entry)
	}
}

func TestWrap_OffMode(t *testing.T) {
	logs := captureLogs(t)
	serve(testHandler(config.RequestLogOff), "/api/v1/ok", nil)
	if logs.Len() != 0 {
		t.Fatalf("log = %q, want no access entry", logs.String())
	}
}

func TestWrap_PanicIs500AndLogged(t *testing.T) {
	logs := captureLogs(t)
	h := testHandler(config.RequestLogAll)
	w := serve(h, "/api/v1/panic", nil)
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "internal_error") {
		t.Fatalf("panic response = %d %q", w.Code, w.Body.String())
	}
	id := "request_id=" + w.Header().Get("X-Request-Id")
	entry := logs.String()
	if !strings.Contains(entry, "level=ERROR") || !strings.Contains(entry, "panic recovered") || strings.Count(entry, id) != 2 || !strings.Contains(entry, "status=500") {
		t.Fatalf("log = %q, want an error entry and a 500 access entry, both with %s", entry, id)
	}
	if w = serve(h, "/api/healthz", nil); w.Code != http.StatusOK {
		t.Fatalf("healthz after panic = %d", w.Code)
	}
}

func TestRecoverPanic_ReraisesAbortHandler(t *testing.T) {
	captureLogs(t)
	h := recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if v := recover(); v == nil || !errors.Is(v.(error), http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler re-raised", v)
		}
	}()
	serve(h, "/", nil)
}

func TestStatusRecorder_FirstStatusWins(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	_, _ = rec.Write([]byte("x"))
	rec.WriteHeader(http.StatusTeapot)
	if rec.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 from the implicit header", rec.status)
	}
	if http.NewResponseController(rec).Flush() != nil {
		t.Fatal("Flush through the recorder failed")
	}
}
