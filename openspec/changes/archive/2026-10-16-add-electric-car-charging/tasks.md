## 1. Persistence

- [x] 1.1 Add migration `000010_electric_charging` (up): `ALTER TYPE refuel_fuel ADD VALUE` `home`, `ac`, `dc`; `ALTER TABLE refuels RENAME COLUMN liters TO quantity`
- [x] 1.2 Add the down migration: fail if any refuel uses `home`/`ac`/`dc`, otherwise recreate `refuel_fuel` without them and rename `quantity` back to `liters`
- [x] 1.3 Migration test: existing refuels keep subtype and value under `quantity`; `dc` is storable; `premium` is rejected; down migration round-trips on data without charging subtypes

## 2. API contract

- [x] 2.1 In `openapi/openapi.yaml` and `backend/openapi.yaml`, rename refuel `liters` → `quantity` and `perLiter` → `unitPrice` in `Refuel`, `RefuelInput`, `RefuelUpdate` and the chart response; describe `quantity` as kWh for electric cars and the account volume unit otherwise, and `consumption` as quantity per 100 distance units
- [x] 2.2 Extend the refuel fuel enum with `home`, `ac`, `dc` and document the car-fuel rule; add validation reasons `fuel_mismatch` and `fuel_locked`
- [x] 2.3 Add read-only `refuelCount` to `Car` (required in the representation, rejected in `CarInput`/`CarUpdate`)
- [x] 2.4 Run `pnpm generate:api`, `pnpm lint:openapi`, and the backend OpenAPI sync test

## 3. Backend: refuels

- [x] 3.1 Rename `Liters` → `Quantity` and `PerLiter` → `UnitPrice` in `internal/refuels` store and handler (columns, JSON tags, field validation key `quantity`)
- [x] 3.2 Add the allowed-subtypes table (non-electric: `normal`, `special`, `other`; electric: `home`, `ac`, `dc`, `other`) and accept the new enum values in parsing
- [x] 3.3 On create, load the target car's fuel with `FOR SHARE` in the insert transaction and reject a mismatched subtype with `fields.fuel = fuel_mismatch`
- [x] 3.4 On update with a supplied subtype, check it against the refuel's car's fuel (`FOR SHARE`) and reject a mismatch the same way, leaving the refuel unchanged
- [x] 3.5 Handler and store tests: zero quantity → `fields.quantity = non_positive`; `liters` rejected as undocumented; `unitPrice` of a charge; `dc` on electric accepted; `special` on electric and `home` on diesel rejected; `other` accepted on both; chart consumption of a charge

## 4. Backend: cars

- [x] 4.1 Add `refuelCount` (subquery like `attachmentCount`) to the car columns and representation; reject it as an input member
- [x] 4.2 In the car update, lock the car row (`FOR UPDATE`), and when `fuel` crosses the electric boundary and the car has refuels, reject with `fields.fuel = fuel_locked` without changing anything
- [x] 4.3 Tests for every scenario in `specs/api-contract/cars/spec.md`: gasoline→electric and electric→diesel with refuels rejected, diesel→gasoline allowed, no refuels allowed, unchanged `electric` allowed, `refuelCount` value and rejection

## 5. Backend: import and seed

- [x] 5.1 Legacy import: map refuel subtypes by the mapped car's fuel (`Normal`→`ac`, `Special`→`dc`, `Others`→`other` for electric cars) and write `quantity`; add a test with a legacy electric car
- [x] 5.2 Seed: add an electric model, generate its refuels as charges (15–60 kWh, 0.25–0.80 per kWh, subtypes weighted towards `home`/`ac`), keep 3–5 cars and 500 refuels per account; extend the seed test to check the subtype rule

## 6. Frontend foundations

- [x] 6.1 Regenerate types and replace every `liters`/`perLiter` use (forms, table, charts, tests)
- [x] 6.2 Add `refuelFuelsFor(carFuel)` and default subtype per car fuel in `api/client.ts`
- [x] 6.3 Extend `useUnitFormat()` with `quantity(carFuel)` (symbol, name, per-unit name, consumption symbol, singular) — kWh for electric, account volume unit otherwise
- [x] 6.4 Add de/en strings: kWh unit and long forms, neutral headers (Quantity, Unit price, Consumption), subtype labels Home/AC/DC fast/Other (Zuhause/AC/DC-Schnellladen/Sonstige), `fuel_mismatch` and `fuel_locked` messages, the fuel-lock notice with refuel count
- [x] 6.5 Unit tests for `quantity(carFuel)` and `refuelFuelsFor`

## 7. Frontend: charts

- [x] 7.1 Add `axis?: 'left' | 'right'` to `LineSeries` and optional axis labels in `LineChart`; compute per-axis ticks with `chart-ticks`, project right-axis points with their own scale, draw right labels, keep gridlines on the left scale, and render exactly as today without right-axis series
- [x] 7.2 `LineChart` tests: two value ranges get separate ticks and full height; no right axis when unused; nearest-point selection across axes
- [x] 7.3 Fuel-price chart: electric subtype lines (`home`, `ac`, `dc`, `other`), right axis for electric only when non-electric lines exist, axis unit labels when both shown, popover unit per point
- [x] 7.4 Add teal shades for `electric-home`, `electric-ac`, `electric-dc` (and adjust `electric-other`) in `styles.css`, checked for contrast in light and dark themes
- [x] 7.5 Fuel-price chart tests for the scenarios in `specs/frontend/refuels/spec.md` (mixed fuels and charges, electric-only car, German labels)

## 8. Frontend: screens

- [x] 8.1 `RefuelTable`: always neutral headers; per-cell quantity, unit price and consumption units from the row's car fuel (cars map on `/refuels`, the loaded car on the Expenses tab)
- [x] 8.2 Refuel create/edit form: subtype options and quantity suffix follow the selected (or fixed) car; reset an unfitting subtype on car change (keep `other`); show `fuel_mismatch` at the subtype field
- [x] 8.3 Car edit form: disable boundary-crossing fuel options when `refuelCount > 0`, show the inline notice with the count, and show `fuel_locked` at the fuel field
- [x] 8.4 Consumption tab: kWh per 100 distance units for electric cars
- [x] 8.5 Screen tests: mixed table cells, electric form subtypes and reset behaviour, car edit notice and disabled options, `fuel_locked` handling, electric consumption texts, all in de and en
- [x] 8.6 Grep `resources.ts` and components for remaining `liter`, `Liter`, `litre`, `l/100` that should now follow the car's unit

## 9. Docs and verification

- [x] 9.1 Add `ADD-06` (electric cars: kWh charges, charging subtypes, fuel lock) to `docs/parity-checklist.md`; update `docs/legacy-import.md` for the electric subtype mapping
- [x] 9.2 Run backend tests, frontend tests, lint and type checks; manually check `/refuels` with a seeded account in both languages and both themes
