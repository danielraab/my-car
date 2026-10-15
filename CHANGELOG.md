# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [2.3.0] - 2026-10-15

### Added

- **Attachments**: up to 10 files can be attached to each car, refuel, repair
  and ticket. Only images (JPEG, PNG, WebP, GIF) and PDFs are accepted; the
  server checks the type from the file content. Large images are scaled down
  in the browser before upload. On create forms files are queued and uploaded
  once the entry is saved; on edit forms they are uploaded or deleted right
  away. Lists show an attachment icon with the count, and the car Details tab
  shows the car's attachments. Deleting an entry or car deletes its files too.
  Car, refuel, repair and ticket responses gain `attachmentCount`. Database
  migration `000008` adds the attachment tables.
- **File storage configuration**: attachments are stored on the local
  filesystem (`FILE_STORAGE=local`, `FILE_STORAGE_PATH`, default) or in an
  S3-compatible bucket (`FILE_STORAGE=s3` with the `S3_*` variables).
  `MAX_UPLOAD_BYTES` sets the maximum file size (default 10 MB). An unusable
  storage configuration stops the backend at startup. The container image
  stores files in `/data/files`; mount a volume there and back it up together
  with the database, or uploaded files are lost when the container is
  recreated.
- **Attachment viewer**: opening an attachment shows an in-app viewer instead
  of a new browser tab. It steps through all of the entry's files with
  buttons, the arrow keys or swipe, renders images scaled to fit and PDFs with
  pdf.js (loaded only when a PDF is first opened), and offers open-in-new-tab
  and download. If a PDF can't be rendered, it offers to open it in a new tab.
- **Request logging**: `LOG_LEVEL` (`debug | info | warn | error`, default
  `info`) sets the minimum level of all backend log output. `LOG_REQUESTS`
  (`all | errors | off`, default `all`) selects which HTTP requests get an
  access-log line with method, route template, status, duration and request
  ID; it never contains query strings, headers, bodies or concrete `/api/`
  paths. Every response carries an `X-Request-Id` header, and handler log
  lines carry the same ID. Invalid values fail startup.
- Development: `dev.sh` starts the compose `dev` profile, the backend and the
  Vite dev server in one go.

### Changed

- **Fuel-price chart**: draws one line per car fuel and refuel fuel pair
  (e.g. "Diesel · Premium") instead of one line per refuel fuel, matching the
  legacy app. The car's fuel picks the line's hue and the refuel's fuel its
  shade, so each pair keeps the same colour across cars and date ranges. The
  chart waits for the car list as well as the chart data and shows the
  retryable error if either fails.
- The refuels table shows the date without the time of day.
- Attachment downloads are privately cacheable for one hour
  (`Cache-Control: private, max-age=3600, immutable`), and logout sends
  `Clear-Site-Data: "cache"` so cached files don't stay in a shared browser.
- All backend logging goes through `log/slog`. The log format changes from
  `log`'s date prefix to `slog`'s key=value text; anything parsing the old
  format has to adapt.
- A panic in a request handler is logged and answered with `500` instead of
  resetting the connection.
- A download whose stored file is missing is answered with `404` and logged
  as a warning, instead of a generic server error.
- Development: the compose `dev` profile uses DbGate (port 8082, with a
  preconfigured connection) instead of pgAdmin. `backend/.env.example` now
  points `AUTH_BASE_URL` at the Vite dev server (`http://localhost:3000`).

### Security

- Rate-limit log entries name the endpoint by its route template instead of
  the concrete path, so a throttled magic-link request no longer writes its
  still-valid token to the log.

## [2.2.0] - 2026-10-13

### Added

- **Car deregistration**: a car can carry an optional deregistration date. A
  car whose date is today or earlier counts as deregistered and is no longer
  offered on the dashboard cards, in the dashboard car selector, or in the car
  pickers and filters of cars, refuels, repairs and tickets. Its cars-list row
  and all linked entries stay visible. A deregistration date earlier than the
  first registration is rejected. Database migration `000007` adds the column.
- **Active switch**: the existing `isActive` flag is now editable on the car
  create and edit forms (on by default) and shown on the car detail screen. It
  is independent of the deregistration date.
- **Car information**: a car can hold a free-form multi-line text, editable on
  the car forms and shown on the detail screen.
