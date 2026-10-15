## MODIFIED Requirements

### Requirement: Refuels list is filterable, paginated, and charted
The frontend SHALL render `/refuels` in the authenticated shell with a paginated table of the caller's refuels showing the date (without time of day), car, station, odometer reading, volume, per-volume price, amount, and consumption when available, labelled in the account's currency and units as defined by `frontend/display-units`. It SHALL provide a localized caller-car filter and a localized date-range control with a "from" date and an optional "to" date, and SHALL apply both to the collection and chart queries. It SHALL show a fuel-price chart from the complete chart series for the selected car and range rather than from table pages, and a running amount total for loaded table rows. The screen SHALL provide localized loading, retryable-error, empty, load-more, edit, and create states.

The car filter and range SHALL be reflected in the URL's `carId`, `from`, and `to` search parameters, with `from` and `to` as calendar dates (`YYYY-MM-DD`). When `carId` is absent or malformed, all of the caller's cars SHALL be shown. When `from` is absent, it SHALL default to the date six months before today in the user's local time zone; when `to` is absent, the range SHALL be open-ended. Both boundary days SHALL be included in full in the user's local time zone. A malformed `from` or `to` value SHALL be treated as absent. The range SHALL be applied by the backend queries, not by filtering already-loaded rows.

#### Scenario: Default range
- **WHEN** an authenticated user opens `/refuels` without `from` or `to`
- **THEN** the table, running total, and fuel-price chart contain only refuels dated from six months ago onward

#### Scenario: User filters by car
- **WHEN** an authenticated user selects one of their cars
- **THEN** the URL carries that car's `carId`, and the table, running total, and fuel-price chart contain only that car's refuels within the selected range

#### Scenario: User narrows the range
- **WHEN** the user sets "from" to 2026-01-01 and "to" to 2026-01-31
- **THEN** the URL carries `from=2026-01-01&to=2026-01-31`, and both the collection and chart queries request refuels from the start of 2026-01-01 up to but excluding the start of 2026-02-01 in local time

#### Scenario: Filter survives a reload
- **WHEN** the user reloads `/refuels?carId=<id>&from=2025-01-01`
- **THEN** the car filter shows that car, the range starts at 2025-01-01, and the queries use both

#### Scenario: Consumption at the start of the range
- **WHEN** the first refuel in the selected range has a predecessor for the same car dated before "from"
- **THEN** its row shows the consumption measured against that predecessor

#### Scenario: No refuels in the range
- **WHEN** the selected car and range match no refuels
- **THEN** the screen shows a localized empty state for the range

#### Scenario: User loads another page
- **WHEN** the selected refuel collection has another page and the user selects load more
- **THEN** its rows append to the table and its amounts join the running total without truncating the chart

#### Scenario: Refuels fail to load
- **WHEN** the collection or chart operation fails
- **THEN** the screen shows a localized retryable error

#### Scenario: Columns in gallons
- **WHEN** the account volume unit is `gallon`
- **THEN** the volume and per-volume price columns are headed in gallons rather than litres

### Requirement: User can create a refuel
The frontend SHALL render `/refuels/create` with fields for car, date/time, station, optional odometer reading, fuel subtype, volume, and amount, with the odometer, volume and amount fields showing the account's units as defined by `frontend/display-units`. It SHALL offer caller-owned cars, accept `carId` to preselect one, offer caller-owned station suggestions, and localize fuel choices. A valid submission SHALL create the refuel and navigate to the list; a field validation error SHALL remain on the form with its values preserved.

#### Scenario: User creates from a car context
- **WHEN** the user visits `/refuels/create?carId=<id>` for their car
- **THEN** that car is preselected

#### Scenario: A field is rejected
- **WHEN** creation returns a field validation error
- **THEN** the form localizes it at that field and preserves entered values
