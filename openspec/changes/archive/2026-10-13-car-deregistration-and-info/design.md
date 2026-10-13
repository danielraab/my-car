## Context

`cars.is_active` already exists (migration 000002), is carried by `Car`,
`CarInput` and `CarUpdate`, and is imported from the legacy data, but no screen
shows or edits it. Pickers load every car through `allCarsQueryOptions`
(`limit: 100`), and the dashboard cards, the list filters (`CarSelect`), and the
create forms all read from that one list. Edit forms already fix the car of an
existing entry, so they never need to offer a deregistered car.

## Goals / Non-Goals

**Goals:**
- Record a deregistration date, an editable active flag, and free-form
  information per car.
- Keep deregistered cars out of selection UIs and the dashboard while
  preserving their cars-list row and all linked entries.
- Visually mark inactive or deregistered cars wherever their rows appear.

**Non-Goals:**
- No server-side filtering (`?active=`) and no API rule blocking entries for
  inactive or deregistered cars; back-dating stays possible.
- No coupling between `isActive` and `deregistrationDate`.
- No change to statistics, totals, or cars-list ordering.

## Decisions

- **Two independent flags.** `isActive` only drives the muted look;
  `deregistrationDate` drives both the muted look and hiding. *Alternative:*
  derive one from the other, rejected as hidden coupling that the user did not
  want for now.
- **Deregistered means `deregistrationDate <= today`** in the user's local time
  zone, computed in the frontend by one shared helper `isDeregistered(car,
  today)`. A future date therefore flips at local midnight with no job.
  *Alternative:* a backend `?active=true` filter, rejected for now because the
  full car list is already loaded for name lookups and the 100-car cap makes a
  second query pointless.
- **Hiding is scoped to selection.** The helper is applied in the create-form
  pickers, `CarSelect` filters, and dashboard cards/selector. Lists, detail
  screens, and edit forms keep using the unfiltered list. A `?carId=` deep link to
  a deregistered car is left untouched.
- **Muted row = `!isActive || isDeregistered`**, with a text badge so colour is
  not the only signal. Linked-entry lists already resolve the car through the
  all-cars lookup, so they reuse it.
- **Validation lives in the backend and mirrors in the form.** On create both
  dates come from the request. On PATCH an omitted date is taken from the stored
  car, so changing only one of them is still checked. Failure returns `400` with
  `fields.deregistrationDate` set to `before_first_registration` (new reason
  code in `apierror`). Date format reuses `checkDate`.
- **`information`** is nullable `TEXT`, trimmed, blank stored as `null`, limited
  to 2000 characters (`too_long`, an existing reason). It is rendered with
  preserved line breaks.
- **Nullable patch pattern.** Both new members follow `licensePlate`/`fin`: the
  `Nullable` patch type, `null` clears, and OpenAPI uses `type: [string, 'null']`.

## Risks / Trade-offs

- [Car beyond the first 100 is never offered] → pre-existing limitation of
  `allCarsQueryOptions`; unchanged.
- [Hiding by client clock] → a wrong device date mis-hides a car; acceptable for
  a convenience filter, since data stays reachable in the cars list.
- [Deregistered car cannot be used as a list filter] → intentional per the
  request; its entries remain in the unfiltered lists and the car's detail
  Expenses tab.
- [A `?carId=` link to a deregistered car] → the filter selects keep listing the
  car that is currently selected through the URL so the control stays truthful;
  create forms treat an unavailable `carId` as absent.

## Migration Plan

Migration `000007` adds `deregistration_date DATE` and `information TEXT`
(both nullable, no backfill) and a check that `information` is not blank when
set. The down migration drops both columns. Existing rows keep `is_active`
unchanged. Deploy backend and frontend together; the new members are additive,
so an old client keeps working.
