# my-car

A car management app — vehicles, refuels, repairs, and traffic tickets in
one place. This repository is a ground-up **rewrite** of a legacy Next.js
full-stack app; the legacy code is no longer in the tree and remains
available in git history (vendored in commit `bbd18eb`).

[![License: GPLv3](https://img.shields.io/github/license/danielraab/my-cars)](LICENSE)
![Status](https://img.shields.io/badge/status-rewrite--in--progress-yellow)
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
