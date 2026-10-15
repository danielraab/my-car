## ADDED Requirements

### Requirement: Rate-limit log entries contain no URL secrets
A rate-limit log entry SHALL identify the throttled endpoint by its method
and route template, never by the concrete request path or query string.
This ensures that no magic-link token, OIDC code or state, or any other
secret carried in the URL is written to the log.

#### Scenario: Throttled magic-link consumption
- **WHEN** a client's request to consume a magic link is throttled
- **THEN** the log entry names the endpoint as `GET /api/v1/auth/magic-links/{token}` and does not contain the token from the request

#### Scenario: Throttled OIDC callback
- **WHEN** a client's OIDC callback request carrying `code` and `state` query parameters is throttled
- **THEN** the log entry contains neither the `code` nor the `state` value
