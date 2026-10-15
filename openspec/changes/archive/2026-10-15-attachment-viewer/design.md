## Context

Every place that lists stored attachments renders them through `FileList`
(`frontend/src/attachments/attachment-list.tsx`), whose `OpenLink` is an
`<a target="_blank">` pointing at `GET /api/v1/attachments/{id}/content`. It has
three callers:

```
                         FileList ── OpenLink <a target=_blank>
                            ▲
       ┌────────────────────┼─────────────────────┐
 AttachmentIndicator   AttachmentOverview   AttachmentManager
 (popover in 5 lists)  (car Details tab)    (edit forms, + delete)
```

The download endpoint answers with `Content-Security-Policy: sandbox` and
`Cache-Control: private, no-store`. The sandbox header stops Chromium's
built-in PDF viewer, and Android Chrome has no inline PDF viewer at all. So a
viewer can't simply embed the URL in an `<iframe>`. `no-store` means every
look at a file downloads it again, which stepping back and forth in a viewer
would do constantly.

HeadlessUI's `Dialog` is already used for the mobile menu (`app-shell.tsx`) and
gives focus trapping, Escape handling and scroll locking. The project has no
Content-Security-Policy for its own pages yet.

## Goals / Non-Goals

**Goals:**
- One viewer component replaces the new-tab link in all three callers.
- Images and PDFs are both viewable in place, including on mobile browsers.
- Stepping back and forth doesn't download files again.
- No weakening of the download sandbox, and no attachment content left in the
  browser cache after an explicit logout.

**Non-Goals:**
- Thumbnails or a filmstrip. Listing still loads no file contents.
- Zooming, rotating or annotating inside the viewer. Pages fit the viewer's
  width, and the browser's own page zoom still works.
- Deleting or otherwise changing attachments from the viewer.
- A shareable URL for an open viewer (route state). The viewer is local UI
  state.
- Viewing queued, not yet uploaded files on create forms.

## Decisions

### D1: The viewer is a HeadlessUI `Dialog`, not a lightbox library

`AttachmentViewer` takes the entry's attachment list and a start index and
keeps the current index in state. `Dialog` provides focus trapping, Escape,
backdrop click and the `inert` background. On top of that, we add only
←/→ keys, prev/next buttons, a counter, swipe (a small pointer-events handler:
horizontal movement over a threshold, a clearly more horizontal than vertical
gesture, ignored while a PDF page is being scrolled sideways), and the
header actions.

*Alternatives:* yet-another-react-lightbox, PhotoSwipe. They're polished for
images, but neither renders PDFs, both add their own styling to reconcile with
Tailwind, and the extra pieces we'd write ourselves are small.

### D2: `FileList` gets an `onOpen(index)` callback instead of building links

`FileList` renders a button for each stored attachment and reports the
activated index. Each caller owns its viewer state, because the caller has
the attachment list:

- `AttachmentOverview` and `AttachmentManager` render the viewer next to the
  list.
- `AttachmentIndicator`: the viewer has to outlive the popover, so its state
  lives in `AttachmentIndicator` (outside `PopoverPanel`). On activation, we
  call the popover's `close()` and then set the viewer's state. The attachment
  list is already in the React Query cache (`attachmentListQueryOptions`), so
  the viewer reads it from there and needs no fetch after the popover is gone.
- Focus return: `Dialog` returns focus to the previously focused element. In
  the popover case that element has been unmounted, so we pass the popover
  button as the explicit return target.

Queued files (no `attachmentId`) stay plain text, as today.

### D3: PDFs are rendered with pdf.js (`pdfjs-dist`), loaded on demand

`PdfSlide` does a dynamic `import('pdfjs-dist')`. Vite emits it as its own
chunk, and the worker is bundled through `pdfjs-dist/build/pdf.worker.min.mjs?url`,
so both are same-origin static files built into the backend's static output.
It fetches the bytes itself with `fetch(attachmentContentUrl(id))` (same origin,
session cookie, and it uses the HTTP cache from D5), passes them to
`getDocument({ data })`, and renders one `<canvas>`
per page. A page is drawn only when its placeholder scrolls into view
(`IntersectionObserver`), at the slide's width × `devicePixelRatio`. Text and
annotation layers are left out. Closing the viewer or switching slides calls
`loadingTask.destroy()`.

