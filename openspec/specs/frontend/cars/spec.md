# frontend/cars Specification

## Purpose

Defines the authenticated cars screens where a user lists, creates, views,
and edits their cars, in German and English.

## Requirements

### Requirement: Cars list shows the caller's cars, paginated
The frontend SHALL render the `/cars` route inside the authenticated
application shell and SHALL load the caller's cars from the documented
paginated cars operation, showing type, make, name, fuel (localized),
first registration, license plate, and purchase price for each row. An
absent first registration or license plate SHALL be shown as the localized
"not recorded" placeholder. It SHALL NOT display FIN, purchase date,
deregistration date, or information in the list. Every car SHALL be listed,
including deregistered ones, in the order the operation returns them. A row
of a car that is inactive or deregistered (deregistration date set and today
or earlier in the user's local time zone) SHALL be shown muted with a
localized text badge. While
the first page loads it SHALL show a localized loading state; a load
failure SHALL show a localized error with a retry action. When a further
page is available the screen SHALL offer a localized action to load it,
appending the results to the visible list without discarding what is
already shown. Each row SHALL link to that car's detail screen, and the
screen SHALL offer a localized action to create a new car.

#### Scenario: User opens the cars list
- **WHEN** an authenticated user navigates to `/cars`
- **THEN** the screen shows the documented columns for each of the
  caller's cars and a link to create a new car

#### Scenario: Car without registration details is listed
- **WHEN** the cars list contains a car whose first registration and
  license plate are `null`
- **THEN** both cells show the localized "not recorded" placeholder

#### Scenario: Inactive car is listed muted
- **WHEN** the cars list contains a car with `isActive` set to `false`
- **THEN** its row is muted and carries the localized badge

#### Scenario: Deregistered car is listed muted
- **WHEN** the cars list contains a car whose deregistration date is today
  or earlier
- **THEN** its row is shown in place, muted, with the localized badge

#### Scenario: Future deregistration date
- **WHEN** a car's deregistration date is after today and `isActive` is `true`
- **THEN** its row is not muted

#### Scenario: User loads a further page
- **WHEN** the caller has more cars than the first page returned and the
  user activates the load-more action
- **THEN** the additional cars are appended to the visible list

#### Scenario: Cars list cannot be loaded
- **WHEN** the cars operation fails with a transport or server error
- **THEN** the screen shows a localized error with a retry action

### Requirement: User can create a car
The frontend SHALL render a `/cars/create` route with a form for type,
make, name, fuel (a localized choice among the four documented codes),
an active-state switch that is on by default, and optional first
registration, license plate, FIN, purchase date, purchase price,
deregistration date, and information (a multi-line text area). An optional
field left blank SHALL be submitted as `null`. The form SHALL reject a
deregistration date earlier than the first registration when both are
entered. Submitting a valid form SHALL create the car and navigate to its
detail screen. A validation rejection SHALL be shown as a localized message
associated with the affected field, and SHALL NOT navigate away or lose the
entered values.

#### Scenario: User creates a car
- **WHEN** the user fills in the required fields and submits the form
- **THEN** the car is created, active, and the user is taken to its detail
  screen

#### Scenario: User creates a car without registration details
- **WHEN** the user leaves first registration and license plate blank and
  submits an otherwise valid form
- **THEN** the car is created with both values `null`

#### Scenario: User creates an inactive car with information
- **WHEN** the user turns the active switch off, enters an information text,
  and submits
- **THEN** the car is created with `isActive` `false` and that information

#### Scenario: Deregistration precedes first registration
- **WHEN** the user enters a deregistration date earlier than the first
  registration
- **THEN** the form shows a localized message at the deregistration field and
  does not submit

#### Scenario: Backend rejects a field
- **WHEN** the backend responds `400` with a field detail
- **THEN** the frontend shows a localized validation message on that field
  and keeps the entered values

### Requirement: Car detail screen shows the caller's car
The frontend SHALL render a `/cars/{carId}` route that loads the car from
the documented single-car operation and shows a localized action to edit
the car. It SHALL show the car's type, make, name, fuel (localized), first
registration, license plate, FIN, purchase date, purchase price, active
state, deregistration date, and information as read-only values on its
Details tab, showing the localized "not recorded" placeholder for any absent
optional value. The information SHALL preserve its line breaks. A header that
includes the license plate SHALL omit it when it is absent. A car that cannot
be loaded, including one owned by another account, SHALL show a localized
not-found or error state rather than partial data or any tab content.

#### Scenario: User opens a car's detail screen
- **WHEN** an authenticated user navigates to one of their car's detail
  screens
- **THEN** the screen shows the car's documented fields on the Details tab
  and an edit action

#### Scenario: Car without registration details is shown
- **WHEN** the user opens the detail screen of a car whose first
  registration and license plate are `null`
- **THEN** both values show the localized "not recorded" placeholder and
  the header shows no license plate separator

#### Scenario: Multi-line information is shown
- **WHEN** the car's information contains line breaks
- **THEN** the Details tab shows them as separate lines

#### Scenario: Deregistered car detail remains reachable
- **WHEN** the user opens the detail screen of a deregistered car
- **THEN** the screen loads normally and shows the deregistration date

#### Scenario: Car cannot be found
- **WHEN** the requested car does not exist or belongs to another account
- **THEN** the screen shows a localized not-found state and no tabs

### Requirement: User can edit or delete a car
The frontend SHALL render a `/cars/{carId}/edit` route pre-filled with the
car's current values, using the same fields as car creation (including the
active-state switch, deregistration date, and information), plus a
two-step-confirm delete action. Submitting a valid edit SHALL save the
changes and every part of the application that displays the car SHALL
reflect them without a page reload, including the selections and dashboard
cards that hide deregistered cars. Deleting SHALL require an explicit
second confirmation before the car and its expenses are removed, after
which the user is returned to the cars list.

