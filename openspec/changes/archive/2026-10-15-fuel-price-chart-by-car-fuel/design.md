## Context

`FuelPriceChart` (`frontend/src/refuels/fuel-price-chart.tsx`) builds one
`LineChart` series per `refuelFuels` value and filters the chart items by
`item.fuel`. Series colour comes from a fixed CSS class per series
(`.chart-normal`, `.chart-special`, `.chart-other` in `styles.css`), which sets
the `--series` custom property used by the line, dots and legend.

The `/refuels` route already loads `allCarsQueryOptions` and builds `carsById`
for the table, but renders the chart as soon as the chart query resolves.
Each `Refuel` carries `carId`; each `Car` carries `fuel` (`gasoline`,
`diesel`, `electric`, `other`). The legacy chart joined refuels to cars on the
client and keyed lines by `${car.fuel} ${refuel.fuel}`.

The app has a single light theme; there is no dark mode to tune for.

## Goals / Non-Goals

**Goals:**
- One line per (car fuel, refuel fuel) pair present, matching legacy `SCR-13`.
- A fixed, readable colour per pair: hue from car fuel, shade from refuel fuel.
- Localized combined labels.
- No series built from a missing car fuel while cars are loading.

**Non-Goals:**
- Per-kWh units, wording or a separate chart for electric cars (future
  electric-cars proposal). Electric only gets its colours reserved.
- Using the car's fuel at the time of the refuel. Cars store only their current
  fuel, as legacy did.
- Any backend or OpenAPI change, e.g. returning car fuel on chart rows.
- Changing `LineChart` itself.

## Decisions

**Join on the client using the cars already on the page.** `FuelPriceChart`
takes a `Map<string, Car>` (the route's `carsById`) alongside the items, looks
up each item's car fuel, and builds one series per pair. The cars list is
already loaded for the table, so this costs nothing extra.
*Alternative:* add `carFuel` to the chart response. Rejected as it changes the
API contract for a value the client already has.

**Series order is fixed: car fuel order, then refuel fuel order.** Iterate
`carFuels × refuelFuels` (from `api/client.ts`) and let `LineChart` drop empty
series as it does today. This keeps the legend order stable across filters
and matches how the keyboard order breaks ties between series.

**The chart waits for both queries.** The route treats the chart as pending
while either `chart` or `cars` is pending, and as failed if either failed; retry
refetches whichever failed. A refuel whose car isn't in the map (shouldn't
happen for caller-scoped data) is left out of the chart rather than drawn under
a made-up "unknown" series.
*Alternative:* draw refuel-fuel-only lines until cars arrive. Rejected: lines
would split and recolour a moment after first paint.

**Colour by CSS class per pair.** Class `chart-fuel-<carFuel>-<refuelFuel>`
(12 classes), each setting `--series`, replacing `.chart-normal`,
`.chart-special` and `.chart-other`. Palette (Tailwind steps), checked with the
`dataviz` skill's validator against the white chart card:

| car fuel  | normal (mid)     | premium (dark)   | other (light)    |
|-----------|------------------|------------------|------------------|
| gasoline  | amber 600 d97706 | amber 800 92400e | amber 400 fbbf24 |
| diesel    | blue 600 2563eb  | blue 900 1e3a8a  | blue 400 60a5fa  |
| electric  | teal 600 0d9488  | teal 800 115e59  | teal 400 2dd4bf  |
| other     | stone 600 57534e | stone 900 1c1917 | stone 400 a8a29e |

Validator results: the three shades of each hue are ≥ 15 ΔE apart for normal
vision and pass colour-blind separation; the four mids, the four darks and the
four lights each pass the normal-vision floor (worst 15.2) and colour-blind
separation (worst 6.3 tritan, with the legend as secondary encoding). Mids and
darks have ≥ 3:1 contrast; the light shades (1.7–2.5:1) rely on the legend,
popover and refuel table. Green and slate were replaced by teal and stone
because dark green vs dark amber merged under protanopia and light slate vs
light blue were too close (10.8). The validator's lightness-band and chroma
checks fail by design: they assume one hue per series, which this shade
scheme and a grey "other" deliberately break.
*Alternative:* an ordered categorical palette assigned by appearance.
Rejected because colours would shift when the filter changes.

**Labels compose existing translations.** Label =
`${t('cars.fuels.<carFuel>')} · ${t('refuels.fuels.<refuelFuel>')}`. No new
per-pair keys; only `refuels.chartDescription` changes in en and de.

## Risks / Trade-offs

- [Three shades of one hue can be hard to tell apart, especially for colour
  vision deficiencies] → the legend and popover name the series; validate the
  palette with the `dataviz` skill and widen the shade steps if needed.
- [Editing a car's fuel moves all its past refuels to another line] → accepted
  for parity with legacy; noted in non-goals.
- [Electric refuel prices are per kWh but share the per-litre axis] → no
  electric cars exist yet; deferred to the electric-cars proposal.
- [Chart now also blocks on the cars query] → the query is already loaded for
  the table and cached across routes, so the delay is negligible.
