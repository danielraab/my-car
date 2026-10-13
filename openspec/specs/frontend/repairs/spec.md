# frontend/repairs Specification

## Purpose
Defines the authenticated repairs screens where a user lists, creates, and
edits their repairs, in German and English.

## Requirements

### Requirement: Repairs list shows the caller's repairs, paginated, with a running total
The frontend SHALL render the `/repairs` route inside the authenticated
application shell and SHALL load the caller's repairs from the documented
paginated repairs operation, showing date, car (linked to that car's
detail screen), station, odometer reading, type (localized), and amount
for each row, and a running sum of the amount of every row loaded so far.
The screen SHALL provide a localized caller-car filter and a localized
date-range control with a "from" date and an optional "to" date, and SHALL
apply both to the repairs operation. While the first page loads it SHALL
show a localized loading state; a load failure SHALL show a localized
error with a retry action; when the selected car and range match no
repairs it SHALL show a localized empty state. When a further page is
available the screen SHALL offer a localized action to load it, appending
the results to the visible list and updating the running sum without
discarding what is already shown. Each row SHALL link to that repair's
edit screen, and the screen SHALL offer a localized action to create a new
repair.

The car filter and range SHALL be reflected in the URL's `carId`, `from`,
and `to` search parameters, with `from` and `to` as calendar dates
(`YYYY-MM-DD`). When `carId` is absent or malformed, all of the caller's
cars SHALL be shown. When `from` is absent, it SHALL default to the date
six months before today in the user's local time zone; when `to` is
absent, the range SHALL be open-ended. Both boundary days SHALL be
included in full in the user's local time zone. A malformed `from` or
`to` value SHALL be treated as absent. The filter and range SHALL be
applied by the backend query, not by filtering already-loaded rows.

#### Scenario: User opens the repairs list
- **WHEN** an authenticated user navigates to `/repairs` without search
  parameters
- **THEN** the screen shows the documented columns for each of the
  caller's repairs dated from six months ago onward, across all cars, a
  running sum of their amounts, and a link to create a new repair

#### Scenario: User filters by car
- **WHEN** the user selects one of their cars
- **THEN** the URL carries that car's `carId`, and the list and running
  sum contain only that car's repairs within the selected range

#### Scenario: User narrows the range
- **WHEN** the user sets "from" to 2026-01-01 and "to" to 2026-01-31
- **THEN** the URL carries `from=2026-01-01&to=2026-01-31`, and the
  repairs operation is requested from the start of 2026-01-01 up to but
  excluding the start of 2026-02-01 in local time

#### Scenario: Filter survives a reload
- **WHEN** the user reloads `/repairs?carId=<id>&from=2025-01-01`
- **THEN** the car filter shows that car, the range starts at 2025-01-01,
  and the repairs operation uses both

#### Scenario: No repairs in the range
- **WHEN** the selected car and range match no repairs
- **THEN** the screen shows a localized empty state for the range

#### Scenario: User loads a further page
- **WHEN** the selected repairs have more rows than the first page
  returned and the user activates the load-more action
- **THEN** the additional repairs are appended to the visible list and the
  running sum includes them

#### Scenario: Repairs list cannot be loaded
- **WHEN** the repairs operation fails with a transport or server error
- **THEN** the screen shows a localized error with a retry action

### Requirement: User can create a repair
The frontend SHALL render a `/repairs/create` route with a form for car (a
choice among the caller's cars), date/time, station (with autocomplete
suggestions from the caller's prior repairs), optional odometer reading,
type (a localized choice among the four documented codes), amount, and
optional description. It SHALL accept an optional `carId` query parameter
to preselect the car. Submitting a valid form SHALL create the repair and
navigate to the repairs list. A validation rejection SHALL be shown as a
localized message associated with the affected field, and SHALL NOT
navigate away or lose the entered values.

#### Scenario: User creates a repair
- **WHEN** the user fills in the required fields and submits the form
- **THEN** the repair is created and the user is taken to the repairs list

#### Scenario: User creates a repair from a car's context
- **WHEN** the user navigates to `/repairs/create?carId=<id>` for one of
  their own cars
- **THEN** the form's car field is preselected to that car

#### Scenario: Backend rejects a field
- **WHEN** the backend responds `400` with a field detail
- **THEN** the frontend shows a localized validation message on that field
  and keeps the entered values

### Requirement: User can edit or delete a repair
The frontend SHALL render a `/repairs/{repairId}/edit` route pre-filled
with the repair's current values, using the same fields as repair
creation with the car field fixed, plus a two-step-confirm delete action.
Submitting a valid edit SHALL save the changes and return the user to the
repairs list with them reflected. Deleting SHALL require an explicit
second confirmation before the repair is removed, after which the user is
returned to the repairs list. A repair that cannot be loaded, including
one whose car belongs to another account, SHALL show a localized
not-found or error state rather than partial data.

#### Scenario: User edits a repair
- **WHEN** the user changes a field on the edit form and submits it
- **THEN** the change is saved and shown on the repairs list

#### Scenario: User deletes a repair
- **WHEN** the user activates delete once and then confirms
- **THEN** the repair is deleted and the user is returned to the repairs
  list

#### Scenario: User cancels a delete
- **WHEN** the user activates delete once but does not confirm
- **THEN** the repair is not deleted

#### Scenario: Repair cannot be found
- **WHEN** the requested repair does not exist or its car belongs to
  another account
- **THEN** the screen shows a localized not-found state

### Requirement: Repairs screens are fully localized
Every user-facing string on the repairs screens, including repair-type
labels, validation, and error messages, SHALL be available in German and
English and SHALL follow the active locale.

#### Scenario: User switches language on a repairs screen
- **WHEN** the user switches the language from English to German on any
  repairs screen
- **THEN** all labels, hints, and messages on that screen are shown in
  German

### Requirement: Repair screens hide deregistered cars from car selections and mute their rows
The frontend SHALL NOT offer a deregistered car in the car field of the repair creation form or in the caller-car filter of the repairs list, except a car that is currently selected through the `carId` search parameter. A car is deregistered when its deregistration date is set and is today or earlier in the user's local time zone. A `carId` that names a car not offered SHALL NOT preselect the creation form's car field. The repairs list SHALL continue to show every repair of every car, including deregistered cars, and SHALL show each row of a car that is inactive or deregistered with a muted appearance and a localized text badge identifying it as such. The edit screen's fixed car SHALL be unaffected.

#### Scenario: Deregistered car is not selectable
- **WHEN** the caller has one active car and one deregistered car and opens
  `/repairs/create`
- **THEN** the car field offers only the active car

#### Scenario: Deregistered car is not a list filter
- **WHEN** the caller opens `/repairs` with a deregistered car among their cars
- **THEN** the car filter offers only the cars that are not deregistered

#### Scenario: Entries of a deregistered car stay visible
- **WHEN** the caller has repairs linked to a deregistered car
- **THEN** the list shows those rows, muted and badged

#### Scenario: Inactive car rows are muted
- **WHEN** a repair's car has `isActive` set to `false`
- **THEN** its row is muted and badged even though the car remains selectable

#### Scenario: Create link for a deregistered car
- **WHEN** the user opens `/repairs/create?carId=<id>` for a deregistered car
- **THEN** the car field is not preselected
