# backend

The Go half of the my-car rewrite. It owns the REST API described by
[`openapi/openapi.yaml`](../openapi/openapi.yaml) and, in production, serves the
frontend's static build from the binary itself — there is no Node process at
runtime.

## Authentication

Authentication is passwordless and entirely backend-owned:

- **OIDC** is optional and uses issuer discovery and the authorization-code flow
  with state, nonce, and PKCE. Configure `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`,
  and `OIDC_CLIENT_SECRET` together to enable it; register
  `${AUTH_BASE_URL}/api/v1/auth/oidc/callback` at the provider.
- **Magic links** are sent with `SMTP_FROM`, `SMTP_HOST`, `SMTP_PORT`,
  `SMTP_TLS`, and optional `SMTP_USER`/`SMTP_PASSWORD`. `SMTP_TLS` accepts
  `none`, `starttls`, or `tls`.

- **Passkeys** (WebAuthn discoverable credentials) are always enabled and need
  no extra configuration. A signed-in user adds them from the profile screen;
  adding one requires a login within the last 5 minutes, deleting one is always
  allowed and also ends every session that was established with it.

OIDC and magic links normalize a provider-verified email and resolve it to the
same account. A passkey never resolves through email: it belongs to the account
that registered it and never creates an account. Passwords, bearer JWTs,
refresh tokens, and browser-localStorage credentials are deliberately
unsupported.

The passkey relying party is derived from `AUTH_BASE_URL`: its host (without
port) is the relying-party ID and its origin is the only accepted origin. This
has two consequences:

- Passkeys only work when the app is opened through that exact origin. They do
  not work through the Vite dev server (`localhost:3000`) while `AUTH_BASE_URL`
  points at the backend.
- Changing the deployment's domain invalidates every registered passkey,
  because passkeys are bound to the relying-party ID. Users can still sign in
  with a magic link and register new passkeys.

Successful login sets the opaque `my_car_session` cookie. It is `Secure`,
`HttpOnly`, `SameSite=Lax`, and backed by a revocable server-side session. This
means browser login testing requires HTTPS (browsers treat `localhost`
specially in some contexts, but deployments must terminate TLS).

## Rate limiting

The public sign-in endpoints are rate limited in memory (limits reset on
restart and are per process):

| Endpoints | Limit per client address |
|---|---|
| Magic-link request | 5 per 10 minutes, burst 3 |
| Magic-link link, OIDC start and callback | 30 per minute |
| Passkey login options and assertion | 30 per minute |

On top of that, one email address receives at most 3 magic links per
15 minutes. Further requests get the same `202` as always, but no email is
sent, so the limit reveals nothing about the address. Throttled JSON requests
get `429` with `Retry-After` and error code `rate_limited`; throttled browser
navigations (the magic link, OIDC start and callback) are redirected to
`/auth/login?error=rate_limited`. Every throttled request is logged at `INFO`
level with the limit, endpoint group and client key; a recipient appears only
as the first 12 hex characters of its SHA-256 hash.

The client address is the TCP peer unless that peer is listed in
`TRUSTED_PROXIES` (comma-separated IPs or CIDR ranges). Then the rightmost
`X-Forwarded-For` entry that is not itself a trusted proxy is used. IPv6
clients are grouped by `/64`. Two misconfigurations to avoid:

- **Behind a proxy with `TRUSTED_PROXIES` empty**, every request appears to
  come from the proxy, so all users share one limit and a single client can
  block sign-in for everyone. The log shows it: every key is the proxy's
  address.
- **Trusting more than your proxy** lets clients pick their own address via
  `X-Forwarded-For` and bypass the per-client limits.

