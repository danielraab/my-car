## Context

`cars.fuel` already accepts `electric`, but a refuel is modelled as fuel only:
`refuels.liters` (`NUMERIC > 0`), subtype enum `refuel_fuel` = `normal`,
`special`, `other`, and API members `liters` / `perLiter`. The frontend labels
quantities through `useUnitFormat()` in `lib/display-units.tsx`, which reads
one account-wide `volumeUnit` from `DisplayUnitsContext`. Consumption is
computed on the backend (`internal/refuels/handler.go`, `quantity / distance
× 100`, per refuel against the car's previous odometer reading) and does not
depend on the unit. The fuel-price chart (`refuels/fuel-price-chart.tsx`)
draws one line per (car fuel, subtype) pair on the single value axis of
`components/line-chart.tsx`, and teal is already reserved for electric.

The refuel table (`refuels/refuel-table.tsx`) receives a `cars` map on
`/refuels` (car column shown) and none on the car's Expenses tab; the
fuel-price chart and the refuel form already have the caller's cars.

Earlier changes (`fuel-price-chart-by-car-fuel`, `add-account-display-units`)
explicitly deferred electric cars to this change.

## Goals / Non-Goals

**Goals:**
- Charges of electric cars are entered, stored, listed and charted in kWh,
  €/kWh and kWh/100 distance, with charging-specific subtypes.
- The data can never contain a refuel whose subtype contradicts its car's
  fuel.
- API and column names stop saying "litres".

**Non-Goals:**
- Plug-in hybrids, a stored unit per refuel, units other than volume for
  `other` fuel, averaged or state-of-charge-based consumption, charging
  losses, tariffs/session fees, a separate electric chart.

## Decisions

### Quantity unit is derived from the car's fuel, not stored
`unit(refuel) = car.fuel == electric ? kWh : account.volumeUnit`. No column
is added.
- *Alternative:* `refuels.unit` column. Rejected: plug-in hybrids are out of
  scope, and with fuel changes blocked (below) the car's fuel is always
  consistent with its refuels, so a stored unit would only duplicate it.
- Consequence: the frontend needs the car's fuel to label a row. All refuel
  views already have it or can get it cheaply (next decision), so the API
  does not repeat it on each refuel.

### Frontend resolves units per car fuel
`useUnitFormat()` keeps its account-level fields and gains
`quantity(carFuel)` returning `{ symbol, name, perName, consumptionSymbol,
singular }` — kWh-based for `electric`, the account volume unit otherwise.
Existing `volume*` fields stay as the non-electric case so unrelated call
sites need no change. New i18n keys: `units.energy.kwh` ("kWh"), long forms
(`Kilowattstunden`/`Kilowatt-hours`, `pro kWh`/`per kWh`) and neutral
headers (`Menge`/`Quantity`, `Preis pro Einheit`/`Unit price`,
`Verbrauch`/`Consumption`).

`RefuelTable` gets a required `fuelOf(carId) => CarFuel` (or the cars map is
always passed and `showCar` becomes a separate flag). On the car's Expenses
tab the car is already loaded, so it passes a constant function. The table
always uses neutral headers and per-cell units, including on the
single-car Expenses tab, so one table has one layout.
- *Alternative:* neutral headers only when cars are mixed. Rejected: headers
  would change with the filter, for little gain.

### Charging subtypes extend the existing enum
`ALTER TYPE refuel_fuel ADD VALUE 'home'`, `'ac'`, `'dc'`. `other` is shared.
Allowed sets live in one Go table in `internal/refuels`:

```
non-electric: normal, special, other
electric:     home, ac, dc, other
```

The frontend mirrors it in `api/client.ts` (`refuelFuelsFor(carFuel)`), with
default `normal` / `ac`. Chart shades for electric (darkest → lightest):
`dc`, `ac`, `home`, `other`; teal ramp (e.g. 800/600/400/300) checked for
contrast in both themes like the existing pairs.
- *Alternative:* a separate `charge_type` column or enum. Rejected: the
  subtype already drives chart lines and labels; one column keeps the chart,
  table and API shapes unchanged.
- Postgres caveat: a value added with `ADD VALUE` cannot be used in the same
  transaction. The migration only adds values and renames the column, and
  nothing in it uses the new values, so it runs as is; the seed and tests
  run in later transactions.

### Subtype-versus-car check in the refuel handler, reason `fuel_mismatch`
On create, the handler loads the target car (it already checks ownership)
and validates the subtype against its fuel. On update, it validates a
supplied subtype against the refuel's car's fuel. Rejections are field
errors on `fuel` with a new reason `fuel_mismatch` documented in the common
validation reasons.
- *Alternative:* a database trigger. Rejected: cross-table checks belong in
  Go here, like the other domain rules; the DB keeps simple enum and value
  constraints.

### Fuel change guard in the car update, reason `fuel_locked`
`PATCH /cars/{id}` with a `fuel` that crosses the electric boundary relative
to the stored fuel is rejected when the car has refuels. Implemented as part
of the update transaction:

```
SELECT fuel FROM cars WHERE id=$1 AND account_id=$2 FOR UPDATE
-- if crossing: SELECT EXISTS(SELECT 1 FROM refuels WHERE car_id=$1)
-- → 400 fields.fuel = fuel_locked
UPDATE cars ...
```

Refuel creation and subtype updates take `SELECT fuel FROM cars ... FOR
SHARE` in their transaction, so a concurrent fuel change and refuel insert
serialize on the car row and cannot produce a mismatched pair.
- *Alternative:* remap subtypes on fuel change, or tolerate mismatches.
  Rejected during exploration in favour of strict consistency.

### `refuelCount` on the car representation
Added like `attachmentCount`: a `(SELECT count(*) FROM refuels r WHERE
r.car_id = cars.id)` column in the car query. The edit form uses it to
disable options and show the notice without an extra request. The value can
be stale by the time the form is saved; the backend guard and the
`fuel_locked` field message cover that.

### Rename `liters` → `quantity`, `perLiter` → `unitPrice`
OpenAPI members, Go structs and JSON tags, and the column (`ALTER TABLE
refuels RENAME COLUMN liters TO quantity`; the `CHECK (liters > 0)`
constraint follows the column automatically, and its name is kept). The
validation reason key changes from `fields.liters` to `fields.quantity`.
`special` keeps its name. The backend and frontend ship together; no
compatibility alias is kept.

### Second value axis in `LineChart`
`LineSeries` gains `axis?: 'left' | 'right'`. The chart computes ticks for
each axis from its own series with the existing `chart-ticks` helper, maps
right-axis points with their own scale, draws right tick labels, and keeps
gridlines on the left scale only. Optional `axisLabels: { left?, right? }`.
Nearest-point lookup already uses screen coordinates, so it works unchanged
once points are projected. The fuel-price chart assigns electric lines to
`right` only when non-electric lines are present; otherwise everything is
on `left`, so the single-car electric view looks like today's chart.
- *Alternative:* normalise prices to one axis, or two stacked charts.
  Rejected during exploration.
- Narrow screens: right tick labels reduce plot width; the label width is
  measured like the left labels.

### Legacy import and seed
Import maps legacy subtypes per mapped car fuel (`Normal`→`ac`,
`Special`→`dc`, `Others`→`other` for electric cars), since a fuel subtype
on an electric car would now be invalid. Legacy had no "home" concept, so
nothing maps to `home`. Seed adds one electric model (e.g. "Volkswagen
ID.3", within 3–5 cars) with 15–60 kWh charges at 0.25–0.80 per kWh,
subtype weighted towards `home`/`ac`.

## Risks / Trade-offs

- [Per-charge consumption is noisy, because EVs are rarely charged to full]
  → Accepted; the formula is identical to fuel and the user chose it. An
  averaged view would be its own proposal.
- [A user with a wrongly typed car cannot change fuel without deleting its
  refuels] → Accepted as rare; the notice explains it. A later "move
  refuels" or bulk delete would help.
- [Renamed API members break external clients] → None known; the frontend
  ships with the backend. The change is marked BREAKING in the proposal.
- [Legacy `Normal` charges might have been home charging] → They import as
  `ac`; the user can edit them. Mapping to `home` would be a guess too.
- [Two axes can be misread] → Both axes carry unit labels whenever both are
  shown, and popovers state each value's unit.
- [Missed hard-coded "liters"/"perLiter" in the frontend] → Regenerating
  `schema.gen.ts` makes TypeScript flag every use; grep `liter` in
  `resources.ts` and components as a task.

## Migration Plan

1. Migration `000010_electric_charging`: `ALTER TYPE refuel_fuel ADD VALUE`
   ×3; `ALTER TABLE refuels RENAME COLUMN liters TO quantity`. Additive plus
   a rename; existing rows are untouched.
2. Down migration renames the column back. Enum values cannot be dropped
   from a Postgres enum in place; the down migration first fails if any
   refuel uses `home`/`ac`/`dc`, otherwise recreates `refuel_fuel` without
   them (create new type, alter column using cast, drop old type, rename).
3. Backend and frontend deploy together. Rollback: previous build plus the
   down migration (only possible while no charging subtypes are stored).

## Open Questions

None. Decisions from exploration: unit derived from car fuel; per-charge
consumption; second axis on the fuel-price chart; separate electric subtypes
`home`, `ac`, `dc` with shared `other`; electric-boundary fuel changes
blocked while a car has refuels, with an inline notice; API and column
renamed; `other` fuel stays on the volume unit.
