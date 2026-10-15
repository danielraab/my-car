package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"at.draab/my-car/internal/apierror"
	"at.draab/my-car/internal/config"
)

type ctxKey int

const loggerKey ctxKey = iota

// Logger returns the request-scoped logger, tagged with the request's
// request_id, or the default logger outside a request. Handlers log through it
// so their entries correlate with the access log and the X-Request-Id header.
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Wrap applies the standard middleware chain to h, outermost first:
// requestContext (request ID, X-Request-Id header, request-scoped logger),
// recoverPanic (a panic becomes a logged 500), then logRequests (one access
// entry per request as selected by mode; left out entirely for off), per
// specs/platform/logging.
func Wrap(h http.Handler, mode config.RequestLogMode) http.Handler {
	if mode != config.RequestLogOff {
		h = logRequests(h, mode == config.RequestLogErrors)
	}
	return requestContext(recoverPanic(h))
}

func requestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), loggerKey, slog.Default().With("request_id", id))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// recoverPanic turns a panic in any inner layer into an error entry and, if no
// response has been started yet, a 500. http.ErrAbortHandler is re-raised so
// net/http aborts the response silently, as that sentinel intends.
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if err, ok := v.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(v)
			}
			Logger(r.Context()).Error("panic recovered", "value", v, "route", r.Pattern)
			if !rec.wroteHeader {
				apierror.Internal(w)
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// logRequests writes one info-level access entry per request; with errorsOnly
// only responses with status 400 or higher are logged. API requests are named
// by their route template only, since a concrete /api/ path can carry a secret
// such as the magic-link token; query strings, headers and bodies are never
// logged. The entry is written from a deferred call so a panicking handler is
// logged too, as the 500 recoverPanic is about to answer.
func logRequests(next http.Handler, errorsOnly bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		completed := false
		defer func() {
			status := rec.status
			if !completed && !rec.wroteHeader {
				status = http.StatusInternalServerError
			}
			if errorsOnly && status < http.StatusBadRequest {
				return
			}
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				// The mux sets Pattern on the request it was handed, so it is
				// visible here once ServeHTTP has returned.
				slog.String("route", r.Pattern),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
			}
			if r.URL.Path != "/api" && !strings.HasPrefix(r.URL.Path, "/api/") {
				attrs = append(attrs, slog.String("path", r.URL.Path))
			}
			Logger(r.Context()).LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		}()
		next.ServeHTTP(rec, r)
		completed = true
	})
}

// statusRecorder captures the response status for the access log and whether
// a response has been started.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// newRequestID returns 8 random bytes, hex encoded.
func newRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error.
	return hex.EncodeToString(b[:])
}
