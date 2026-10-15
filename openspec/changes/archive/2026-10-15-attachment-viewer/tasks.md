## 1. Backend headers and contract

- [x] 1.1 Change the download response in `internal/attachments/handler.go` to `Cache-Control: private, max-age=3600, immutable` (`200` path only), and update the header assertion in `handler_test.go`, including a check that a `404` content response doesn't carry it
- [x] 1.2 Add `Clear-Site-Data: "cache"` to the successful logout response in `internal/auth/service.go`, with tests that the `204` carries it and a `401` logout doesn't
- [x] 1.3 Update the download and logout header documentation in `openapi/openapi.yaml` and the synced `backend/openapi.yaml`, run `pnpm lint:openapi`, regenerate `frontend/src/api/schema.gen.ts` (`pnpm generate:api`), and confirm `pnpm check:api` passes

## 2. Viewer shell

- [x] 2.1 Create `src/attachments/attachment-viewer.tsx`: a HeadlessUI `Dialog` taking the attachment list, a start index and an optional focus-return element, with the current-index state, header (file name, "n / total", open-in-new-tab, download with `download={filename}`, close), prev/next buttons disabled at the ends, ←/→ keys, no wrap-around
- [x] 2.2 Add swipe navigation (pointer events, horizontal threshold, gesture clearly more horizontal than vertical)
- [x] 2.3 Add the viewer styles in `styles.css` (full-screen on mobile, backdrop, slide area, header/actions), matching the existing `mobile-dialog` patterns

## 3. Slides

- [x] 3.1 Create `ImageSlide`: `<img>` from `attachmentContentUrl`, fitted with `object-fit: contain` and never enlarged, loading state, `onError` fallback with an open-in-new-tab action
- [x] 3.2 Preload the previous and next image attachments with `new Image()` (no PDF neighbours)
- [x] 3.3 Add `pdfjs-dist` (pinned current major version) and create `PdfSlide`: lazy `import()`, worker via `?url`, `fetch` the content, `getDocument({ data })` (v6 has no eval path), one canvas per page drawn when it scrolls into view at slide width × `devicePixelRatio`, `destroy()` on unmount, loading state and fallback on failure
- [x] 3.4 Run `pnpm build` and confirm pdf.js and its worker are in separate chunks, not in the entry chunk

## 4. Wire into the lists

- [x] 4.1 Change `FileList` in `attachment-list.tsx`: stored attachments become buttons calling `onOpen(index)`; queued files stay plain text; remove the `target="_blank"` link
- [x] 4.2 `AttachmentOverview` (car Details tab) and `AttachmentManager` (edit forms) open the viewer from their lists; the viewer offers no delete
- [x] 4.3 `AttachmentIndicator`: keep the viewer state outside `PopoverPanel`, close the popover and then open the viewer, read the list from the query cache, and return focus to the popover button on close

## 5. Localization

- [x] 5.1 Add the viewer strings (position, previous/next, open in new tab, download, close, dialog label, loading, image/PDF fallback messages) to `de` and `en` in `src/i18n/resources.ts`, and drop strings that are no longer used (e.g. the old "open in new tab" title, if replaced)

## 6. Tests and checks

- [x] 6.1 Update `attachments.test.tsx`: activating a file opens the viewer instead of linking out; queued files aren't openable; popover closes and viewer opens; no delete in the viewer opened from the manager
- [x] 6.2 Viewer tests: start index, "n / total", arrow keys and buttons, disabled ends, single attachment, Escape closes and returns focus, download action has the filename, image error shows the fallback
- [x] 6.3 `PdfSlide` tests with pdf.js mocked: no import before a PDF is shown, rendering failure shows the fallback
- [x] 6.4 Update the car detail test (`-car-detail.test.tsx`) for the Details tab opening the viewer, and the list tests that asserted new-tab links
- [x] 6.5 Check in German that every viewer label is translated
- [x] 6.6 Run backend tests, `pnpm test`, `pnpm lint`, `pnpm build`, and check manually in a desktop browser and a mobile viewport: image stepping without repeated downloads (network panel), a multi-page PDF, swipe, and logout clearing the cache
