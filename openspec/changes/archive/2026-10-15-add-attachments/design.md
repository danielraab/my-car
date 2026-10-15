## Context

Entries (cars, refuels, repairs, tickets) live in Postgres. Expenses reference
`cars` with `ON DELETE CASCADE`, and every store scopes its queries to the
session account in SQL (e.g. `car_id IN (SELECT id FROM cars WHERE account_id =
$1)` in `internal/refuels/store.go`), answering a foreign resource with `404`.
Handlers decode JSON bodies capped by `MaxBytesReader(maxBodyBytes)`. Optional
configuration groups (OIDC) are validated in `internal/config`, whose test
keeps `EnvKeys` and `backend/.env.example` in sync. The production image is
distroless and runs as `nonroot` (uid 65532); the session cookie is
`HttpOnly; Secure; SameSite=Lax` on the same origin that would serve files.

## Goals / Non-Goals

**Goals:**
- Attach images and PDFs to the four entry types, stored locally or in S3.
- Make owner-only access a property of the queries, not of the caller's
  diligence.
- Keep stored files and database records consistent enough that a record never
  points at a missing file.

**Non-Goals:**
- No thumbnails or server-side image processing; downscaling happens in the
  browser only.
- No presigned URLs, public buckets or CDN delivery.
- No built-in migration between storage backends, no per-account quota beyond
  10 files per entry, no virus scanning.
- No sweeper that diffs storage against the table (see Risks).
- Seed data and legacy import create no attachments.

## Decisions

### One `attachments` table with four nullable foreign keys

```
attachments
  id            uuid pk
  account_id    uuid not null → accounts  (RESTRICT)
  car_id        uuid → cars     ON DELETE CASCADE
  refuel_id     uuid → refuels  ON DELETE CASCADE
  repair_id     uuid → repairs  ON DELETE CASCADE
  ticket_id     uuid → tickets  ON DELETE CASCADE
  CHECK (num_nonnulls(car_id, refuel_id, repair_id, ticket_id) = 1)
  storage_key   text not null unique
  filename      text not null
  content_type  text not null   (CHECK in the five allowed types)
  size_bytes    bigint not null CHECK (> 0)
  created_at    timestamptz not null default now()
  partial indexes on each FK WHERE col IS NOT NULL
```

*Alternative:* a polymorphic `(owner_type, owner_id)` pair. Rejected, because the
database could no longer enforce referential integrity or cascade, and every
ownership check would branch on the type.

`account_id` is denormalized on purpose. It is copied from the owner within the
insert (`INSERT ... SELECT ... FROM <owner> JOIN cars ... WHERE account_id =
$caller`), so download and delete check a single `WHERE id = $1 AND account_id
= $2`. Cars never change account, so it can't drift.

### Storage behind a small interface

```go
type BlobStore interface {
    Put(ctx, key string, r io.Reader, size int64, contentType string) error
    Get(ctx, key string) (io.ReadCloser, error)
    Delete(ctx, key string) error
    Check(ctx) error   // startup probe
}
```

`localStore` writes to a temporary file in the same directory, then renames it
into place (0600 file, 0700 directories). It resolves `filepath.Join(base, key)`
and rejects anything outside `base`. `s3Store` uses `minio-go/v7`. *Alternative:*
`aws-sdk-go-v2`, rejected as a much heavier dependency for four operations.
minio-go works against AWS, MinIO, Hetzner, R2 and others. Keys are
`{accountId}/{attachmentId}`, with `S3_PREFIX` prepended for S3 only.

### Downloads are proxied through the backend

`GET /api/v1/attachments/{id}/content` checks ownership, then streams
`BlobStore.Get` to the response. *Alternative:* presigned S3 redirects. Rejected
because a presigned link works for anyone holding it until it expires, there is
no equivalent for local storage, and receipt-sized files don't need offloading.
Plain `<a href>` links work because the session cookie is sent on same-origin
navigation.

### Upload handling

1. `RequireSession`, then `http.MaxBytesReader(MAX_UPLOAD_BYTES + multipart
   overhead)` on the body. Read the `file` part with `multipart.Reader`,
   streaming instead of `ParseMultipartForm`, so nothing is buffered beyond the
   first bytes.
2. Read the first 512 bytes and run `http.DetectContentType`. Reject with `415`
   unless the type is in the allowlist. Continue streaming the sniffed prefix
   followed by the rest, counting bytes, and abort with `413` above the limit.
