# api-contract/authentication Specification

## Purpose

Defines passwordless OIDC and magic-link authentication that converge on one email-based account and establish browser sessions without browser token storage.

## Requirements

### Requirement: Enabled authentication methods are publicly discoverable
The system SHALL document an unauthenticated `GET /api/v1/auth/methods`
operation that returns the enabled passwordless login method codes. The response
SHALL always include `magic_link` and `passkey` and SHALL include `oidc` only
when OIDC is fully configured. It SHALL NOT expose issuer URLs, client
identifiers, client secrets, relying-party settings, or other provider
configuration.

#### Scenario: Magic-link-only deployment reports methods
- **WHEN** a caller requests authentication methods from a deployment without OIDC configuration
- **THEN** the response is `200` and contains only `magic_link` and `passkey`

#### Scenario: OIDC-enabled deployment reports methods
- **WHEN** a caller requests authentication methods from a deployment with complete OIDC configuration
- **THEN** the response is `200` and contains `magic_link`, `passkey`, and `oidc`

### Requirement: Authentication is provided through OIDC and magic links
The system SHALL document an OIDC sign-in initiation route and callback route, a magic-link request route and consumption route, and passkey assertion routes. Password registration, password login, password reset, email verification, bearer access tokens, and refresh tokens SHALL NOT be part of the v1 contract.

#### Scenario: User starts OIDC sign-in
- **WHEN** a browser requests the documented OIDC initiation route
- **THEN** it is redirected to the configured identity provider with state protection

#### Scenario: User requests a magic link
- **WHEN** a user submits a syntactically valid email to the magic-link request route
- **THEN** the API responds without disclosing whether that email already has an account

#### Scenario: User signs in with a passkey
- **WHEN** a browser completes the documented passkey assertion operations with a valid credential
- **THEN** the API responds without a body and sets the documented HttpOnly session cookie

### Requirement: Equivalent verified emails resolve to one account
The system SHALL associate a successfully verified OIDC identity and a consumed magic link with the same account when their normalized verified email addresses match. A newly verified email with no matching account SHALL create one account.

#### Scenario: Existing magic-link user signs in through OIDC
- **WHEN** OIDC returns a verified email matching an existing magic-link account
- **THEN** the resulting session is authenticated as that existing account

### Requirement: Browser authentication uses secure cookie sessions
The system SHALL establish the authenticated browser session with an HttpOnly, Secure, SameSite cookie and SHALL provide documented current-session and logout operations. The contract SHALL not require the frontend to receive or persist a session token in localStorage.

#### Scenario: Successful authentication establishes a session
- **WHEN** an OIDC callback or magic-link consumption succeeds
- **THEN** the response sets the documented HttpOnly session cookie and redirects to the frontend

#### Scenario: User signs out
- **WHEN** an authenticated browser invokes the logout operation
- **THEN** the server invalidates the session and clears its cookie

### Requirement: Passkey login operations are documented
The contract SHALL document unauthenticated `POST /api/v1/auth/passkey/options`
returning WebAuthn request options in their JSON form, and
`POST /api/v1/auth/passkey` accepting the assertion JSON and responding `204`
with the session cookie on success or `401` on failure. Neither operation
SHALL accept or reveal an email address.

#### Scenario: Passkey login options are requested
- **WHEN** an anonymous caller requests passkey login options
- **THEN** the response is `200` with a challenge, the relying-party ID, user verification `required`, and no allowed-credential list

#### Scenario: Passkey assertion fails
- **WHEN** a caller submits an invalid, expired, replayed, or unknown assertion
- **THEN** the response is `401` with the standard error body and no session cookie

### Requirement: Passkey management operations are documented
The contract SHALL document session-authenticated operations to list
(`GET /api/v1/passkeys`), start registration
(`POST /api/v1/passkeys/registration-options`), complete registration
(`POST /api/v1/passkeys`), rename (`PATCH /api/v1/passkeys/{passkeyId}`), and
delete (`DELETE /api/v1/passkeys/{passkeyId}`) the caller's passkeys.

#### Scenario: Passkey is registered
- **WHEN** a fresh session completes registration with a valid attestation response and a name
- **THEN** the response is `201` with the passkey summary

#### Scenario: Passkey summary shape
- **WHEN** passkeys are listed or one is created or renamed
- **THEN** each summary contains `id`, `name`, `createdAt`, nullable `lastUsedAt`, nullable `authenticatorName`, and `backedUp`, and no key material

#### Scenario: Passkey is deleted
- **WHEN** an authenticated caller deletes one of their passkeys
- **THEN** the response is `204`

### Requirement: Reauthentication-required error is distinguishable
The contract SHALL document that registration operations respond `403` with
an error body whose code is `reauthentication_required` when the session is
older than the fresh-login window, so clients can distinguish it from other
authorization failures.

#### Scenario: Stale session starts registration
- **WHEN** a session older than 5 minutes requests registration options
- **THEN** the response is `403` with code `reauthentication_required`

### Requirement: Rate-limited authentication responses are documented
The contract SHALL document a `429` response with a `Retry-After` header and
the standard error body (code `rate_limited`) for the magic-link request and
the passkey login operations, and a redirect to
`/auth/login?error=rate_limited` for magic-link consumption, OIDC start, and
OIDC callback.

#### Scenario: Client reads the magic-link request contract
- **WHEN** a client inspects the magic-link request operation
- **THEN** it documents `429` with `Retry-After` and error code `rate_limited`

#### Scenario: Client reads the OIDC start contract
- **WHEN** a client inspects the OIDC start operation
- **THEN** it documents the redirect to the login route with `error=rate_limited`

### Requirement: Logout clears the browser's cache for the app
A successful logout response SHALL carry `Clear-Site-Data: "cache"`, so that
privately cached responses such as attachment contents don't stay in the
browser after an explicit sign-out. The header SHALL NOT ask the browser to
clear cookies or storage, and a logout that fails SHALL NOT carry it.

#### Scenario: User signs out
- **WHEN** an authenticated browser invokes the logout operation successfully
- **THEN** the `204` response carries `Clear-Site-Data: "cache"` along with
  the cleared session cookie

#### Scenario: Logout without a session
- **WHEN** a request without a valid session invokes the logout operation
- **THEN** the `401` response doesn't carry `Clear-Site-Data`
