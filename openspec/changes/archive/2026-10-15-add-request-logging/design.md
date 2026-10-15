# Design

## Context

- `httpserver.NewMux` builds one `http.ServeMux`. `main.go` hands it
  straight to `http.ListenAndServe`, so there is no middleware layer yet.
- 36 `log.Printf`/`log.Fatalf` calls are spread over `main.go`, `auth` and
  every domain handler. `auth/ratelimit.go` already uses `slog`.
- The sibling project `family-finances` has the same middleware chain and
  the same two settings in production. This design ports it, with one
  correction: family-finances logs `r.URL.Path`, which in my-cars would leak
  the magic-link token (see proposal.md).
- Go 1.26: `r.Pattern` holds the matched mux pattern after routing.

## Goals / Non-Goals

**Goals:**
- One logger, one level setting, one access line per request.
- Any handler log entry can be matched to the request and response that
  caused it.

**Non-Goals:**
- JSON output or a `LOG_FORMAT` setting. Text output is enough for a
  self-hosted app, and a format option is cheap to add later.
- Client IP or user/account ID in the access entry. Both are personal data;
  the rate-limit entries already carry the client key where it matters.
- Excluding static assets or `/api/healthz` from the access log. Operators
  who find it noisy use `LOG_REQUESTS=errors`.
- Tracing, metrics, or accepting an upstream request ID.

## Decisions

**Middleware chain, outermost first.** It wraps the mux in `main.go`:

```
requestContext ─▶ recoverPanic ─▶ logRequests ─▶ ServeMux
 (ID, header,      (log + 500)     (skipped when    (routes)
  ctx logger)                       LOG_REQUESTS=off)
```

`requestContext` sits outermost so the panic entry and the access entry both
have the ID. `recoverPanic` sits outside `logRequests` so that the
recorder's status reflects the 500 that `recoverPanic` writes. Concretely,
`logRequests` uses a deferred log call, which also runs when the panic
unwinds through it, and records status 500 when nothing was written yet. The
chain lives in `internal/httpserver` (`middleware.go`), next to the mux it
wraps.

**Route template from `r.Pattern`, path only outside `/api/`.**
`logRequests` sits outside the mux, so it reads the pattern after
`next.ServeHTTP` returns. The mux sets the pattern on the same `*Request`
it was handed, so the outer middleware can read it. A test pins this
behaviour so a Go upgrade can't silently empty the field. Unknown `/api/…`
paths match the `/api/` catch-all, so their concrete path, which could
contain anything, is never logged either.
- *Alternative: log `r.URL.Path` and redact known secret routes.* Rejected.
  It fails open when a new route carries a secret.

**Request-scoped logger via context.** `requestContext` stores
`slog.Default().With("request_id", id)` in the context.
`httpserver.Logger(ctx)` returns it, or `slog.Default()` outside a request.
Handlers log errors with `httpserver.Logger(r.Context()).Error(...)`.
- *Alternative: a custom `slog.Handler` that pulls the ID from the context
  passed to `slog.InfoContext`.* It is more transparent, but only works if
  every call site uses the `…Context` variants, which is easy to get wrong.
  The explicit logger is greppable. `ratelimit.go` moves to the
  request-scoped logger too.

**Request ID.** 8 random bytes from `crypto/rand`, hex encoded (16
characters). It is unique enough to correlate logs, and it identifies
nothing.

**Levels and setup.** Right after loading the config, `runServer` installs
`slog.NewTextHandler(os.Stderr, {Level: cfg.LogLevel})` as the default.
Merely calling `slog.SetLogLoggerLevel` would keep slog's default
`log`-backed output (`2026/10/15 12:00:00 INFO msg k=v`), which is not the
`key=value` format promised to operators. Setting a real handler also routes
the standard `log` package (for example net/http's own errors) through slog.
A config error is still printed, at error level, by the default logger.
Mapping of existing calls: `log.Fatalf` → `slog.Error` + `os.Exit(1)`;
infrastructure errors in handlers → `Error` on the request logger;
client-caused verification failures (passkey registration/login, OIDC
identity) and the cloned-authenticator notice → `Warn`; startup notices
(`listening`, `analytics snippet configured`, seed summary) → `Info`. The
`healthcheck` and `seed` subcommands use the default slog logger.

**Config.** Two fields, `LogLevel slog.Level` and `LogRequests` (a small
string enum `all|errors|off`). They are parsed in `config.Load` the same way
as `RATE_LIMIT_PER_IP`, and added to `EnvKeys` and `.env.example`;
`config_test.go` enforces that the two match.

**Status recorder.** It records the first `WriteHeader`, treats a `Write`
without a header as 200, and implements `Unwrap()` so `http.ResponseController`
keeps working through it.

**OpenAPI.** `X-Request-Id` is documented once as a reusable response
header component and mentioned in the document's description. Adding it to
every operation's responses would be pure noise.

## Risks / Trade-offs

- [Log format change breaks someone's parser] → Note it in the release
  notes. No known consumer exists today.
- [`r.Pattern` stops being visible to outer middleware in a future Go
  release] → The test from tasks 2.x fails, and the fallback is an empty
  route, never the concrete path.
- [`/api/healthz` from the Docker healthcheck adds a line every interval]
  → Accepted. `LOG_REQUESTS=errors` removes it.
- [A panic after the response has started] → The 500 cannot be sent. The
  connection is closed by `http.Server` and the panic is still logged with
  the request ID. The access entry shows the already written status.