3. In one transaction: lock the owner row (`SELECT ... FOR UPDATE` scoped to the
   caller's account, `404` if not found), count its attachments (`409` at 10),
   `Put` the blob, insert the row, commit. If anything after `Put` fails, delete
   the blob best-effort.

Holding the row lock across `Put` serialises uploads to the same entry only,
which is acceptable for at most 10 files. *Alternative:* `Put` before the
transaction and count afterwards. Rejected, since a race would briefly allow 11
files and needs compensation.

`http.DetectContentType` covers JPEG, PNG, GIF, WebP and PDF. Everything else,
including `image/svg+xml` (sniffed as text/XML) and HEIC, falls outside the
allowlist.

### Deletion: a trigger queues files, a cleaner removes them

```
DELETE attachment (direct, or cascaded from an entry or car)
  └─ AFTER DELETE trigger ──▶ INSERT storage_key INTO attachment_file_deletions
                                (same transaction as the delete)
Cleaner (every minute, and kicked after an attachment delete):
  for each queued key: BlobStore.Delete(key) ──▶ DELETE queue row
  first failure: log, stop, retry next run
```

The record and its queue entry are committed together, so a record never
points at a missing file, and a queued file stays queued until its removal
succeeds. Cascades from car, refuel, repair or ticket deletes are covered
without changing those packages. `TRUNCATE` fires no row triggers, so the
seed command queues the files of the attachments it truncates explicitly.

*Alternative:* collect the storage keys inside each entry's delete
transaction and remove the files after commit. Rejected during
implementation: it touches all four delete paths, and a crash or failed
removal after commit leaves orphan files that nothing retries.

### Response headers on download

`Content-Type` (stored), `Content-Length`, `X-Content-Type-Options: nosniff`,
`Content-Security-Policy: sandbox`, `Cache-Control: private, no-store`,
`Content-Disposition: inline; filename="<ascii fallback>"; filename*=UTF-8''<pct>`.
`sandbox` gives a top-level PDF or image document an opaque origin, so even a
hostile PDF can't read the app's origin.

### `attachmentCount` on representations

List and single-entry queries add a correlated `(SELECT count(*) FROM
attachments a WHERE a.refuel_id = r.id)`, served by the partial FK index. Cars
count only `car_id`, not their expenses' attachments.

### Configuration

Validated in `internal/config` alongside the existing groups. `FILE_STORAGE`
defaults to `local`, `FILE_STORAGE_PATH` to `./data/files`, `MAX_UPLOAD_BYTES`
to `10485760`. The `S3_*` group is only required with `FILE_STORAGE=s3`. The
startup probe runs in `main.go` after config and before the server listens.
The Dockerfile creates `/data/files` owned by 65532, and `docker-compose.yml`
sets `FILE_STORAGE_PATH=/data/files` on a named `files` volume. A `minio`
service plus bucket init is added to the `dev` profile for exercising S3.
`backend/data/` is git-ignored.

### Frontend

- `attachments/` module: `AttachmentUploader` (with `queue` and `immediate`
  modes), `AttachmentList` (icon + name + size + open, optional delete),
  `AttachmentPopover` (HeadlessUI `Popover`, fetches on open via TanStack
  Query), `downscaleImage(file)`, and the limits query (cached for the session).
- Downscaling: `createImageBitmap(file, { imageOrientation: 'from-image' })`,
  `OffscreenCanvas` (or `<canvas>` fallback), `convertToBlob({ type, quality:
  0.85 })` for JPEG/WebP, PNG kept as PNG. Re-encoding drops EXIF, including GPS
  location; the hint mentions scaling, not the privacy effect.
- Uploads use `fetch` with `FormData`. For per-file progress, `XMLHttpRequest`
  is acceptable inside the uploader only; the generated client isn't used for
  multipart.
- Create flow: the form's `onSubmit` stays unchanged. The route's mutation
  `onSuccess` gets the new id, uploads the queue sequentially, then navigates
  either as before or to the edit screen with a failed-files message in router
  state.
- Icons: lucide `Paperclip` (indicator), `FileImage` (image types), `FileText`
  (PDF). The list column sits at the row end, so it doesn't collide with
  `CarStateBadge`.
- After uploads or deletes, invalidate the entry's attachments query and the
  list and detail queries, so `attachmentCount` refreshes.

## Risks / Trade-offs

- [Orphan files] → deletions are queued transactionally and retried, so
  orphans only arise when an upload's store succeeds but its record is
  confirmed absent and the immediate cleanup also fails (logged). A sweeper
  that diffs storage against the table can be added later without schema
  changes.
- [Cleaner runs in every backend instance] → removing a file is idempotent;
  concurrent instances may try the same key, which is harmless.
- [Local storage without a volume loses files on container recreate] →
  documented in compose and README; the image pre-creates `/data/files` so a
  mounted volume inherits correct ownership.
- [Database and file backups drift apart] → README states that both must be
  backed up together; orphans are tolerated, and a restored database newer than
  the files would produce `404`/`500` on download of the missing ones (logged).
- [`http.DetectContentType` misses an exotic but valid image variant] →
  rejected with a clear `415`; the allowlist can be widened later.
- [Row lock held during a slow S3 `Put`] → only blocks other uploads to the same
  entry; the upload size is bounded.
- [Client-side downscaling fails, e.g. canvas memory on old phones] → fall back
  to the original file, which is then subject to the size limit.
- [PDF opened inline could carry scripts] → served under
  `Content-Security-Policy: sandbox` on an opaque origin with `nosniff`.

## Migration Plan

Migration `000008_attachments` creates the table, indexes, the
`attachment_file_deletions` queue and its trigger; the down migration drops
them (it does not delete stored files). Existing deployments get
local storage at `./data/files` by default. Container deployments should mount
a volume at `/data/files` before uploading. Rollback is to redeploy the previous
image and run the down migration; stored files can be removed manually.

## Open Questions

None blocking. An orphan sweeper and per-account quotas are possible
follow-ups.
