## Why

Receipts, invoices, fines and car documents (registration papers, insurance)
currently live outside the app, so an entry's amount can't be checked against
its source document. Users want to keep those files right next to the car,
refuel, repair or ticket they belong to. The legacy app has no file support,
so this is an intentional addition (`ADD-04`).

## What Changes

- Users can attach up to 10 files to each car, refuel, repair and ticket.
  Only images (JPEG, PNG, WebP, GIF) and PDF documents are accepted; the
  server decides the type from the file content, and blocks everything else
  (including SVG and HEIC).
- File contents are stored either on the backend's filesystem (the default)
  or in an S3-compatible bucket, chosen through environment variables
  (`FILE_STORAGE`, `FILE_STORAGE_PATH`, `S3_*`). The maximum file size is set
  by `MAX_UPLOAD_BYTES`, default 10 MB. An unusable storage configuration
  stops the backend at startup.
- New API: list and upload attachments under each owner resource, download
  and delete a single attachment, and read the upload limits. Downloads
  always go through the backend; the store is never reachable directly.
- Attachments are visible only to the account owning the entry. Another
  account's attachment is answered with `404`, like every other resource.
- Car, refuel, repair and ticket representations gain `attachmentCount`.
- Deleting an entry or a car also deletes its attachments and their stored
  files.
- Frontend: an upload component on all create and edit forms. Large images
  are scaled down in the browser before upload, and the component tells the
  user so. On create forms, files are queued and uploaded after the entry is
  saved. On edit forms, they are uploaded or deleted immediately.
- Frontend: lists show a clickable attachment icon with the count, opening a
  popover with the files. The car detail Details tab shows the car's
  attachments. Files are shown with a type icon (image or document); there are
  no thumbnails.
- `docs/parity-checklist.md` records `ADD-04`.

## Capabilities

### New Capabilities

- `platform/file-storage`: storage backend selection (local filesystem or
  S3), its environment variables, upload size limit, startup validation and
  at-rest protection.
- `api-contract/attachments`: attachment endpoints, accepted types, limits,
  owner-only access, download response headers, error codes.
- `frontend/attachments`: the shared upload component (downscaling, limits,
  queueing), the attachment list popover and the file list with type icons.

### Modified Capabilities

- `persistence/schema`: a new attachments record owned by exactly one car,
  refuel, repair or ticket, deleted with its owner.
- `api-contract/cars`: car representations carry `attachmentCount`.
- `api-contract/expenses`: refuel, repair and ticket representations carry
  `attachmentCount`.
- `frontend/cars`: the car forms accept attachments, the cars list shows the
  attachment icon, the Details tab shows the attachments, and the Expenses
  tab tables show the icon.
- `frontend/refuels`, `frontend/repairs`, `frontend/tickets`: the forms accept
  attachments and the lists show the attachment icon.

## Impact

- Backend: migration `000008`, new `internal/attachments` package (store,
  handler, storage backends), `internal/config` (new variables, `.env.example`),
  list/get queries in `internal/cars`, `refuels`, `repairs`, `tickets` (count),
  car and expense delete paths (file cleanup), `main.go` wiring, `openapi.yaml`.
  New dependency: `github.com/minio/minio-go/v7` for S3.
- Frontend: generated API schema, `api/client.ts`, new attachment components,
  all create/edit forms, all list tables, car detail Details tab, i18n
  catalogs (`de`, `en`).
- Ops: `Dockerfile` pre-creates a writable `/data/files`; `docker-compose.yml`
  gets a files volume, the new variables, and a MinIO service in the `dev`
  profile; README notes that a full backup now needs the database and the file
  storage.
- Docs: `docs/parity-checklist.md` (`ADD-04`).
- No breaking API change: `attachmentCount` and the new endpoints are additive.
  Seed data and legacy import create no attachments.