#### Scenario: User edits a car
- **WHEN** the user changes a field on the edit form and submits it
- **THEN** the change is saved and shown on the car's detail screen

#### Scenario: User switches a car to inactive
- **WHEN** the user turns the active switch off and saves
- **THEN** the car is saved with `isActive` `false` and its cars-list row is
  muted

#### Scenario: User sets a deregistration date
- **WHEN** the user sets a deregistration date of today and saves
- **THEN** the car no longer appears on the dashboard or in car selections
  without a page reload

#### Scenario: User clears the deregistration date
- **WHEN** the user clears the deregistration date and saves
- **THEN** the car reappears in the dashboard and car selections

#### Scenario: User deletes a car
- **WHEN** the user activates delete once and then confirms
- **THEN** the car is deleted and the user is returned to the cars list

#### Scenario: User cancels a delete
- **WHEN** the user activates delete once but does not confirm
- **THEN** the car is not deleted

### Requirement: Cars screens are fully localized
Every user-facing string on the cars screens, including fuel-type labels,
validation, and error messages, SHALL be available in German and English
and SHALL follow the active locale.

#### Scenario: User switches language on a cars screen
- **WHEN** the user switches the language from English to German on any
  cars screen
- **THEN** all labels, hints, and messages on that screen are shown in
  German

### Requirement: Car detail screen is organized in addressable tabs
The car detail screen SHALL present localized **Details**, **Expenses**, and
**Consumption** tabs as an accessible tab list operable by keyboard. The
selected tab SHALL be reflected in the URL's `tab` search parameter
(`details`, `expenses`, or `consumption`) so that a tab can be linked to and
browser history navigates between tabs. A missing or unknown `tab` value
SHALL select Details. The screen SHALL request expense or chart data only
for the tab that is shown.

#### Scenario: User switches tabs
- **WHEN** the user selects the Expenses tab on a car's detail screen
- **THEN** the Expenses content is shown and the URL carries `tab=expenses`

#### Scenario: User opens a tab link
- **WHEN** the user navigates to `/cars/{carId}?tab=consumption` for one of
  their cars
- **THEN** the Consumption tab is selected on load

#### Scenario: Unknown tab value
- **WHEN** the URL carries a `tab` value that is not a documented tab
- **THEN** the Details tab is selected

### Requirement: Expenses and Consumption share a date range
The Expenses and Consumption tabs SHALL share one localized date-range
control with a "from" date and an optional "to" date, reflected in the
URL's `from` and `to` search parameters as calendar dates (`YYYY-MM-DD`).
When `from` is absent, it SHALL default to the date six months before
today in the user's local time zone; when `to` is absent, the range SHALL
be open-ended. Both boundary days SHALL be included in full in the user's
local time zone. The range SHALL be applied by the backend queries, not by
filtering already-loaded rows, and SHALL persist when the user switches
between the Expenses and Consumption tabs. The control SHALL NOT be shown
on the Details tab. A malformed `from` or `to` value SHALL be treated as
absent.

#### Scenario: Default range
- **WHEN** the user opens the Expenses tab without `from` or `to`
- **THEN** only expenses dated from six months ago onward are requested and
  shown

#### Scenario: User narrows the range
- **WHEN** the user sets "from" to 2026-01-01 and "to" to 2026-01-31
- **THEN** the URL carries `from=2026-01-01&to=2026-01-31`, and the queries
  request expenses from the start of 2026-01-01 up to but excluding the
  start of 2026-02-01 in local time

#### Scenario: Range carries across tabs
- **WHEN** the user changes the range on the Expenses tab and then selects
  the Consumption tab
- **THEN** the consumption chart uses the same range

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
- **THEN** each section lists only that car's records within the range, in
  date order, without a car column

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
- **THEN** that section's rows append without affecting the other sections

### Requirement: Consumption tab charts the car's consumption
The Consumption tab SHALL show a localized line chart of the car's
consumption in litres per 100 km over time within the selected date range,
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

#### Scenario: Not enough data
- **WHEN** no refuel of the car within the range has a consumption value
- **THEN** the tab shows a localized empty-state message

#### Scenario: Chart fails to load
- **WHEN** the chart operation fails
- **THEN** the tab shows a localized retryable error
