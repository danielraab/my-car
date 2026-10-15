## MODIFIED Requirements

### Requirement: Expenses tab lists the car's refuels, repairs, and tickets
The Expenses tab SHALL show three localized sections — refuels, repairs,
and tickets — each with a paginated table of that car's records within the
selected date range, loaded from the car-filtered collection operation. The
tables SHALL show the same columns as the corresponding `/refuels`,
`/repairs`, and `/tickets` lists except the car column, including refuel
consumption where available, and each row SHALL link to that record's edit
screen. Each section SHALL offer a localized "add new" action that opens
the corresponding create screen with this car preselected. Each section
SHALL provide its own localized loading, empty, retryable-error, and
load-more states. The tab SHALL NOT show amount totals.

#### Scenario: User views a car's expenses
- **WHEN** the user opens the Expenses tab for a car with refuels, repairs,
  and tickets in the range
- **THEN** each section lists only that car's records within the range,
  newest first, without a car column

#### Scenario: User adds a refuel from the car
- **WHEN** the user selects the refuels section's "add new" action
- **THEN** the create-refuel screen opens with this car preselected

#### Scenario: One section fails to load
- **WHEN** the repairs query fails while refuels and tickets succeed
- **THEN** the repairs section shows a localized retryable error and the
  other sections show their records

#### Scenario: Section has more records
- **WHEN** a section's collection has another page and the user selects
  load more
- **THEN** that section's older rows append below the loaded ones without
  affecting the other sections

