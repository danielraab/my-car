## Why

Amounts are shown as bare numbers ("65,90") with no currency, and volumes and
distances are hard-coded to litres and kilometres in the translations. Users
who pay in another currency or think in gallons and miles cannot make the app
describe their data correctly. Giving each account a currency, a volume unit
and a distance unit lets every screen label values the way the user records
them.

## What Changes

- Each account gets three display settings: a **currency** (any ISO 4217
  code, default `EUR`), a **volume unit** (`liter` or `gallon`, default
  `liter`) and a **distance unit** (`kilometer` or `mile`, default
  `kilometer`). Existing accounts and imported legacy accounts get the
  defaults, so nothing changes for them until they pick something else.
- The settings are **labels only**. Changing one never converts or
  recalculates a stored value: a history entered as EUR, litres and km is
  shown as the same numbers in the new currency and units. The profile screen
  says so next to the settings.
- `GET /api/v1/me` (and the session's embedded profile) returns the three
  settings; `PATCH /api/v1/me` accepts them alongside the names. Invalid codes
  are rejected with `400` and a field error.
- The profile screen gains a currency selector (every ISO 4217 code, with
  localized names) and selectors for volume and distance unit. A save applies
  everywhere without a page reload.
- Every amount in the app shows the account's currency symbol in every cell:
  table rows and totals, car purchase price, dashboard chart values, the
  per-volume price column and the fuel-price chart. The active language
  decides where the symbol goes; amounts always keep 2 fraction digits (3 for
  price per volume), whatever the currency's usual minor unit, so stored
  values are shown exactly.
- Odometer, volume, per-volume price and consumption labels, cells, form
  fields and chart texts use the configured units instead of hard-coded
  "km", "l", "litre" and "l/100 km". Consumption is relabelled as
  `<volume>/100 <distance>` (for example "gal/100 mi"); it is not inverted
  to miles per gallon.
- Form inputs for amounts show the currency as a suffix; volume and odometer
  inputs show their unit.
- Non-goals: currency conversion or unit conversion of any kind; per-car
  currency or units; inverted consumption such as mpg or km/l; separate US
  and imperial gallons; kWh or other units for electric cars. API member
  names (`liters`, `perLiter`, `odometerReading`) keep their names.
- This is a new feature outside `docs/parity-checklist.md` (the legacy app had
  no currency or unit settings), so it is proposed on its own.

## Capabilities

### New Capabilities

- `frontend/display-units`: how the frontend formats amounts, volumes,
  distances, per-volume prices and consumption from the caller's account
  settings, and that these follow a settings change without a reload.

### Modified Capabilities

- `api-contract/profile`: the profile representation gains `currency`,
  `volumeUnit` and `distanceUnit`; the update operation accepts and
  validates them in addition to first and last name.
- `persistence/schema`: accounts persist the three settings with defaults and
  reject values outside the documented sets.
- `frontend/profile`: the profile screen lets the user view and change the
  three settings, with a hint that existing values are not converted.
- `frontend/refuels`: the list's columns are described by the configured
  volume unit instead of litres.
- `frontend/cars`: the Consumption tab charts consumption in the configured
  units instead of litres per 100 km.

## Impact

- **Database**: new migration adding `currency`, `volume_unit` and
  `distance_unit` to `accounts`; legacy import relies on the column defaults.
- **Backend**: `internal/profile` reads, validates and updates the settings;
  the session response picks them up through the shared profile. Currency
  validation needs an ISO 4217 code list (for example
  `golang.org/x/text/currency`, a new dependency).
- **OpenAPI** (`backend/openapi.yaml`): `Profile` and `ProfileUpdate` change;
  descriptions of `liters`, `perLiter`, `odometerReading`, `distance` and
  `consumption` say the values are in the account's units. Regenerate
  `frontend/src/api/schema.gen.ts`.
- **Frontend**: `cars/format.ts` (`formatAmount`) and the other number
  formatters in the refuel table and fuel-price chart; the refuel, repair,
  ticket and car tables, forms and detail screen; dashboard and consumption
  charts; the profile route; many de/en strings in `i18n/resources.ts`.
- **Compatibility**: adding required members to `Profile` is additive for this
  repo's own frontend, which is deployed with the backend.
