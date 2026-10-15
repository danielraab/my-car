# Proposal

## Why

When a request to `GET /api/v1/auth/magic-links/{token}` is rate limited, the
info-level log entry includes the concrete request path, so the magic-link
token ends up in the log. The throttled request never consumes the token, so
anyone who can read the logs gets a login link that still works until it
expires.

## What Changes

- Rate-limit log entries name the endpoint by its route template (for
  example `GET /api/v1/auth/magic-links/{token}`) instead of the concrete
  request path. This applies to both the per-client and the per-recipient
  log entries.
- The rate-limiting spec gains a requirement that these log entries never
  contain a magic-link token or any other secret from the request URL.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `authentication/rate-limiting`: adds a requirement that rate-limit log
  entries identify the endpoint without any secret from the request URL.

## Impact

- `backend/internal/auth/ratelimit.go`: the two `rate limit exceeded` log
  calls.
- `backend/internal/auth/ratelimit_test.go`: a regression test for throttled
  magic-link consumption.
- Log output: the `path` field becomes a route template. Anyone grepping
  existing logs for concrete paths will need to search for the template
  instead. No API, schema or configuration changes.
- Independent of the planned request-logging change. That change applies the
  same rule to the access log.
