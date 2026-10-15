# Spec Delta

## MODIFIED Requirements

### Requirement: Downloads are served with protective headers
Every download response SHALL carry `X-Content-Type-Options: nosniff`,
`Content-Security-Policy: sandbox` and
`Cache-Control: private, max-age=3600, immutable`, and a `Content-Disposition`
of `inline` with the original filename encoded per RFC 6266, so that images and
PDFs open in the browser but never run in the app's origin, are never stored
in a shared cache, and the requesting browser may reuse a file it already
downloaded for up to one hour without asking again. An attachment's content
never changes under its id, so the response SHALL NOT need revalidation within
that hour.

#### Scenario: Download headers
- **WHEN** an authenticated caller downloads an owned image attachment
- **THEN** the response carries `nosniff`, `sandbox`,
  `private, max-age=3600, immutable` and an inline disposition with the
  original filename

#### Scenario: Error responses are not cached
- **WHEN** a caller requests the content of an attachment that doesn't exist or
  belongs to another account
- **THEN** the `404` response doesn't carry the one-hour `Cache-Control`
