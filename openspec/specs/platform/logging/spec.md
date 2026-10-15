# platform/logging Specification

## Purpose
Defines what the backend logs and at which level, what each HTTP request's
access-log entry contains and must never contain, and how log lines of one
request are correlated.

## Requirements

### Requirement: Log level is configurable
The backend SHALL write only log entries at or above the level configured by
`LOG_LEVEL`, one of `debug`, `info`, `warn` or `error`, defaulting to `info`
when unset. Any other value SHALL fail startup with an error naming the
setting.

#### Scenario: Default level
- **WHEN** the backend starts without `LOG_LEVEL`
- **THEN** info-level entries such as rate-limit hits are written and debug-level entries are not

#### Scenario: Raised level hides info entries
- **WHEN** the backend runs with `LOG_LEVEL=warn` and a request is rate limited
- **THEN** no entry is written for the rate-limit hit

#### Scenario: Invalid level
- **WHEN** `LOG_LEVEL` is `verbose`
- **THEN** startup fails with an error naming `LOG_LEVEL`

### Requirement: Requests are access-logged per configured mode
The backend SHALL write one info-level access entry per completed HTTP
request, with method, route template, status, duration and request ID.
`LOG_REQUESTS` selects which requests are logged: `all` (default when
unset), `errors` (status 400 or higher only) or `off`. Any other value SHALL
fail startup with an error naming the setting.

#### Scenario: Every request logged by default
- **WHEN** the backend runs without `LOG_REQUESTS` and serves `GET /api/v1/cars` with status 200
- **THEN** one access entry is written with method `GET`, route `GET /api/v1/cars`, status 200, a duration and the request ID

#### Scenario: Errors-only mode
- **WHEN** the backend runs with `LOG_REQUESTS=errors` and serves one request with status 200 and one with status 404
- **THEN** only the 404 request has an access entry

#### Scenario: Rate-limited request in errors-only mode
- **WHEN** the backend runs with `LOG_REQUESTS=errors` and answers a request with 429
- **THEN** that request has an access entry

#### Scenario: Access log off
- **WHEN** the backend runs with `LOG_REQUESTS=off`
- **THEN** no access entries are written, and responses still carry `X-Request-Id`

#### Scenario: Invalid mode
- **WHEN** `LOG_REQUESTS` is `some`
- **THEN** startup fails with an error naming `LOG_REQUESTS`

### Requirement: Access entries contain no request secrets
An access entry SHALL NOT contain the query string, any request header value,
the request body, or the concrete path of a request under `/api/`. It SHALL
identify the request by its route template. For a request outside `/api/` it
SHALL also contain the concrete path.

#### Scenario: Magic-link consumption
- **WHEN** a request to `GET /api/v1/auth/magic-links/<token>` is access-logged
- **THEN** the entry contains the route `GET /api/v1/auth/magic-links/{token}` and does not contain the token

#### Scenario: OIDC callback
- **WHEN** an OIDC callback request carrying `code` and `state` query parameters is access-logged
- **THEN** the entry contains neither the `code` nor the `state` value

#### Scenario: Session cookie
- **WHEN** an authenticated request is access-logged
- **THEN** the entry does not contain the session cookie value

#### Scenario: Frontend request
- **WHEN** a request for `/cars/abc/refuels` is answered by the frontend fallback and access-logged
- **THEN** the entry contains the path `/cars/abc/refuels`

### Requirement: Responses carry a request ID that correlates log entries
The backend SHALL assign every request a new unique ID, return it in the
`X-Request-Id` response header, and include it in the access entry and in
every entry the backend writes while handling that request. An
`X-Request-Id` sent by the client SHALL be ignored.

#### Scenario: Header on every response
- **WHEN** any request is served, including a 404 and a frontend asset
- **THEN** the response has a non-empty `X-Request-Id` header

#### Scenario: Handler error correlated with access entry
- **WHEN** a handler logs an error while serving a request that then answers 500
- **THEN** the error entry and the access entry carry the same request ID as the response header

#### Scenario: Client-supplied ID ignored
- **WHEN** a request arrives with header `X-Request-Id: abc`
- **THEN** the response's `X-Request-Id` is a newly generated value, not `abc`

### Requirement: Handler panics are answered with 500 and logged
When handling a request panics, the backend SHALL log an error-level entry
with the request ID and answer `500` if no response has been started. The
server SHALL keep serving other requests.

#### Scenario: Panicking handler
- **WHEN** a handler panics before writing a response
- **THEN** the client receives status 500, an error-level entry with the request ID is written, and the access entry records status 500

#### Scenario: Server keeps running
- **WHEN** a handler has panicked
- **THEN** the next request to `/api/healthz` succeeds
