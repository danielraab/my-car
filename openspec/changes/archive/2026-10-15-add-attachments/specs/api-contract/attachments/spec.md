## ADDED Requirements

### Requirement: Caller can list and upload attachments of an owned entry
The system SHALL document `GET` and `POST` on
`/api/v1/cars/{carId}/attachments`, `/api/v1/refuels/{refuelId}/attachments`,
`/api/v1/repairs/{repairId}/attachments` and
`/api/v1/tickets/{ticketId}/attachments`. `GET` SHALL return every attachment
of the entry (at most 10) in the common collection shape, ordered by upload time
and then identifier, with `nextCursor` always `null`. `POST` SHALL accept a
`multipart/form-data` body with exactly one file part named `file` and SHALL
return `201` with the created attachment. An attachment representation SHALL
contain `id`, `filename`, `contentType`, `size` (bytes) and `createdAt`.

#### Scenario: Caller uploads a receipt to a refuel
- **WHEN** an authenticated caller posts a JPEG file to an owned refuel's
  attachments
- **THEN** the response is `201` with an attachment whose `contentType` is
  `image/jpeg` and whose `size` equals the uploaded byte count

#### Scenario: Caller lists a car's attachments
- **WHEN** an authenticated caller requests an owned car's attachments
- **THEN** the response contains that car's attachments only, oldest first, and
  `nextCursor` is `null`

#### Scenario: Upload without a file part
- **WHEN** an authenticated caller posts a multipart body without a `file` part
- **THEN** the response is `400` with `fields.file` set to `required`

### Requirement: Only images and PDF documents are accepted
The system SHALL determine an uploaded file's type from its content, ignoring
the client-supplied content type and filename extension, and SHALL accept only
`image/jpeg`, `image/png`, `image/webp`, `image/gif` and `application/pdf`. Any
other content SHALL be rejected with `415` and the code
`unsupported_media_type`, and nothing SHALL be stored. The stored and returned
`contentType` SHALL be the detected type.

#### Scenario: SVG is rejected
- **WHEN** a caller uploads an SVG file declared as `image/svg+xml`
- **THEN** the response is `415` with code `unsupported_media_type`

#### Scenario: Disguised HTML is rejected
- **WHEN** a caller uploads an HTML document named `receipt.jpg` and declared as
  `image/jpeg`
- **THEN** the response is `415` with code `unsupported_media_type`

#### Scenario: Mislabelled PDF is accepted with its real type
- **WHEN** a caller uploads a PDF document declared as `application/octet-stream`
- **THEN** the response is `201` and the attachment's `contentType` is
  `application/pdf`

### Requirement: Upload size and count are limited
The system SHALL reject a file larger than the configured maximum upload size
with `413` and the code `file_too_large`, without storing it. An entry SHALL
hold at most 10 attachments; an upload to an entry that already has 10 SHALL be
rejected with `409` and the code `attachment_limit_reached`. The limit SHALL
hold under concurrent uploads to the same entry.

#### Scenario: File exceeds the limit
- **WHEN** a caller uploads a file one byte larger than `MAX_UPLOAD_BYTES`
- **THEN** the response is `413` with code `file_too_large` and the entry's
  attachment count is unchanged

#### Scenario: Eleventh file
- **WHEN** a caller uploads a file to an entry that already has 10 attachments
- **THEN** the response is `409` with code `attachment_limit_reached`

#### Scenario: Two uploads race for the last slot
- **WHEN** two uploads to an entry holding 9 attachments are processed at the
  same time
- **THEN** exactly one succeeds and the other is rejected with `409`

### Requirement: Upload limits are readable by the client
The system SHALL document `GET /api/v1/attachments/limits`, returning
`maxFileBytes` (the configured maximum upload size), `maxFilesPerEntry` (10)
and `allowedContentTypes` (the accepted types).

#### Scenario: Client reads the limits
- **WHEN** an authenticated caller requests the upload limits with
  `MAX_UPLOAD_BYTES` unset
- **THEN** the response contains `maxFileBytes` 10485760 and
  `maxFilesPerEntry` 10

### Requirement: Caller can download and delete an owned attachment
The system SHALL document `GET /api/v1/attachments/{attachmentId}/content`,
returning the stored bytes with the stored `contentType`, and
`DELETE /api/v1/attachments/{attachmentId}`, returning `204` and permanently
removing the attachment and its stored file. The file contents SHALL be
reachable only through this download endpoint.

#### Scenario: Caller downloads a PDF
- **WHEN** an authenticated caller requests an owned PDF attachment's content
- **THEN** the response is `200` with `Content-Type: application/pdf` and the
  uploaded bytes

#### Scenario: Caller deletes an attachment
- **WHEN** an authenticated caller deletes an owned attachment
- **THEN** the response is `204`, a following download returns `404`, and the
  entry's `attachmentCount` drops by one

### Requirement: Attachments are accessible only to the owning account
The system SHALL require the session for every attachment operation. An
attachment SHALL belong to the account that owns its entry. Listing or uploading
on an entry owned by another account, or downloading or deleting another
account's attachment, SHALL respond with `404` and SHALL NOT return, create or
delete any file. Requests SHALL identify attachments only by their identifier
and SHALL never carry a storage key or path.

#### Scenario: Caller downloads another account's attachment
- **WHEN** caller B requests the content of an attachment of caller A's refuel
- **THEN** the response is `404` and no file content is returned

#### Scenario: Caller uploads to another account's car
- **WHEN** caller B posts a file to caller A's car's attachments
- **THEN** the response is `404` and nothing is stored

#### Scenario: Caller deletes another account's attachment
- **WHEN** caller B deletes an attachment of caller A's ticket
- **THEN** the response is `404` and the attachment still exists

#### Scenario: No session
- **WHEN** a request without a valid session reaches any attachment endpoint
- **THEN** the response is `401`

### Requirement: Downloads are served with protective headers
Every download response SHALL carry `X-Content-Type-Options: nosniff`,
`Content-Security-Policy: sandbox` and `Cache-Control: private, no-store`, and a
`Content-Disposition` of `inline` with the original filename encoded per RFC
6266, so that images and PDFs open in the browser but never run in the app's
origin or end up in a shared cache.

#### Scenario: Download headers
- **WHEN** an authenticated caller downloads an owned image attachment
- **THEN** the response carries `nosniff`, `sandbox`, `private, no-store` and an
  inline disposition with the original filename

### Requirement: Original filenames are kept safely
The system SHALL store the uploaded file's name for display, reduced to its
last path segment, stripped of control characters and limited to 255
characters. A missing or empty name SHALL be replaced by `file`.

#### Scenario: Filename with directories
- **WHEN** a caller uploads a file named `C:\scans\invoice.pdf`
- **THEN** the attachment's `filename` is `invoice.pdf`
