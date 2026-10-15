# Legacy Parity Checklist

This file is the acceptance criterion for parity with the legacy app for
every rewrite slice (`api-contract-v1`, `db-schema-v1`,
`auth-oidc-magic-link`, each feature vertical, ...). If a piece of legacy
behaviour is not listed below and not listed in [Non-goals](#non-goals), it
is not yet a decided parity requirement — cite the item you're implementing
by id, or open a proposal to add a missing one.

Derived from the legacy Next.js app at legacy commit `774a37f`, vendored
into this repository in commit `bbd18eb`. The `old/` folder has since been
deleted; the `old/...` paths cited below can be read with
`git show bbd18eb:old/<path>`. This document is what survives it.

## Id conventions

- `SCR-nn` — a screen (page) in the legacy frontend.
- `EP-nn` — a method-and-path pair served by a legacy API handler.
- `DRV-nn` — a value the legacy app computes rather than stores.
- `NG-nn` — a non-goal: something in the legacy app that is deliberately not a
  parity requirement, with the reason why.
- `ADD-nn` — an intentional addition: behaviour the legacy app does not have,
  adopted through its own proposal.

Numbers are assigned once per prefix and never reused, so a citation
("covers `EP-12`, `EP-13`") stays valid as items are added. Later proposals
and tasks should cite ids rather than restating the items below.

## Screens

- **SCR-01** — `/` (`old/pages/index.tsx`). Public landing page. Shows a
  welcome message and four feature cards (Expenses, Refuels, Repairs,
  Tickets); each links to `/home`, `/refuels`, `/repairs`, `/tickets` when
  logged in, or to `/auth/login` otherwise. Links to login and register.
- **SCR-02** — `/home` (`old/pages/home.tsx`). Authenticated dashboard.
  Shows a car-cards summary, buttons to add a new refuel/repair/ticket, and
  an overall-expenses stacked bar chart (fed by `EP-28`) or a "no expenses"
  message if the caller has none. The rewrite charts one calendar year at a
  time (January to December in the viewer's time zone) with previous/next
  year navigation and a car filter, instead of the whole history
  (`dashboard`).
- **SCR-03** — `/profile` (`old/pages/profile.tsx`). Shows a form to edit
  the caller's own user info (email, firstname, lastname; backed by
  `EP-34`/`EP-35`) and a read-only dump of the decoded JWT's claims
  (`iat`/`exp` formatted as dates). The claims dump and email editing are
  not ported — see `NG-07`.
- **SCR-04** — `/auth/login` (`old/pages/auth/login.tsx`). Email + password
  login form (`EP-01`).
- **SCR-05** — `/auth/register` (`old/pages/auth/register.tsx`). Email +
  password registration form (`EP-03`).
- **SCR-06** — `/auth/passwordForgotten`
  (`old/pages/auth/passwordForgotten.tsx`). Email input that requests a
  password-reset link (`EP-04`); links to register.
- **SCR-07** — `/auth/passwordReset/[userId]/[specialToken]`
  (`old/pages/auth/passwordReset/[[...params]].tsx`). New-password form;
  submits `userId`, `specialToken` (from the URL) and the new password to
  `EP-05`.
- **SCR-08** — `/auth/mailVerification/[userId]/[specialToken]`
  (`old/pages/auth/mailVerification/[[...params]].tsx`). No form — on
  mount, POSTs `userId`/`specialToken` (from the URL) to `EP-06` and shows
  the resulting status message.
- **SCR-09** — `/cars` (`old/pages/cars/index.tsx`). List of the caller's
  cars (`EP-07`), columns minus `isActive`/`fin`/`purchaseDate`; link to add
  a new car; each row links to the car's detail and edit screens. The
  rewrite also mutes inactive or deregistered rows (`ADD-03`).
- **SCR-10** — `/cars/create` (`old/pages/cars/create.tsx`). Add-car form
  (type, make, name, fuel, first registration, license plate, FIN, purchase
  date, purchase price); submits to `EP-08`. Only type, make, name and fuel
  are required: first registration and license plate are optional, as in
  the legacy model (`old/db/migrations/20221204230207-create-car.js` allows
  `NULL` for both).
- **SCR-11** — `/cars/[carId]` (`old/pages/cars/[carId]/index.tsx`). Car
  detail with three tabs: **Details** (all car fields, `DataList`),
  **Expenses** (the car's refuels/repairs/tickets as tables — `EP-12`,
  `EP-14`, `EP-16` — filtered to a date range that defaults to the last 6
  months, each with an "add new" shortcut), and **Consumption** (a line
  chart of `DRV-03` over the car's refuel list). "Edit Car" button links to
  `SCR-12`.
- **SCR-12** — `/cars/[carId]/edit` (`old/pages/cars/[carId]/edit.tsx`).
  Edit-car form (same fields as `SCR-10`, submits to `EP-10`) plus a
  two-step-confirm delete-car button (`EP-11`).
- **SCR-13** — `/refuels` (`old/pages/refuels/index.tsx`). A fuel-price line
  chart (`DRV-01` over time, one line per base+sub fuel type) above the list
  of all the caller's refuels (`EP-18`), with a client-side car filter built
  from the cars present in the already-loaded list and a running sum
  (`AmountSum`) of the currently-shown rows' `amount`; each row links to
  `SCR-15`; link to add a new refuel.
- **SCR-14** — `/refuels/create` (`old/pages/refuels/create.tsx`). Add-refuel
  form (car, date/time, station — autocomplete from `EP-22` — odometer
  reading, fuel subtype, liter, amount); submits to `EP-13`; accepts an
  optional `?carId=` to preselect the car.
- **SCR-15** — `/refuels/[refuelId]/edit`
  (`old/pages/refuels/[refuelId]/edit.tsx`). Edit-refuel form (same fields
  as `SCR-14`, car fixed/disabled, submits to `EP-20`) plus a
  two-step-confirm delete-refuel button (`EP-21`).
- **SCR-16** — `/repairs` (`old/pages/repairs/index.tsx`). List of all the
  caller's repairs (`EP-23`) with an `AmountSum` of the currently-shown
  rows; each row links to `SCR-18`; link to add a new repair.
- **SCR-17** — `/repairs/create` (`old/pages/repairs/create.tsx`). Add-repair
  form (car, date, station — autocomplete from `EP-27` — odometer reading,
  type, amount, description); submits to `EP-15`; accepts an optional
  `?carId=`.
- **SCR-18** — `/repairs/[repairId]/edit`
  (`old/pages/repairs/[repairId]/edit.tsx`). Edit-repair form (submits to
  `EP-25`) plus a two-step-confirm delete-repair button (`EP-26`).
- **SCR-19** — `/tickets` (`old/pages/tickets/index.tsx`). List of all the
  caller's tickets (`EP-29`) with an `AmountSum` of the currently-shown
  rows; each row links to `SCR-21`; link to add a new ticket.
- **SCR-20** — `/tickets/create` (`old/pages/tickets/create.tsx`). Add-ticket
  form (car, date, type, location — autocomplete from `EP-33` — amount,
  description); submits to `EP-17`; accepts an optional `?carId=`.
- **SCR-21** — `/tickets/[ticketId]/edit`
  (`old/pages/tickets/[ticketId]/edit.tsx`). Edit-ticket form (submits to
  `EP-31`) plus a two-step-confirm delete-ticket button (`EP-32`).

## Endpoints

All authenticated endpoints require a `Bearer` JWT access token in the
`Authorization` header, verified by `isAuthorizedOrResponse` /
`isAuthorizedForUserOrResponse` (`old/lib/backend/middleware/auth.ts`).
"Scoping" below means what restricts which rows a caller can see or act on.

- **EP-01** — `POST /api/v1/auth/login` (`old/pages/api/v1/auth/login.ts`).
  No auth. Looks up the user by email; requires `isVerified` and a matching
  `hashedPassword`; returns a new access + refresh token pair. No sort order
  (single record).
- **EP-02** — `POST /api/v1/auth/refreshToken`
  (`old/pages/api/v1/auth/refreshToken.ts`). No `Authorization` header;
  instead authorizes via the request body's `refreshToken` matched against
  a stored `RefreshToken` row whose `UserId` equals the body's `userId` and
  whose `validUntil` hasn't passed. No sort order (single record, then
  destroyed and replaced).
- **EP-03** — `POST /api/v1/auth/register`
  (`old/pages/api/v1/auth/register.ts`). No auth. Creates a user; if
  verification mail is enabled, sets `specialToken` (a new UUID) and
  `isVerified: false` and emails the token, otherwise creates the user
  already verified.
- **EP-04** — `POST /api/v1/auth/requestPasswordResetLink`
  (`old/pages/api/v1/auth/requestPasswordResetLink.ts`). No auth. Looks up
  the user by email, issues a new `specialToken` (UUID), emails a reset
  link.
- **EP-05** — `POST /api/v1/auth/resetPassword`
  (`old/pages/api/v1/auth/resetPassword.ts`). No `Authorization` header;
  authorized instead by the body's `specialToken` matching
  `user.specialToken` for the body's `userId`. On success clears
  `specialToken`, sets `isVerified: true`, sets the new hashed password.
- **EP-06** — `POST /api/v1/auth/verifyMail`
  (`old/pages/api/v1/auth/verifyMail.ts`). Same `specialToken`-in-body
  pattern as `EP-05`, scoped to the body's `userId`. On success clears
  `specialToken` and sets `isVerified: true`.
- **EP-07** — `GET /api/v1/cars` (`old/pages/api/v1/cars/index.ts`). Auth
  required. Scoped via `user.getCars()` — only the caller's own cars. No
  explicit sort order (Sequelize default).
- **EP-08** — `POST /api/v1/cars` (`old/pages/api/v1/cars/index.ts`). Auth
  required. Creates a car owned by the caller (`user.createCar(...)`).
- **EP-09** — `GET /api/v1/cars/{carId}` (`old/pages/api/v1/cars/[carId]/index.ts`).
  Auth required, scoped by `isUserAuthorizedForCarOrResponse` (403 if
  `car.UserId !== caller.id`). Single record, no sort order.
- **EP-10** — `PUT /api/v1/cars/{carId}`
  (`old/pages/api/v1/cars/[carId]/index.ts`). Same auth + scoping (checked
  twice: once before the handler branches, once again inside
  `handleUpdate`). Partial update — a field is only overwritten if its new
  value is truthy, so `""`, `0` and `false` are silently ignored.
- **EP-11** — `DELETE /api/v1/cars/{carId}`
  (`old/pages/api/v1/cars/[carId]/index.ts`). Same auth + scoping. Hard
  delete; the database cascades the delete to the car's refuels, repairs
  and tickets (`onDelete: "cascade"` in `old/db/migrations/20230115205539-create-refuel.js`,
  `...205846-create-repair.js`, `...210035-create-ticket.js`).
- **EP-12** — `GET /api/v1/cars/{carId}/refuels`
  (`old/pages/api/v1/cars/[carId]/refuels.ts`). Auth required, scoped by
  car ownership. Sort: `date ASC`. Covered in the rewrite by `GET
  /api/v1/refuels?carId=` (plus `/refuels/chart?carId=`) with an optional
  `from`/`to` range; there is no per-car path (`car-detail-tabs`).
- **EP-13** — `POST /api/v1/cars/{carId}/refuels`
  (`old/pages/api/v1/cars/[carId]/refuels.ts`). Same auth + scoping.
  Creates a refuel under that car.
- **EP-14** — `GET /api/v1/cars/{carId}/repairs`
  (`old/pages/api/v1/cars/[carId]/repairs.ts`). Auth required, scoped by
  car ownership. Sort: `date ASC`. Covered in the rewrite by `GET
  /api/v1/repairs?carId=` with an optional `from`/`to` range; there is no
  per-car path (`car-detail-tabs`).
- **EP-15** — `POST /api/v1/cars/{carId}/repairs`
  (`old/pages/api/v1/cars/[carId]/repairs.ts`). Same auth + scoping.
  Creates a repair under that car.
- **EP-16** — `GET /api/v1/cars/{carId}/tickets`
  (`old/pages/api/v1/cars/[carId]/tickets.ts`). Auth required, scoped by
  car ownership. Sort: `date ASC`. Covered in the rewrite by `GET
  /api/v1/tickets?carId=` with an optional `from`/`to` range; there is no
  per-car path (`car-detail-tabs`).
- **EP-17** — `POST /api/v1/cars/{carId}/tickets`
  (`old/pages/api/v1/cars/[carId]/tickets.ts`). Same auth + scoping.
  Creates a ticket under that car.
- **EP-18** — `GET /api/v1/refuels` (`old/pages/api/v1/refuels/index.ts`).
  Auth required. Scoped via an inner join on `Car` with `Car.UserId =
  caller.id`. Sort: `date ASC`. Returns every refuel across all of the
  caller's cars, unpaged.
- **EP-19** — `GET /api/v1/refuels/{refuelId}`
  (`old/pages/api/v1/refuels/[refuelId].ts`). Auth required. Looked up by
  primary key with no user filter in the query itself, then scoped
  after the fact via `isUserAuthorizedForCarOrResponse(refuel.CarId)` — 403
  if the refuel's car belongs to another user.
- **EP-20** — `PUT /api/v1/refuels/{refuelId}`
  (`old/pages/api/v1/refuels/[refuelId].ts`). Same lookup + scoping as
  `EP-19`. Partial update; falsy values ignored (same pattern as `EP-10`).
- **EP-21** — `DELETE /api/v1/refuels/{refuelId}`
  (`old/pages/api/v1/refuels/[refuelId].ts`). Same lookup + scoping as
  `EP-19`. Hard delete.
- **EP-22** — `GET /api/v1/refuels/stations`
  (`old/pages/api/v1/refuels/stations.ts`). Auth required (any
  authenticated user), but the query is **not** scoped to the caller —
  `Refuel.findAll({ attributes: ["station"] })` reads every user's station
  names. Sort: `station ASC`. See `NG-03`.
- **EP-23** — `GET /api/v1/repairs` (`old/pages/api/v1/repairs/index.ts`).
  Same pattern as `EP-18` for repairs. Sort: `date ASC`.
- **EP-24** — `GET /api/v1/repairs/{repairId}`
  (`old/pages/api/v1/repairs/[repairId].ts`). Same pattern as `EP-19` for
  repairs.
- **EP-25** — `PUT /api/v1/repairs/{repairId}`
  (`old/pages/api/v1/repairs/[repairId].ts`). Same pattern as `EP-20` for
  repairs.
- **EP-26** — `DELETE /api/v1/repairs/{repairId}`
  (`old/pages/api/v1/repairs/[repairId].ts`). Same pattern as `EP-21` for
  repairs.
- **EP-27** — `GET /api/v1/repairs/stations`
  (`old/pages/api/v1/repairs/stations.ts`). Same as `EP-22`, unscoped
  across users. Sort: `station ASC`. See `NG-03`.
- **EP-28** — `GET /api/v1/stats/expenses`
  (`old/pages/api/v1/stats/expenses.ts`). Auth required. Scoped to the
  caller's cars via join. Optional `from`/`to` query params (`date >=
  from`, `date < to`). Returns `{ refuels, repairs, tickets }`, each an
  array of `{ id, date, amount, CarId }`. Sort: `date ASC` per list.
  Covered in the rewrite by `GET /api/v1/stats/expenses`, which requires a
  `from`/`to` instant range and returns one `{ date, kind, amount }` row per
  expense; the browser groups the rows into local months (`dashboard`).
- **EP-29** — `GET /api/v1/tickets` (`old/pages/api/v1/tickets/index.ts`).
  Same pattern as `EP-18`/`EP-23` for tickets. Sort: `date ASC`.
- **EP-30** — `GET /api/v1/tickets/{ticketId}`
  (`old/pages/api/v1/tickets/[ticketId].ts`). Same pattern as `EP-19`/`EP-24`
  for tickets.
- **EP-31** — `PUT /api/v1/tickets/{ticketId}`
  (`old/pages/api/v1/tickets/[ticketId].ts`). Same pattern as `EP-20`/`EP-25`
  for tickets.
- **EP-32** — `DELETE /api/v1/tickets/{ticketId}`
  (`old/pages/api/v1/tickets/[ticketId].ts`). Same pattern as
  `EP-21`/`EP-26` for tickets.
- **EP-33** — `GET /api/v1/tickets/locations`
  (`old/pages/api/v1/tickets/locations.ts`). Same as `EP-22`/`EP-27`,
  unscoped across users. Sort: `location ASC`. See `NG-03`.
- **EP-34** — `GET /api/v1/users/{userId}`
  (`old/pages/api/v1/users/[userId].ts`). Auth required, scoped by
  `isAuthorizedForUserOrResponse` — 401 unless the token's own id equals the
  path `userId`. Returns `{ id, email, firstname, lastname }`.
- **EP-35** — `PUT /api/v1/users/{userId}`
  (`old/pages/api/v1/users/[userId].ts`). Same auth + scoping as `EP-34`.
  Rejects if the new email is already taken by another user; updates
  email/firstname/lastname.

## Derived values

All three are computed client-side in
`old/hooks/fetch/use-refuels-backend.ts`, over a refuel list fetched in
`date ASC` order (`EP-18`/`EP-12`).

- **DRV-01** — `perLiter`. Formula (in `refuelToFrontendRefuel`):
  `Math.round((1000 * refuel.amount) / refuel.liter) / 1000` — 3 decimal
  places. Computed for every refuel row. Displayed as the "per liter" column
  on `/refuels` (`SCR-13`) and used as the Y value of the fuel-price chart on
  `/refuels` (`SCR-13`).
- **DRV-02** — `distance`. Formula (in `refuelListToFrontendRefuelList`):
  walking the list in fetched order, keeping a running map of the previous
  `odometerReading` seen per `CarId`: `distance = odometerReading -
  previousOdometerReadingForThisCar`. Only set once a previous reading
  exists for that car — the first refuel of each car has no `distance`. Not
  displayed directly; feeds `DRV-03`.
- **DRV-03** — `consumption`. Formula: `Math.round((10000 * liter) /
  distance) / 100` — 2 decimal places — only computed when `DRV-02`'s
  `distance` is available for that row. Displayed as the "Consumption"
  column wherever `RefuelList` renders it (`/refuels` `SCR-13`, and the
  Expenses tab of `/cars/[carId]` `SCR-11`), and is the Y value of the
  consumption chart on the Consumption tab of `/cars/[carId]` (`SCR-11`).

### Flagged for `api-contract-v1`

Recorded, not resolved — each is a contract-shaped question that
`api-contract-v1` owns.

1. **Derived refuel fields losing their predecessor across a page
   boundary.** `DRV-02`/`DRV-03` are computed by walking the *whole*
   date-ascending refuel list per car, keeping the previous odometer
   reading in memory. Under cursor pagination, a client holding only the
   current page no longer has the predecessor row once it's past the first
   page, so the first row of every later page would render without a
   `distance`/`consumption` value. Affects `DRV-02`, `DRV-03`, `SCR-13`,
   `SCR-11`. Open question: where does this move — server-side storage,
   a "previous odometer" value returned alongside each page, or something
   else?
2. **Client-side car filter seeing only the loaded page.** `/refuels`
   (`SCR-13`), `/repairs` (`SCR-16`) and `/tickets` (`SCR-19`) build their
   car filter (`ButtonSelect`) from the cars present in the
   already-fetched list (e.g. `refuelListToCarFilterList` in
   `old/pages/refuels/index.tsx`), then filter client-side. Under cursor
   pagination the filter would only ever see cars present in the current
   page. Affects `SCR-13`, `SCR-16`, `SCR-19`. Open question: does
   filtering become a server-side query parameter instead of a client-side
   list filter?
3. **Charts fed the same truncated array as the list.** The fuel-price
   chart (`SCR-13`) and consumption chart (`SCR-11`) are fed the exact same
   list the table renders; the overall-expenses chart on `/home` (`SCR-02`)
   is fed by `EP-28`, itself unpaged today. Paging any of these lists would
   silently truncate the corresponding chart's data. Affects `DRV-01`,
   `SCR-13`, `SCR-11`, `SCR-02`, `EP-28`. Open question: do charts get
   their own unpaged/aggregated endpoint, or does the frontend accumulate
   pages before charting?

## Intentional additions

Behaviour the rewrite adds on top of parity. Each was adopted through its
own OpenSpec change, cited after the item.

- **ADD-01** — Editable active state. The legacy `isActive` flag is stored
  and exposed by `EP-07`–`EP-10` but never shown or editable. The car forms
  (`SCR-10`, `SCR-12`) offer it as a switch, on by default, and the car
  detail Details tab (`SCR-11`) shows it. It only changes how the car's rows
  look (`ADD-03`); it does not hide the car and is independent of
  `ADD-02`. (`car-deregistration-and-info`)
- **ADD-02** — Deregistration date and information. Cars gain an optional
  deregistration date, which may not precede the first registration, and an
  optional multi-line information text (at most 2000 characters). Both are
  editable on `SCR-10`/`SCR-12` and shown on `SCR-11`, information with its
  line breaks. (`car-deregistration-and-info`)
- **ADD-03** — Hiding deregistered cars from selection. A car whose
  deregistration date is today or earlier (viewer's local time zone) is not
  offered as a dashboard card (`SCR-02`), in the dashboard and list car
  filters (`SCR-02`, `SCR-13`, `SCR-16`, `SCR-19`, except a car already
  selected through `carId`), or in the create forms' car field (`SCR-14`,
  `SCR-17`, `SCR-20`, where such a `carId` preselects nothing). Its
  cars-list row, its detail and edit screens, all its entries, and all
  statistics stay unchanged. Rows of an inactive or deregistered car in the
  cars, refuels, repairs, and tickets lists are muted and carry a text
  badge. Filtering is client-side only. (`car-deregistration-and-info`)
- **ADD-04** — Attachments. Cars, refuels, repairs, and tickets can each hold
  up to 10 files: JPEG, PNG, WebP, or GIF images and PDF documents, detected
  from the file content, at most `MAX_UPLOAD_BYTES` (default 10 MB) each.
  Files are added on the create and edit forms (`SCR-10`, `SCR-12`, `SCR-14`,
  `SCR-15`, `SCR-17`, `SCR-18`, `SCR-20`, `SCR-21`); large images are scaled
  down in the browser first. Lists (`SCR-09`, `SCR-13`, `SCR-16`, `SCR-19`,
  and the `SCR-11` Expenses tab) show a clickable attachment indicator, and
  the `SCR-11` Details tab lists the car's files. Files are stored on the
  backend's filesystem or in S3 and are only served to the owning account.
  (`add-attachments`)

## Non-goals

Restated from the Scope decisions table (outside this repository) as a
closed list — each of these is deliberately *not* a parity requirement, with
the reason why.

- **NG-01** — Password login. The legacy app supports email + password
  login (`User.hashedPassword`, `EP-01`). The rewrite's target architecture
  is OIDC + magic link only (`openspec/config.yaml`); password login is
  replaced, not ported.
- **NG-02** — `specialToken`. A single-use UUID stored on `User.specialToken`,
  reused for both password-reset (`EP-05`) and mail-verification (`EP-06`)
  links, compared server-side and cleared after use. Replaced entirely by
  the rewrite's magic-link flow — there's no separate mechanism to port.
- **NG-03** — Unscoped suggestion endpoints. `EP-22`, `EP-27` and `EP-33`
  require auth but return every user's station/location values, not just
  the caller's. This is a scoping bug in the legacy app, not intended
  behaviour — parity is the autocomplete *feature*, not the cross-user data
  leak.
- **NG-04** — Reminders. Mentioned only in old product copy (already
  removed from `README.md` by `bootstrap-build`); no reminder model,
  endpoint, or screen exists anywhere in `old/`. Never built in the legacy
  app, so there is nothing to port.
- **NG-05** — Unpaged lists. Every list-returning endpoint (`EP-07`,
  `EP-18`, `EP-23`, `EP-28`, `EP-29`) returns its entire result set with no
  pagination. The rewrite adopts cursor pagination (`api-contract-v1`);
  parity is about the data and ordering returned, not the absence of
  paging — the tension this creates is the subject of the three flagged
  conflicts above, not this non-goal.
- **NG-06** — Tokens in `localStorage`. The legacy frontend stores the JWT
  access and refresh tokens in `localStorage`
  (`old/lib/frontend/userService.ts`). The target architecture explicitly
  forbids storing tokens/sessions in browser localStorage (`AGENTS.md`);
  the new backend-issued session mechanism replaces this.
- **NG-07** — Profile token claims and email changes. `SCR-03` shows a
  read-only dump of the decoded JWT's claims, and `EP-35` lets the caller
  change their email address. The rewrite has no browser-visible token
  (`NG-06`), so there are no claims to show; the profile screen shows the
  email and the first and last name only. The email address is how OIDC and
  magic-link logins find an account, so letting a user set it without
  verifying it would let them claim an address they do not own; it is
  read-only (`api-contract/profile`) until a verified email-change flow is
  proposed.
