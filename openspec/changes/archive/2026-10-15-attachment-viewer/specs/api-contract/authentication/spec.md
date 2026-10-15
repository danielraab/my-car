# Spec Delta

## ADDED Requirements

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