*Why not an `<iframe>` of the content URL?* Under `CSP: sandbox`, Chromium
shows no PDF, Android Chrome downloads instead of showing, and iOS Safari
shows only the first page. Removing the sandbox for PDFs (the other
alternative) would weaken the download protection and would still fail on
Android. Because pdf.js reads the bytes as data, the download headers can stay
strict.

*Security:* pdf.js had an arbitrary-JS CVE (CVE-2024-4367, fixed in 4.2.67)
in its eval-based font path. We use pdfjs-dist 6, where that path and the
`isEvalSupported` option no longer exist, pin the major version, and use only
the core display API (PDF JavaScript runs only in pdf.js's viewer component
with scripting enabled, which we don't use). The file was already restricted
to the owner's own uploads.

### D4: Image loading and neighbour preloading

`ImageSlide` is an `<img src=contentUrl>` with `object-fit: contain`,
`max-width/max-height: 100%`, and `onError` showing the fallback. For the
slide before and after the current one, we create `new Image()` with the
content URL, so stepping is instant and their responses land in the HTTP cache.
PDF neighbours aren't preloaded, so the pdf.js chunk is only fetched when a PDF
is actually shown.

### D5: `Cache-Control: private, max-age=3600, immutable` on downloads

An attachment id is never reused for other content (there's upload and delete,
no replace), so `immutable` is correct and avoids revalidation requests.
`private` keeps shared proxies out. One hour is enough for a viewing session
and keeps the window short. The header is set only on the `200` content
response. Error responses keep the JSON error defaults, so a `404` is never
cached for an hour.

*Alternative:* keep `no-store` and hold blob URLs in memory while the viewer is
open. That avoids any disk caching, but needs manual revocation, doubles memory
for large images, and doesn't help a file reopened from another list.

### D6: `Clear-Site-Data: "cache"` on successful logout

`logout` in `internal/auth/service.go` adds the header next to
`ClearSessionCookie` on the `204` path only. We ask only for `"cache"`, not
`"cookies"` or `"storage"`. The session cookie is already cleared explicitly,
and clearing storage could remove harmless per-browser preferences.

## Risks / Trade-offs

- [Attachments stay in the browser cache after a session expires or a tab is
  closed without logout] → limited to one hour by `max-age`. The cache is
  per-browser-profile and the files belong to the user. We accept this.
- [`Clear-Site-Data` is only honoured in secure contexts (HTTPS or localhost),
  and Safari's support has been partial] → production is expected to run on
  HTTPS. Where the header is ignored, the one-hour expiry still applies.
- [Chromium clears the whole origin's HTTP cache, including the app's hashed
  JS and CSS chunks] → the next visit after logout downloads the app again.
  Small and rare.
- [pdf.js adds about 1 MB of JS + worker] → loaded only when a PDF is first
  shown, and never part of the entry chunk. We confirm this in the build
  output.
- [pdf.js security history] → pinned current major version (no eval path), core
  API only (no scripting), owner-only files. Renovate/Dependabot-style updates keep it
  current.
- [Swipe conflicting with scrolling a PDF] → swipe needs a mostly horizontal
  gesture. PDF pages fit the width, so horizontal scrolling is rare. The
  buttons always work.
- [Large multi-page PDFs] → pages are rendered only as they scroll into view,
  and the document is destroyed when the user navigates away or closes the
  viewer.

## Migration Plan

Frontend and backend ship together in one build, so there's no separate
rollout order. Older cached responses were `no-store` and don't exist, so
nothing in the cache has to be invalidated. Rollback means reverting the
commit. Files cached with `max-age=3600` simply expire.

## Open Questions

None.
