## Why

Cars that were sold, scrapped, or deregistered stay in every car picker and on
the dashboard forever, cluttering day-to-day entry, while their history must
remain visible. The legacy `isActive` flag is stored and exposed by the API but
never shown or editable, and a car has nowhere to keep free-form notes.

## What Changes

- Add a nullable `deregistrationDate` (calendar date) to cars. A car whose date
  is today or earlier is **deregistered**.
- Make the existing `isActive` flag editable through a switch on the car create
  and edit forms (default on) and show it on the car detail screen.
  `isActive` and `deregistrationDate` are independent of each other.
- Add a nullable multi-line `information` text to cars, editable on the car
  forms and shown on the car detail screen.
- Reject a `deregistrationDate` earlier than `firstRegistration` when both are
  set.
- Hide deregistered cars from the dashboard car cards and from every car
  selection: the create-form pickers and the list/dashboard car filters.
  Their cars-list row and all linked refuels, repairs, and tickets stay
  visible.
- Show a muted row with a text badge for a car that is inactive or deregistered
  in the cars list, and for rows of refuels, repairs, and tickets that belong to
  such a car.
- Filtering by deregistration is frontend-only (no API query parameter);
  statistics and totals keep including all cars.
- Record these as intentional additions to `docs/parity-checklist.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `persistence/schema`: cars gain an optional deregistration date and
  information text.
- `api-contract/cars`: car representations, `CarInput` and `CarUpdate` gain
  `deregistrationDate` and `information`; date-order validation.
- `frontend/cars`: list, create, detail, and edit screens expose the active
  switch, deregistration date, and information, and the list mutes inactive or
  deregistered cars.
- `frontend/dashboard`: deregistered cars are not offered as cards or in the
  car selector.
- `frontend/refuels`, `frontend/repairs`, `frontend/tickets`: car pickers and
  filters exclude deregistered cars; rows of inactive or deregistered cars are
  muted.

## Impact

- Backend: migration `000007`, `internal/cars` store and handler, `openapi.yaml`,
  seed and legacy import (new columns stay null).
- Frontend: generated API schema and `api/client.ts`, `cars/car-form.tsx`, car
  detail and list routes, `components/car-select.tsx`, dashboard `home.tsx`,
  refuel/repair/ticket create forms and lists, i18n catalogs (`de`, `en`; the existing localization requirement already covers the new strings).
- Docs: `docs/parity-checklist.md`.
- No breaking API change: both new members are additive; `isActive` is already
  part of the contract.
