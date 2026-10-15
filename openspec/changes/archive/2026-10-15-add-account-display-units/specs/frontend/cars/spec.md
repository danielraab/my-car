## MODIFIED Requirements

### Requirement: Consumption tab charts the car's consumption
The Consumption tab SHALL show a localized line chart of the car's
consumption, in the account's volume unit per 100 of its distance unit as
defined by `frontend/display-units`, over time within the selected date
range, drawn from the complete chart series rather than table pages. Refuels
without a consumption value SHALL be omitted from the line. When no refuel
in the range has a consumption value, the tab SHALL show a localized empty
state instead of a chart. A failed chart query SHALL show a localized
retryable error.

#### Scenario: User views consumption
- **WHEN** the user opens the Consumption tab for a car whose refuels in
  the range have consumption values
- **THEN** the chart plots one point per such refuel, in date order, with
  localized axis and value formatting

#### Scenario: Consumption in gallons per 100 miles
- **WHEN** the account uses `gallon` and `mile` and the user opens the
  Consumption tab
- **THEN** the chart's description and popover values name gallons per 100
  miles, and the plotted values are the same as with litres and kilometres

#### Scenario: Not enough data
- **WHEN** no refuel of the car within the range has a consumption value
- **THEN** the tab shows a localized empty-state message

#### Scenario: Chart fails to load
- **WHEN** the chart operation fails
- **THEN** the tab shows a localized retryable error
