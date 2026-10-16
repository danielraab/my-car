## Why

A car can already be `electric`, but its refuels are still recorded and shown
as fuel: quantities in litres or gallons, prices per litre, consumption in
l/100 km, and fuel subtypes `normal`/`premium`/`other` that mean nothing for
charging. The fuel-price chart would even plot €/kWh on the same axis as
€/l. Both `fuel-price-chart-by-car-fuel` and `add-account-display-units`
explicitly deferred electric cars to their own proposal; this is it.

## What Changes

- **Quantity unit follows the car.** A refuel of an electric car is a charge
  measured in **kWh**; refuels of every other car fuel (`diesel`,
  `gasoline`, `other`) keep the account volume unit (litre or gallon). The
  distance unit stays per account, so electric consumption reads "kWh/100 km"
  or "kWh/100 mi". Nothing is converted, and consumption is still computed
  per refuel (per charge) with the existing formula.
- **Screens that mix cars** (`/refuels` with all cars) head the columns
  neutrally ("Quantity", "Unit price", "Consumption") and show the unit in
  every cell, because rows can be in litres, gallons or kWh.
- **Fuel-price chart gets a second value axis.** Fuel lines stay on the left
  axis (price per litre or gallon); electric lines are plotted against a
  right axis in price per kWh, shown only when electric lines are present.
  Electric uses the teal hue already reserved for it, with one shade per
  charging subtype.
- **Electric charging subtypes.** The refuel subtype gains `home`, `ac` and
  `dc`. A non-electric car accepts `normal`, `special` and `other`; an
  electric car accepts `home`, `ac`, `dc` and `other`. The backend rejects a
  subtype that does not fit the car's fuel. Forms only offer the subtypes of
  the selected car.
- **Fuel changes across the electric boundary are blocked while a car has
  refuels.** Changing a car between electric and any non-electric fuel is
  rejected with a field error on `fuel` when the car has at least one refuel;
  changes among `diesel`, `gasoline` and `other` stay allowed. The car edit
  form disables those options and explains why in an inline notice that
  states how many refuels the car has. Car representations gain a
  `refuelCount`.
- **BREAKING (API and schema):** the refuel member `liters` becomes
  `quantity` and the derived `perLiter` becomes `unitPrice`; the database
  column `refuels.liters` becomes `quantity`. The enum value `special` is not
  renamed. The backend and frontend ship together, so only external API
  clients (none known) would be affected.
- The legacy import maps refuels of legacy electric cars to the electric
  subtypes; the test-data seed gains an electric car with charges.
- Non-goals: plug-in hybrids (one car with both litres and kWh); a stored
  unit per refuel; non-volume units for `other` fuel (CNG in kg, LPG);
  averaged or state-of-charge-based consumption; charging losses; separate
  electric charts; tariff or session-fee modelling.
- This is behaviour the legacy app did not have and is recorded as a new
  `ADD-06` entry in `docs/parity-checklist.md`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `api-contract/expenses`: refuel members are renamed (`quantity`,
  `unitPrice`); the subtype accepts the electric values and is validated
  against the car's fuel; chart data uses the renamed members.
- `api-contract/cars`: crossing the electric boundary is rejected while the
  car has refuels; the car representation carries `refuelCount`.
- `persistence/schema`: the refuel subtype enumeration gains `home`, `ac`,
  `dc`; the quantity column is renamed.
- `frontend/display-units`: a refuel's quantity unit depends on its car's
  fuel (kWh for electric), including per-unit price, consumption and form
  inputs; mixed screens show the unit per cell.
- `frontend/refuels`: neutral column headers with per-cell units; the
  fuel-price chart's right axis for electric lines; subtype choices follow
  the selected car.
- `frontend/charts`: line charts can plot series against a second value
  axis on the right.
- `frontend/cars`: the edit form blocks electric-boundary fuel changes with
  an inline notice; the Consumption tab of an electric car is in kWh.
- `platform/legacy-data-import`: refuels of legacy electric cars map to
  electric subtypes.
- `platform/test-data-seeding`: seeded accounts include an electric car.

## Impact

- **Database**: migration `000010` adds `home`, `ac`, `dc` to `refuel_fuel`
  and renames `refuels.liters` to `quantity`.
- **Backend**: `internal/refuels` (rename, subtype-versus-car validation on
  create and update, chart), `internal/cars` (fuel-change guard,
  `refuelCount`), `internal/legacyimport`, `internal/seed`, their tests.
- **OpenAPI** (`backend/openapi.yaml`, `openapi/openapi.yaml`): `Refuel`,
  `RefuelInput`, `RefuelUpdate`, refuel fuel enum, `Car`, new validation
  reasons; regenerate `frontend/src/api/schema.gen.ts`.
- **Frontend**: `lib/display-units.tsx` (unit per car), `refuels/*` (table,
  form, fuel-price chart), `components/line-chart.tsx` (optional second
  axis), `cars/*` (edit form notice, consumption, expenses tab),
  `i18n/resources.ts` (de/en strings for kWh, subtypes, notice, headers).
- **Docs**: `docs/parity-checklist.md` gains `ADD-06`.
