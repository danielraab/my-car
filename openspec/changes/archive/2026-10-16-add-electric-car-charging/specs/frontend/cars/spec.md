## MODIFIED Requirements

### Requirement: Consumption tab charts the car's consumption
The Consumption tab SHALL show a localized line chart of the car's
consumption, in the car's quantity unit per 100 of the account distance
unit as defined by `frontend/display-units` (kWh for an electric car, the
account volume unit otherwise), over time within the selected date range,
drawn from the complete chart series rather than table pages. Refuels
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
  Consumption tab of a gasoline car
- **THEN** the chart's description and popover values name gallons per 100
  miles, and the plotted values are the same as with litres and kilometres

#### Scenario: Consumption of an electric car
- **WHEN** the account uses `liter` and `kilometer` and the user opens the
  Consumption tab of an electric car
- **THEN** the chart's description and popover values name kWh per 100 km

#### Scenario: Not enough data
- **WHEN** no refuel of the car within the range has a consumption value
- **THEN** the tab shows a localized empty-state message

#### Scenario: Chart fails to load
- **WHEN** the chart operation fails
- **THEN** the tab shows a localized retryable error

## ADDED Requirements

### Requirement: Car edit form blocks fuel changes across the electric boundary
While the edited car has at least one refuel, the `/cars/{carId}/edit` form
SHALL disable every fuel choice that would cross the electric boundary: all
non-electric choices for an electric car, and the electric choice for any
other car. It SHALL then show a localized inline notice next to the fuel
field stating that the fuel cannot be switched to or from electric while the
car has refuels, and naming the car's refuel count. Without refuels, every
fuel choice SHALL be enabled and no notice SHALL be shown. A `fuel_locked`
rejection from the backend SHALL be shown as a localized message at the fuel
field, keeping the entered values.

#### Scenario: Gasoline car with refuels
- **WHEN** the user edits a gasoline car with 23 refuels
- **THEN** the electric choice is disabled, diesel and other remain enabled,
  and the notice names 23 refuels

#### Scenario: Electric car with refuels
- **WHEN** the user edits an electric car with refuels
- **THEN** every non-electric choice is disabled and the notice is shown

#### Scenario: Car without refuels
- **WHEN** the user edits a car without refuels
- **THEN** every fuel choice is enabled and no notice is shown

#### Scenario: Refuel added in the meantime
- **WHEN** the user switches a car without refuels to electric, but a refuel
  was added to the car before saving
- **THEN** the form shows the localized `fuel_locked` message at the fuel
  field and keeps the entered values

#### Scenario: German notice
- **WHEN** the interface language is German
- **THEN** the notice is shown in German
