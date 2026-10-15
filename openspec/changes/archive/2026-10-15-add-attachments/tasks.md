## 1. Configuration

- [x] 1.1 Add `FILE_STORAGE`, `FILE_STORAGE_PATH`, `S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_USE_PATH_STYLE`, `S3_PREFIX`, `MAX_UPLOAD_BYTES` to `internal/config` (`EnvKeys`, defaults, validation: backend name, required S3 group, absolute `S3_ENDPOINT`, boolean path style, positive integer limit)
- [x] 1.2 Document the variables in `backend/.env.example` and extend `config_test.go` (defaults, each invalid value, missing S3 member, S3 vars ignored for local)

## 2. Storage backends

- [x] 2.1 Create `internal/attachments` with the `BlobStore` interface and key builder (`{accountId}/{attachmentId}`)
- [x] 2.2 Implement `localStore` (create dir 0700, temp file + rename, files 0600, reject paths outside base, `Check` probe) with tests including traversal rejection
- [x] 2.3 Add `github.com/minio/minio-go/v7` and implement `s3Store` (prefix, path-style, `Check` via bucket existence), with an in-memory fake for handler tests
- [x] 2.4 Wire storage selection and the startup probe into `main.go`, exiting with an error naming the relevant variables on failure

## 3. Database

- [x] 3.1 Add migration `000008_attachments` (up/down): table with four nullable cascading FKs, exactly-one CHECK, `account_id`, unique `storage_key`, content-type and size checks, partial FK indexes
- [x] 3.2 Extend `db_test.go` for the exactly-one check and cascade from car, refuel, repair, ticket

## 4. Attachments API

- [x] 4.1 Document in `openapi/openapi.yaml` (and the synced `backend/openapi.yaml`): `Attachment` schema, list/upload under the four owners, `GET /attachments/{id}/content`, `DELETE /attachments/{id}`, `GET /attachments/limits`, error codes `unsupported_media_type`, `file_too_large`, `attachment_limit_reached`, and `attachmentCount` on `Car`, `Refuel`, `Repair`, `Ticket`
- [x] 4.2 Implement the attachments store: owner-scoped insert with row lock and count, list by owner, get/delete by id and account, key collection for owner deletion
- [x] 4.3 Implement the upload handler: size-capped streaming multipart read, content sniffing against the allowlist, `413`/`415`/`409`/`400` responses, filename sanitizing, blob cleanup on failure
- [x] 4.4 Implement list, download (protective headers, RFC 6266 disposition), delete and limits handlers, and register the routes behind `RequireSession`
- [x] 4.5 Handler tests: each accepted type, SVG/HTML/mislabelled rejection, oversize, 11th file, concurrent last slot, cross-account `404` on all endpoints, `401` without session, download headers, filename sanitizing

## 5. Existing resources

- [x] 5.1 Add `attachmentCount` to list and single-entry queries in `internal/cars`, `refuels`, `repairs`, `tickets`, rejecting it as an unknown input member
- [x] 5.2 Queue stored files of deleted attachments (direct or cascaded from an entry or car) through a database trigger into `attachment_file_deletions`, and remove them with a background cleaner that retries failures; seed queues the files of the attachments it truncates
- [x] 5.3 Tests: counts per resource (car counts only its own), deleting a car with attachments on itself and its expenses removes all files, a failed file removal keeps the entry deleted and is retried
- [x] 5.4 Confirm seed, legacy import and `openapi_sync_test.go` still pass

## 6. Ops and docs

- [x] 6.1 Dockerfile: pre-create `/data/files` owned by uid 65532
- [x] 6.2 `docker-compose.yml`: new variables on `app`, `files` volume at `/data/files`, `minio` service with bucket init in the `dev` profile; git-ignore `backend/data/`
- [x] 6.3 README: storage configuration, backups need database + file storage, manual copy between backends, S3 bucket must block public access
- [x] 6.4 Add `ADD-04` (attachments) to `docs/parity-checklist.md`

## 7. Frontend foundation

- [x] 7.1 Regenerate `frontend/src/api/schema.gen.ts`; add attachment list, upload (`FormData`, progress), delete and limits calls to `api/client.ts`
- [x] 7.2 Implement `downscaleImage(file)` (EXIF orientation, 2000 px long edge, JPEG/WebP quality 0.85, PNG kept, GIF/PDF and small images untouched, fallback to original on error) with unit tests
- [x] 7.3 Add `de` and `en` strings: downscaling hint with types and limits, refusal reasons, upload progress and failure, failed-after-create message, no attachments, delete confirmation, accessible labels

## 8. Frontend components

- [x] 8.1 `AttachmentList`: type icon (`FileImage`/`FileText`), name, size, opens download in a new tab, optional two-step delete
- [x] 8.2 `AttachmentUploader`: multi-file picker (images + PDF, camera capture), limits from the limits query, client-side refusal, hint; `queue` and `immediate` modes
- [x] 8.3 `AttachmentPopover`: `Paperclip` button with count and accessible label, HeadlessUI popover loading the entry's attachments on open
- [x] 8.4 Component tests for refusal reasons, queue add/remove, immediate upload and delete, popover loading

## 9. Screens

- [x] 9.1 Car create/edit forms: uploader in queue/immediate mode; create uploads after save, navigates to detail or to edit with failed-files message
- [x] 9.2 Refuel, repair, ticket create/edit forms: same, navigating to the list on success
- [x] 9.3 Add the attachment indicator column to the cars, refuels, repairs, tickets tables and the car Expenses tab tables
- [x] 9.4 Car Details tab: attachments section with "no attachments" state
- [x] 9.5 Invalidate attachment, list and detail queries after upload/delete so counts refresh
- [x] 9.6 Update route tests (`-cars`, `-car-detail`, `-refuels`, `-repairs`, `-tickets`) for forms, indicator and Details section, in both locales
