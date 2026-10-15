## ADDED Requirements

### Requirement: Fuel-price chart draws one line per car fuel and refuel fuel
The `/refuels` fuel-price chart SHALL draw one line for each pair of car fuel and refuel fuel that occurs in the chart series, where a refuel's car fuel is the current fuel of the car it belongs to. Pairs with no refuels SHALL NOT be drawn or listed in the legend. Each line SHALL be labelled in the legend and popover with the localized car fuel followed by the localized refuel fuel (for example "Diesel · Premium"). A line's hue SHALL be determined by the car fuel and its shade by the refuel fuel, with normal as the middle shade, premium as the darkest and other as the lightest; every car fuel, including electric, SHALL have a hue. A pair's colour SHALL NOT depend on which other pairs are present, the selected car, or the selected range, and every pair SHALL stay distinguishable from the others against the chart background. The chart SHALL be drawn only once both the chart series and the caller's cars have loaded, and SHALL show the localized retryable chart error if either fails.

#### Scenario: Two car fuels with the same refuel fuel
- **WHEN** the selected range holds normal refuels of a diesel car and normal refuels of a gasoline car
- **THEN** the chart draws two lines, "Diesel · Normal" and "Gasoline · Normal", each with only its own car fuel's refuels

#### Scenario: Several cars with the same fuel
- **WHEN** the selected range holds normal refuels of two different diesel cars
- **THEN** they share one "Diesel · Normal" line

#### Scenario: Colour is stable across filters
- **WHEN** the user switches the car filter from all cars to a single diesel car
- **THEN** the "Diesel · Premium" line keeps the same colour it had with all cars shown

#### Scenario: Shades within one car fuel
- **WHEN** the chart shows "Gasoline · Normal", "Gasoline · Premium" and "Gasoline · Other"
- **THEN** all three use the gasoline hue, with Premium darkest and Other lightest

#### Scenario: Car list still loading
- **WHEN** the chart series has loaded but the caller's cars have not
- **THEN** the chart shows its localized loading state instead of lines

#### Scenario: Car list fails to load
- **WHEN** loading the caller's cars fails
- **THEN** the chart shows its localized retryable error, and retrying reloads the cars

#### Scenario: German labels
- **WHEN** the interface language is German
- **THEN** the legend and popover use the German names of both fuels
