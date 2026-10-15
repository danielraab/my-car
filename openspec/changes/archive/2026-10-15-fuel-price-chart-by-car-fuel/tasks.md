## 1. Chart series

- [x] 1.1 Give `FuelPriceChart` a `cars: Map<string, Car>` prop and build one series per `carFuels × refuelFuels` pair from the items, leaving out items whose car is not in the map
- [x] 1.2 Label each series `<car fuel> · <refuel fuel>` from the existing `cars.fuels.*` and `refuels.fuels.*` translations, with class `chart-fuel-<carFuel>-<refuelFuel>`

## 2. Colours

- [x] 2.1 Replace `.chart-normal`, `.chart-special` and `.chart-other` in `styles.css` with the 12 `chart-fuel-*` classes from the design's palette (hue per car fuel, mid/dark/light shade for normal/premium/other)
- [x] 2.2 Check the palette with the `dataviz` skill's validator against the chart background and adjust shades until every pair is distinguishable
- [x] 2.3 Update the `line-chart` test fixtures if they rely on the removed classes

## 3. Route and copy

- [x] 3.1 On `/refuels`, show the chart loading state while the chart or cars query is pending, the retryable error if either failed (retry refetches the failed ones), and pass `carsById` to `FuelPriceChart`
- [x] 3.2 Update `refuels.chartDescription` in en and de to say there is one line per car fuel and refuel fuel

## 4. Tests

- [x] 4.1 `fuel-price-chart.test.tsx`: diesel and gasoline normal refuels give two lines; two diesel cars share one line; the legend shows "Diesel · Premium"-style labels; a pair keeps its class when other pairs are absent
- [x] 4.2 `-refuels.test.tsx`: the chart waits for the cars, shows the retryable error when the cars fail, and recovers on retry
- [x] 4.3 Run the frontend lint, typecheck and test suite
