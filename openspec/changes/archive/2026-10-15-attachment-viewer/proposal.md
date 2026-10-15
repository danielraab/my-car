## Why

Opening an attachment today opens its download in a new browser tab, one file
at a time. Checking an entry's receipts means juggling tabs, and on phones a
PDF usually ends up as a download instead of something you can read. Users want
to view an entry's files in place and step through them.

## What Changes

- Frontend: activating an attachment opens a lightweight in-app viewer instead
  of a new tab, wherever attachments are listed (list-row popovers, the car
  Details tab, the edit forms' attachment card). The viewer starts at the
  activated file and steps through all of the entry's attachments in list
  order, using buttons, the arrow keys and swipe. It shows the file name, the
  position (e.g. "2 / 5"), and actions to open the file in a new tab, download
  it, and close the viewer.
- Images are shown scaled to fit. PDFs are rendered in the app with pdf.js,
  loaded only when a PDF is first opened, as scrollable pages. If a PDF can't be
  rendered, the viewer offers to open it in a new tab instead.
- The viewer is read-only: it offers no delete. There are no thumbnails, and
  listing attachments still loads no file contents.
- Opening a file from a list row's popover closes the popover before the viewer
  opens.
- Backend: attachment downloads change from `Cache-Control: private, no-store`
  to `private, max-age=3600, immutable`, so stepping back and forth doesn't
  download the same file again.
- Backend: a successful logout additionally sends `Clear-Site-Data: "cache"`,
  so cached attachments don't stay in a shared device's browser cache after
  sign-out.

## Capabilities

### New Capabilities

None. The viewer is part of `frontend/attachments`.

### Modified Capabilities

- `frontend/attachments`: opening an attachment shows the in-app viewer instead
  of a new tab; new requirements for viewer navigation, image and PDF display
  with fallback, and the viewer's read-only actions; the popover closes before
  the viewer opens; viewer strings are localized.
- `frontend/cars`: the Details tab's attachments open in the viewer instead of a
  new tab.
- `api-contract/attachments`: download responses become privately cacheable for
  one hour (`private, max-age=3600, immutable`) instead of `no-store`.
- `api-contract/authentication`: the logout operation's response also carries
  `Clear-Site-Data: "cache"`.

## Impact

- Frontend: new viewer components in `src/attachments/`, `FileList` opens the
  viewer instead of linking out, `AttachmentIndicator` closes its popover first,
  i18n catalogs (`de`, `en`), tests. New dependency: `pdfjs-dist`, split into a
  lazily loaded chunk with its worker, so the main bundle doesn't grow.
- Backend: `internal/attachments/handler.go` (Cache-Control) and its test,
  `internal/auth/service.go` (logout header) and its test.
- API: `openapi.yaml` header descriptions for the download and logout
  operations; regenerate `frontend/src/api/schema.gen.ts`. No request or
  response shape changes, so nothing breaks.
- Privacy: after this change, attachment contents may stay in the signed-in
  user's private browser cache for up to an hour. An explicit logout clears
  them; an expired session or a closed tab doesn't (see design).
- Docs: the `ADD-04` wording in `docs/parity-checklist.md` stays accurate; no
  parity id changes.
