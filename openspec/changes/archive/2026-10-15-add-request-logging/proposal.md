# Proposal

## Why

The backend writes no log line for an HTTP request, so an operator cannot see
what traffic hit the server or which request produced a given error line.
Logging is also split between `log.Printf` and `slog` with no level control,
although the rate-limiting spec already relies on "info level" entries. A
panic in a handler resets the connection instead of answering with a 500.

## What Changes

- New `LOG_LEVEL` setting (`debug | info | warn | error`, default `info`)
  sets the minimum level of all backend log output.
- New `LOG_REQUESTS` setting (`all | errors | off`, default `all`). It
  selects which HTTP requests get one structured access-log line: every
  request, only those with status >= 400, or none.
- The access line records method, route template, status, duration and
  request ID. It never contains the query string, headers, body or the
  concrete path of an `/api/` request, because API paths can carry secrets
  such as the magic-link token.
- Every response carries an `X-Request-Id` header. Log lines that handlers
  write while serving a request carry the same ID.
- A panic in a handler is logged with its request ID and answered with
  `500` instead of a reset connection.
- All existing `log.Printf`/`log.Fatalf` calls in the backend move to
  `log/slog`, so every line goes through one logger and one level setting.
  Output stays plain text; there is no JSON format option.
- Invalid `LOG_LEVEL` or `LOG_REQUESTS` values fail startup.

## Capabilities

### New Capabilities

- `platform/logging`: backend log levels, the HTTP access log and its
  privacy rules, request IDs and their correlation with handler logs, and
  panic recovery.

### Modified Capabilities

_None._ The rate-limiting spec's info-level entries keep their meaning; this
change only makes the level configurable.

## Impact

- `backend/internal/config`: two new variables, validation and `EnvKeys`.
- `backend/internal/httpserver`: new middleware wrapping the mux.
- `backend/main.go` and every domain package that logs (`auth`, `cars`,
  `profile`, `refuels`, `repairs`, `stats`, `tickets`): move to `slog`.
  Handler error logs use the request-scoped logger.
- `backend/.env.example`, `docker-compose.yml` (app service) and
  `backend/README.md`: document the new variables.
- `backend/openapi.yaml`: documents the `X-Request-Id` response header.
- The log output format changes from `log`'s date prefix to `slog`'s
  key=value text. Anyone parsing the old format has to adapt.
- No database, frontend or dependency changes. Related to
  `fix-ratelimit-log-token-leak`, which applies the same route-template rule
  to the rate-limit log entries. The two are independent and can land in
  either order.
