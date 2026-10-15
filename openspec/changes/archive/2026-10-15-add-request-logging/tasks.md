# Tasks

## 1. Configuration

- [x] 1.1 Add `LOG_LEVEL` and `LOG_REQUESTS` to `internal/config` (fields, parsing, defaults, startup errors naming the setting, `EnvKeys`) with table tests for valid, empty and invalid values; verify with `go test ./internal/config`.
- [x] 1.2 Document both variables in `backend/.env.example`, the app service in `docker-compose.yml` (`${LOG_LEVEL:-info}`, `${LOG_REQUESTS:-all}`) and `backend/README.md`; verify that the `EnvKeys` / `.env.example` sync test passes and `docker compose config` succeeds.

## 2. Middleware

- [x] 2.1 Add `internal/httpserver/middleware.go` with `requestContext` (ID, `X-Request-Id`, context logger, `Logger(ctx)`), and tests covering the header on every response, a client-supplied ID being ignored, and `Logger` falling back to the default outside a request; verify with `go test ./internal/httpserver`.
- [x] 2.2 Add `logRequests` with a status recorder (first status wins, `Write` implies 200, `Unwrap`) and tests for `all` and `errors` mode, the 429 case, route template from the mux, a magic-link path whose token is absent from the entry, a query string absent from the entry, and the concrete path present for a frontend route; verify with `go test ./internal/httpserver`.
- [x] 2.3 Add `recoverPanic` and the `Wrap(handler, mode)` chain (skipping `logRequests` for `off`), with tests for a panicking handler (500, error entry with ID, access entry status 500), the server serving the next request, and `off` mode still setting `X-Request-Id`; verify with `go test ./internal/httpserver`.

## 3. Wiring and slog migration

- [x] 3.1 In `main.go`, set the slog level from config, wrap the mux with the chain, and move all `log.*` calls to `slog` (fatal → `Error` + `os.Exit(1)`); verify that `go build ./...` succeeds and `main.go` no longer imports `"log"`.
- [x] 3.2 Move handler error logging in `auth`, `cars`, `profile`, `refuels`, `repairs`, `stats` and `tickets`, plus the rate-limit entries in `auth/ratelimit.go`, to `httpserver.Logger(r.Context())`; verify that no non-test file imports `"log"` any more and that `go test ./...` and `go vet ./...` pass.
- [x] 3.3 Add an integration test through the real mux plus chain in which a handler error entry and the access entry share the response's `X-Request-Id`; verify with `go test ./...`.

## 4. API contract

- [x] 4.1 Add an `X-Request-Id` response header component to `backend/openapi.yaml` and mention it in the document's description; regenerate the frontend schema if the generator picks it up (`pnpm generate:api`), and verify that `openapi_sync_test.go` and `pnpm check:api` pass.

## 5. Integration check

- [x] 5.1 Run `./dev.sh`, log in via magic link through Mailpit and load a car page; verify that the terminal shows one access line per request with route templates and request IDs, that no magic-link token appears, and that `LOG_REQUESTS=errors` in `backend/.env` suppresses the 200 lines.