If you cannot specify the proxy's addresses, set `RATE_LIMIT_PER_IP=false`.
The backend then logs at startup that per-IP limiting is disabled and only the
per-recipient magic-link limit applies; consider rate limiting in the proxy
instead (for example Traefik's `rateLimit` middleware).

### Traefik

Traefik connects to the app over a Docker network, so its address is a
container IP that can change on restart. Give the shared network a fixed subnet
and trust that subnet:

```yaml
networks:
  web:
    ipam:
      config:
        - subnet: 172.30.0.0/24
```

```bash
TRUSTED_PROXIES=172.30.0.0/24
```

Keep Traefik's `forwardedHeaders.insecure` off (the default): Traefik then
replaces any `X-Forwarded-For` a client sends with the real peer address. If a
CDN or load balancer sits in front of Traefik, list its ranges both in
Traefik's `entryPoints.<name>.forwardedHeaders.trustedIPs` and in
`TRUSTED_PROXIES`. To check the setup, request a few magic links from a phone:
the rate-limit log keys should show its public address, not a `172.x` address.

## Running it

```bash
cd backend
go run .
```

To replace **all application data** in the configured database with sample
data, run from `backend/`:

```bash
DATABASE_URL='postgres://...' go run . seed alice@example.com bob@example.com
```

At least one email address is required. Addresses are normalized and deduplicated;
each account gets 4 cars, 500 refuels, 75 repairs, and 90 tickets spread over
the past five years. The command applies pending migrations and needs only
`DATABASE_URL` (no SMTP, OIDC, or server settings). It deletes existing accounts,
expenses, sessions, and pending login state in one transaction. Do not run it
against a database whose data you want to keep.

The database it will talk to comes from compose:

```bash
docker compose up -d db      # Postgres on localhost:5432
```

Add `--profile dev` for pgadmin (`localhost:8081`) and Mailpit
(`localhost:8025`) alongside it. Mailpit accepts the `.env.example` SMTP
settings and displays delivered magic links in its web UI.

## Importing the legacy data

`import-legacy` copies the legacy MariaDB data, exported with `mysqldump`, into
a **fresh** database. It runs once: it refuses to write if accounts, cars or
expenses already exist, and it writes everything in one transaction. Run a dry
run first. It needs no database and prints only the problems it finds, one per
line, or `no problems found`:

```bash
go run . import-legacy ../dump.sql --dry-run
DATABASE_URL='postgres://...' go run . import-legacy ../dump.sql
```

A real run applies pending migrations, validates the whole dump, writes nothing
if there is any problem, and prints the imported row counts. The mapping
rules (category labels, purchase price `0` → empty, dropped credentials, and
so on) are specified in `openspec/specs/platform/legacy-data-import`.

In production, mount the dump into the app container and run the same binary:

```bash
docker compose --profile prod run --rm -v "$PWD/dump.sql:/dump.sql:ro" \
  app import-legacy /dump.sql --dry-run
```

Nobody should log in between deploying to the empty database and running the
import: a login creates an account, and the database then no longer counts as
fresh. Afterwards, legacy users sign in by magic link or OIDC with their legacy
email address. Delete the dump once it has been imported, because it contains
personal data.


Every variable the service reads is listed in [`.env.example`](.env.example)
with a placeholder value, and the names match what `docker-compose.yml` sets.
All listed variables are required except SMTP username/password and the OIDC
group. Leaving all three OIDC values empty runs in magic-link-only mode. If any
OIDC value is provided, all three are required; the service performs discovery
during startup and fails fast if the provider is unavailable.

Nothing in that file is a real credential, and nothing that is one belongs
there.

## Serving the frontend

The Dockerfile builds `frontend/` with Vite, copies the result into
`static/out/` and compiles it into the binary, so `static/out/` is generated
and git-ignored. Building the backend on its own does not produce it.

### Analytics snippet

Set `ANALYTICS_SNIPPET` to raw HTML, typically the tracker tag of an analytics
tool, to have it included on every page. Single-quote it in `.env` or the
shell:

```sh
ANALYTICS_SNIPPET='<script defer src="https://umami.example/script.js" data-website-id="..."></script>'
```

- At startup the backend inserts the snippet verbatim immediately before the
  first `</head>` of the embedded `index.html`. Every response that serves
  that document (`/`, `/index.html` and every client-side route) carries it;
  `/api/` responses and other static files never do. Trackers that follow
  History API navigations, such as Umami, then count in-app page changes too.
- Unset or empty (including only whitespace) leaves `index.html` unchanged.
- If a snippet is set but the embedded frontend has no `index.html` or it has
  no `</head>`, the backend refuses to start. The startup log only notes that
  a snippet is configured, never its contents.
- The Vite dev server (`pnpm dev`) does not go through the backend, so the
  snippet is absent during frontend development.
- Every visitor, signed in or not, is tracked alike; there is no consent banner
  or opt-out. Choosing a tool that needs no consent (such as cookieless Umami)
  is the operator's responsibility.
- The snippet is not validated. A malformed one breaks the page's `<head>`, and
  a future Content-Security-Policy would have to allow its sources.

## Logging

The backend logs plain `key=value` text to stderr. Two settings control it:

- `LOG_LEVEL` (`debug | info | warn | error`, default `info`) is the minimum
  level of all output. Access-log entries and rate-limit hits are `info`, so
  `warn` or `error` hides them too.
- `LOG_REQUESTS` (`all | errors | off`, default `all`) selects which requests
  get an access-log entry: every request, only those answered with status
  400 or higher (including 429 rate-limit refusals), or none.

An access entry records the method, the route template, the status, the
duration and the request ID:

```text
level=INFO msg=request request_id=3f9c0a1b2d4e5f60 method=GET route="GET /api/v1/cars/{carId}" status=200 duration=2.1ms
```

- API requests are identified only by their route template, never by their
  concrete path, because a path can carry a secret such as the magic-link
  token. Requests outside `/api/` (the frontend) also log their `path`.
  Query strings, headers, cookies and bodies are never logged.
- Every response carries an `X-Request-Id` header. Errors that a handler logs
  while serving the request carry the same `request_id`, so a failed request
  can be traced from the response to its log entries.
- A panicking handler is logged at `error` with its request ID and answered
  with `500`; the server keeps running.
- Docker's healthcheck hits `/api/healthz` regularly, so with `all` it adds
  one line per interval. Use `errors` to keep only problems.

## Conventions

Non-trivial changes go through the OpenSpec `explore → propose → apply`
workflow described in the root [`AGENTS.md`](../AGENTS.md). The OpenAPI document
is the source of truth for request and response shapes: change it in the same
change that changes an endpoint.