- **State badges**: rows of a car that is inactive or deregistered are muted
  and carry a text badge, in the cars list and in the refuel, repair and ticket
  lists.
- README: how to run the backend tests (with and without a database) and the
  frontend tests, type check, lint and API-schema check.

### Changed

- Dropped the "rewrite in progress" status badge from the README.

## [2.1.0] - 2026-10-08

### Added

- **Passkey sign-in**: users can sign in with a passkey (WebAuthn
  discoverable credential) and add, rename and remove passkeys on the profile
  page. Adding one requires a sign-in within the last 5 minutes; removing one
  also ends every session established with it. A passkey belongs to the
  account that registered it and never creates an account. The relying party
  is derived from `AUTH_BASE_URL`, so changing the deployment's domain
  invalidates registered passkeys. Database migration `000006` adds the
  passkey tables.
- **Sign-in rate limiting**: the public sign-in endpoints (magic-link request
  and navigation, OIDC, passkey login) are rate-limited per client address,
  and each email address receives at most 3 magic links per 15 minutes.
  Limited requests get `429` with `Retry-After` (JSON) or a localized message
  on the login page. Behind a reverse proxy, set `TRUSTED_PROXIES` so the
  client address is taken from `X-Forwarded-For`; `RATE_LIMIT_PER_IP=false`
  turns per-client limiting off. See the backend README for a Traefik setup.
- **App icon**: the header's gauge brand mark is the favicon and the icon of
  the installed app.
- **Installable app**: a web app manifest lets browsers install the app to a
  home screen or dock (standalone window, opening at `/home`). There is no
  service worker and no offline support. Known limitation: on iOS an installed
  app keeps its own cookies, so a magic link opened from the mail app signs in
  Safari rather than the installed app; sign in with OIDC there instead.
- **Analytics snippet**: the optional `ANALYTICS_SNIPPET` environment variable
  holds raw HTML (for example an Umami tracker tag) that the backend inserts
  before `</head>` on every page. Unset, pages are unchanged.
- **GitHub link**: the landing page footer and the app sidebar link to the
  GitHub repository.

### Changed

- The product is now called "My cars" in the tab title, header, localized
  texts and the magic-link email.
- CI: bumped the Docker build actions; image builds no longer fail when the
  GitHub Actions cache export is refused.

## [2.0.0] - 2026-10-06

First release of the ground-up rewrite of the legacy Next.js app: a Go
backend with a REST API and a statically built React frontend. The legacy
app is kept in `old/` as reference only.

### Added

- **Backend** (Go): REST API under `/api/v1` documented by an OpenAPI 3.1
  spec, PostgreSQL persistence with the initial domain schema, health check,
  and serving of the frontend's static build in production.
- **Authentication**: passwordless sign-in by magic link and by OIDC, both
  resolving to the same account when the email address matches. OIDC and
  SMTP authentication are optional. Sessions are cookie-based.
- **Profile**: view and manage the caller's profile (`/api/v1/me`).
- **Cars**: list, create, edit and delete cars, with a detail view and
  Expenses and Consumption tabs.
- **Refuels, repairs and tickets**: full management screens and endpoints,
  with car filter and date-range filtering on each list.
- **Dashboard**: monthly expense chart with value popover, keyboard access
  and gridlines.
- **Legacy data import**: import of data from the legacy app (see
  [docs/legacy-import.md](docs/legacy-import.md)); car registration fields
  are nullable to accommodate legacy data.
- **Seed command**: CLI command to generate multi-year test data.
- **Tooling**: Docker image and compose setup, CI with selective jobs per
  changed path, and OpenSpec-driven specs under `openspec/`.
- **Parity checklist**: [docs/parity-checklist.md](docs/parity-checklist.md)
  inventories every legacy screen, endpoint and derived value as the
  acceptance criterion for the rewrite.

[Unreleased]: https://github.com/danielraab/my-cars/compare/v2.3.0...HEAD
[2.3.0]: https://github.com/danielraab/my-cars/releases/tag/v2.3.0
[2.2.0]: https://github.com/danielraab/my-cars/releases/tag/v2.2.0
[2.1.0]: https://github.com/danielraab/my-cars/releases/tag/v2.1.0
[2.0.0]: https://github.com/danielraab/my-cars/releases/tag/v2.0.0
