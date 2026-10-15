# Design

## Context

`Service.limit` in `internal/auth/ratelimit.go` wraps each public auth handler,
and the wrapped handler is registered on the `http.ServeMux`. Because of that,
the mux has already matched the route by the time the limiter runs, and
`r.Pattern` holds the matched pattern (Go 1.23+; the module targets Go 1.26).
For a throttled magic-link consumption, for example, it is
`GET /api/v1/auth/magic-links/{token}`.

## Goals / Non-Goals

**Goals:**
- No secret from the request URL in rate-limit log entries.
- Keep the entry useful: an operator can still tell which endpoint was hit.

**Non-Goals:**
- A general access log or a request-scoped logger. Both belong to the
  separate request-logging change.
- Cleaning up historical logs. Operators who might have affected logs are
  told so in the release notes (see Risks).

## Decisions

- **Log `r.Pattern` instead of `r.URL.Path`.** The pattern is stable,
  contains no request data and groups naturally per route.
  - *Alternative: redact the last path segment of magic-link URLs.* Rejected.
    This is a per-route special case that breaks silently when a new route
    carries a secret in the path.
  - *Alternative: drop the field.* Rejected. `group` alone does not tell the
    three redirect endpoints apart.
- **Keep the field name `path`.** Only its content changes, so existing log
  queries that filter on the field keep working. Renaming it to `route` would
  be clearer but would also break those queries for no gain.
- **Fall back to an empty value if `r.Pattern` is empty.** This cannot happen
  for mux-registered handlers, but if the handler were ever invoked directly
  (for example in a test), it must never fall back to `r.URL.Path`.

## Risks / Trade-offs

- [Tokens may already be in existing logs] → Magic-link tokens are
  short-lived, so any leaked token has long expired. Mention the fix in the
  release notes so operators can purge old logs if their retention policy
  requires it.
- [A handler test calls `limit` without a mux, so `r.Pattern` is empty] → The
  regression test goes through a real `ServeMux`, so it checks what happens
  in production.
