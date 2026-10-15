# Tasks

## 1. Regression tests

- [x] 1.1 In `backend/internal/auth/ratelimit_test.go`, add a test that throttles `GET /api/v1/auth/magic-links/{token}` through a real `ServeMux`, captures the log with `captureLogs`, and asserts the entry contains the route template but not the token. Verify that the test fails against the current code (`go test ./internal/auth -run <name>`).
- [x] 1.2 Add the same kind of test for a throttled OIDC callback carrying `code` and `state` query parameters, asserting that neither value appears in the log. Verify that it passes against the current code, which only logs the path; it guards the new requirement.

## 2. Fix

- [x] 2.1 In `backend/internal/auth/ratelimit.go`, log `r.Pattern` instead of `r.URL.Path` in both `rate limit exceeded` calls, keeping the field name `path`. Verify that the tests from 1.1 and 1.2 pass and that `go test ./...` and `go vet ./...` are green in `backend/`.
