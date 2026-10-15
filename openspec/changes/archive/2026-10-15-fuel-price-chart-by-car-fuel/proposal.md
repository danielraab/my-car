## Why

The fuel-price chart on `/refuels` draws one line per refuel fuel (normal,
premium, other), so a diesel car's and a gasoline car's "normal" refuels are
merged into one line even though their prices are unrelated. The legacy app
drew one line per car fuel and refuel fuel pair (`SCR-13` in
`docs/parity-checklist.md`: "one line per base+sub fuel type"), so this is a
parity gap in the rewrite.

## What Changes

- The fuel-price chart draws one line per (car fuel, refuel fuel) pair present
  in the chart data, using the car's current fuel. The legend and popover show
  both, e.g. "Diesel · Premium", in English and German.
- Line colour: the car's fuel picks the hue and the refuel's fuel picks the
  shade (normal = mid, premium = dark, other = light). Each pair has its own
  colour, which stays the same whatever car or date range is selected.
  All four car fuels get a hue, electric included, so no
  line falls back to a default colour.
- The chart waits for the caller's cars as well as the chart series before
  drawing. If either fails to load, it shows the retryable chart error.
- The chart description text changes from "one line per fuel" to describe the
  new grouping.
- Non-goal: prices for electric cars. A refuel's price is per litre, and
  per-kWh units, charging-specific wording and a separate chart are left to a
  future electric-cars proposal.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `frontend/refuels`: adds a requirement for how the fuel-price chart is split
  into lines, coloured, labelled, and when it waits for the car list.
- `frontend/charts`: the "Chart with several series" scenario names a series
  that doesn't exist today ("diesel"); it now names a "Diesel · Normal" series.

## Impact

- Frontend only: `frontend/src/refuels/fuel-price-chart.tsx` and its test, the
  `/refuels` route (passes the cars to the chart and waits for them), series
  colour tokens in `frontend/src/styles.css`, and i18n strings (en/de).
- The old `.chart-normal` / `.chart-special` / `.chart-other` classes are only
  referenced by the fuel-price chart and the generic `line-chart` test
  fixtures; the fixtures may keep using any class.
- No backend, OpenAPI or database change: the chart already gets each
  refuel's `carId`, and `/refuels` already loads the caller's cars.
