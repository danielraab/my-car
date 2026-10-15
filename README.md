# my-car

A car management app — vehicles, refuels, repairs, and traffic tickets in
one place. This repository is a ground-up **rewrite** of a legacy Next.js
full-stack app; the legacy code is no longer in the tree and remains
available in git history (vendored in commit `bbd18eb`).

[![License: GPLv3](https://img.shields.io/github/license/danielraab/my-cars)](LICENSE)
![Go](https://img.shields.io/badge/backend-Go-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/frontend-React-149ECA?logo=react&logoColor=white)
![pnpm](https://img.shields.io/badge/package_manager-pnpm-F69220?logo=pnpm&logoColor=white)

## Status

🚧 **Active rewrite, not yet usable in production.** Architecture and
scope are being defined via [OpenSpec](openspec/) change proposals before
implementation. See [`openspec/changes/`](openspec/changes) for what's
currently in flight.

## Architecture

```
┌─────────────────────────┐        REST API        ┌───────────────────────────┐
│   frontend/ (React)     │  ─────────────────────▶ │      backend/ (Go)        │
│   static build only     │  ◀───────────────────── │  REST API + auth + data   │
│   TanStack Router,      │     documented via       │  serves frontend's static │
│   Tailwind, HeadlessUI  │     OpenAPI spec         │  build output in prod     │
└─────────────────────────┘                          └───────────────────────────┘
```

- **Backend** (`backend/`) — Go. Owns the REST API, authentication, and
  business data. In production it also serves the frontend's compiled
  static assets — there is no separate Node server at runtime.
- **Frontend** (`frontend/`) — React, built to static files only (no SSR).
  Talks to the backend exclusively over the REST API described in the
  OpenAPI spec.
- **API contract** — every endpoint is documented in an OpenAPI
  specification, which is the source of truth for request/response
  shapes. Contract changes are reviewed like code.
- **Authentication** — handled entirely by the backend. Two login
  methods are supported, **OIDC** and **magic link** (passwordless
  email); if both resolve to the same email address they log into the
  same account. Sessions/tokens are never stored in browser
  `localStorage`.
- **Internationalization** — the frontend ships with **German (de)** and
  **English (en)** at all times; every user-facing string needs both.

## Tech stack

| Layer     | Choices |
|-----------|---------|
| Backend   | Go |
| Frontend  | React, Vite, TanStack Router, Tailwind CSS, HeadlessUI, lucide-react |
| Package manager (frontend) | pnpm |
| API       | REST, documented with OpenAPI |
| Auth      | OIDC + magic link, backend-issued sessions (no localStorage tokens) |
| Process   | [OpenSpec](https://github.com/Fission-AI/OpenSpec) spec-driven development |

## Repository layout

```
.
├── backend/    # Go REST API, serves frontend static build in production
├── frontend/   # React app, builds to static files only
├── openspec/   # Spec-driven change proposals, specs, and project context
└── AGENTS.md   # Ground rules for AI/human contributors
```

## Getting started

> The backend and frontend are early scaffolding; commands below will
> expand as the rewrite progresses.

**Backend**
```bash
cd backend
go run .
```

**Frontend**
```bash
cd frontend
pnpm install
pnpm dev
```

**Backend tests**

Without a database, tests that need Postgres are skipped:

```bash
cd backend
go test ./...
```

To run them as well, set `DATABASE_URL`. The tests run migrations and
truncate tables, so use a throwaway database rather than your development
one:

```bash
docker run -d --rm --name mycars-test -p 55432:5432 \
  -e POSTGRES_USER=mycars -e POSTGRES_PASSWORD=mycars -e POSTGRES_DB=mycars \
  postgres:17
cd backend
DATABASE_URL='postgres://mycars:mycars@localhost:55432/mycars?sslmode=disable' \
  go test ./...
docker stop mycars-test
```

**Frontend tests and checks**

```bash
cd frontend
pnpm test        # Vitest, once (pnpm test:watch to keep it running)
pnpm typecheck   # TypeScript
pnpm check       # Biome lint and formatting (pnpm format --write to fix formatting)
pnpm check:api   # generated API types match openapi/openapi.yaml
```

If `check:api` fails, regenerate the types with `pnpm generate:api`.

**File storage**

Files attached to cars, refuels, repairs and tickets (JPEG, PNG, WebP and
GIF images and PDF documents, at most `MAX_UPLOAD_BYTES` each, default
10 MB) are stored outside the database:

- `FILE_STORAGE=local` (default) keeps them below `FILE_STORAGE_PATH`
  (default `./data/files`; `/data/files` in the container image). Mount a
  volume there, as `docker-compose.yml` does, or files are lost when the
  container is recreated.
- `FILE_STORAGE=s3` keeps them in an S3-compatible bucket configured by the
  `S3_*` variables (see `backend/.env.example`). Block public access on the
  bucket: files are only ever served through the backend, to their owner.
  The compose `dev` profile runs MinIO on port 9000 with a `my-cars` bucket
  for trying this out.

The backend refuses to start if the storage is not usable. A complete backup
consists of the database **and** the file storage. Storage keys do not
depend on the backend, so moving between local storage and S3 is a manual
copy of all files under the same keys; there is no built-in migration.
To run the S3 integration test against the compose MinIO:

```bash
cd backend
S3_TEST_ENDPOINT=http://localhost:9000 S3_TEST_BUCKET=my-cars \
  S3_TEST_ACCESS_KEY_ID=mycars S3_TEST_SECRET_ACCESS_KEY=mycars-secret \
  go test ./internal/attachments/
```

## Development workflow

This project follows **spec-driven development with [OpenSpec](openspec/)**.
Any non-trivial change — a new feature, endpoint, schema change, or
architectural decision — goes through:

1. **Explore** — think through the problem, investigate the existing
   code, and clarify requirements.
2. **Propose** — capture the decision as a change under
   `openspec/changes/` (proposal, design, specs, tasks).
3. **Apply** — implement against the approved proposal.

Validate the API contract locally with:

```bash
cd frontend
pnpm lint:openapi
```

Small, obvious fixes are exempt. See [`AGENTS.md`](AGENTS.md) and
[`openspec/config.yaml`](openspec/config.yaml) for the full project
context and ground rules.

Parity with the legacy app is the acceptance criterion for the rewrite; see
[`docs/parity-checklist.md`](docs/parity-checklist.md) for the checkable
inventory.

## Contributing

- Follow the workflow above — start with an OpenSpec proposal for
  anything beyond a trivial fix.
- Keep the OpenAPI spec in sync with any backend endpoint change.
- Add both `de` and `en` translations for any new frontend string.

## License

Licensed under the [GNU General Public License v3.0](LICENSE).
