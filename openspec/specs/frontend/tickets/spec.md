# frontend/tickets Specification

## Purpose
Defines the authenticated tickets screens where a user lists, creates, and
edits their parking, speeding, and other tickets, in German and English.

## Requirements

### Requirement: Tickets list shows the caller's tickets, paginated, with a running total
The frontend SHALL render the `/tickets` route inside the authenticated
application shell and SHALL load the caller's tickets from the documented
paginated tickets operation. Each row SHALL show date, car (linked to that
car's detail screen), type (localized), location, and amount, and the
screen SHALL show a running sum of the amount of every row loaded so far.
The screen SHALL provide a localized caller-car filter and a localized
date-range control with a "from" date and an optional "to" date, and SHALL
apply both to the tickets operation. While the first page loads it SHALL
show a localized loading state, a load failure SHALL show a localized
error with a retry action, and when the selected car and range match no
tickets it SHALL show a localized empty state. When a further page is
available the screen SHALL offer a localized action to load it, appending
the results to the visible list and updating the running sum without
discarding what is already shown. Each row SHALL link to that ticket's
edit screen, and the screen SHALL offer a localized action to create a new
ticket.

The car filter and range SHALL be reflected in the URL's `carId`, `from`,
and `to` search parameters, with `from` and `to` as calendar dates
(`YYYY-MM-DD`). When `carId` is absent or malformed, all of the caller's
cars SHALL be shown. When `from` is absent, it SHALL default to the date
six months before today in the user's local time zone; when `to` is
absent, the range SHALL be open-ended. Both boundary days SHALL be
included in full in the user's local time zone. A malformed `from` or
`to` value SHALL be treated as absent. The filter and range SHALL be
applied by the backend query, not by filtering already-loaded rows.

#### Scenario: User opens the tickets list
- **WHEN** an authenticated user navigates to `/tickets` without search
  parameters
- **THEN** the screen shows the documented columns for each of the
  caller's tickets dated from six months ago onward, across all cars, a
  running sum of their amounts, and a link to create a new ticket

#### Scenario: User filters by car
- **WHEN** the user selects one of their cars
- **THEN** the URL carries that car's `carId`, and the list and running
  sum contain only that car's tickets within the selected range

#### Scenario: User narrows the range
- **WHEN** the user sets "from" to 2026-01-01 and "to" to 2026-01-31
- **THEN** the URL carries `from=2026-01-01&to=2026-01-31`, and the
  tickets operation is requested from the start of 2026-01-01 up to but
  excluding the start of 2026-02-01 in local time

#### Scenario: Filter survives a reload
- **WHEN** the user reloads `/tickets?carId=<id>&from=2025-01-01`
- **THEN** the car filter shows that car, the range starts at 2025-01-01,
  and the tickets operation uses both

#### Scenario: No tickets in the range
- **WHEN** the selected car and range match no tickets
- **THEN** the screen shows a localized empty state for the range

#### Scenario: User loads a further page
- **WHEN** the selected tickets have more rows than the first page
  returned and the user activates the load-more action
- **THEN** the additional tickets are appended to the visible list and the
  running sum includes them

#### Scenario: Tickets list cannot be loaded
- **WHEN** the tickets operation fails with a transport or server error
- **THEN** the screen shows a localized error with a retry action

### Requirement: User can create a ticket
The frontend SHALL render a `/tickets/create` route with a form for car (a
choice among the caller's cars), date/time, type (a localized choice among
the three documented codes), location (with autocomplete suggestions from
the caller's prior tickets), amount, and optional description. It SHALL
accept an optional `carId` query parameter to preselect the car. Submitting
a valid form SHALL create the ticket and navigate to the tickets list. A
validation rejection SHALL be shown as a localized message associated with
the affected field, and SHALL NOT navigate away or lose the entered values.

#### Scenario: User creates a ticket
- **WHEN** the user fills in the required fields and submits the form
- **THEN** the ticket is created and the user is taken to the tickets list

#### Scenario: User creates a ticket from a car's context
- **WHEN** the user navigates to `/tickets/create?carId=<id>` for one of
  their own cars
- **THEN** the form's car field is preselected to that car

#### Scenario: Location suggestions are offered
- **WHEN** the caller has earlier tickets and opens the create form
- **THEN** the location field offers the caller's distinct prior locations
  as suggestions

#### Scenario: Backend rejects a field
- **WHEN** the backend responds `400` with a field detail
- **THEN** the frontend shows a localized validation message on that field
  and keeps the entered values

### Requirement: User can edit or delete a ticket
The frontend SHALL render a `/tickets/{ticketId}/edit` route pre-filled
with the ticket's current values. It SHALL use the same fields as ticket
creation, with the car field fixed, plus a two-step-confirm delete action.
Submitting a valid edit SHALL save the changes and return the user to the
tickets list with them reflected. Deleting SHALL require an explicit second
confirmation before the ticket is removed, after which the user is returned
to the tickets list. A ticket that cannot be loaded, including one whose
car belongs to another account, SHALL show a localized not-found or error
state rather than partial data.

#### Scenario: User edits a ticket
- **WHEN** the user changes a field on the edit form and submits it
- **THEN** the change is saved and shown on the tickets list

#### Scenario: User deletes a ticket
- **WHEN** the user activates delete once and then confirms
- **THEN** the ticket is deleted and the user is returned to the tickets
  list

#### Scenario: User cancels a delete
- **WHEN** the user activates delete once but does not confirm
- **THEN** the ticket is not deleted

#### Scenario: Ticket cannot be found
- **WHEN** the requested ticket does not exist or its car belongs to
  another account
- **THEN** the screen shows a localized not-found state

### Requirement: Tickets screens are fully localized
Every user-facing string on the tickets screens, including ticket-type
labels, validation, and error messages, SHALL be available in German and
English and SHALL follow the active locale.

#### Scenario: User switches language on a tickets screen
- **WHEN** the user switches the language from English to German on any
  tickets screen
- **THEN** all labels, hints, and messages on that screen are shown in
  German

### Requirement: Ticket screens hide deregistered cars from car selections and mute their rows
The frontend SHALL NOT offer a deregistered car in the car field of the ticket creation form or in the caller-car filter of the tickets list, except a car that is currently selected through the `carId` search parameter. A car is deregistered when its deregistration date is set and is today or earlier in the user's local time zone. A `carId` that names a car not offered SHALL NOT preselect the creation form's car field. The tickets list SHALL continue to show every ticket of every car, including deregistered cars, and SHALL show each row of a car that is inactive or deregistered with a muted appearance and a localized text badge identifying it as such. The edit screen's fixed car SHALL be unaffected.

#### Scenario: Deregistered car is not selectable
- **WHEN** the caller has one active car and one deregistered car and opens
  `/tickets/create`
- **THEN** the car field offers only the active car

#### Scenario: Deregistered car is not a list filter
- **WHEN** the caller opens `/tickets` with a deregistered car among their cars
- **THEN** the car filter offers only the cars that are not deregistered

#### Scenario: Entries of a deregistered car stay visible
- **WHEN** the caller has tickets linked to a deregistered car
- **THEN** the list shows those rows, muted and badged

#### Scenario: Inactive car rows are muted
- **WHEN** a ticket's car has `isActive` set to `false`
- **THEN** its row is muted and badged even though the car remains selectable

#### Scenario: Create link for a deregistered car
- **WHEN** the user opens `/tickets/create?carId=<id>` for a deregistered car
- **THEN** the car field is not preselected

### Requirement: Ticket forms manage attachments
The ticket create form SHALL include the attachment upload component in queueing
mode, and the ticket edit form in immediate mode, as defined by
`frontend/attachments`. After a ticket is created and its queued files are
uploaded, the user SHALL return to the tickets list as before.

#### Scenario: Ticket created with a file
- **WHEN** the user queues an image on the ticket create form and submits a valid
  form
- **THEN** the ticket is created, the image is uploaded, and the user returns to
  the tickets list

#### Scenario: File added while editing
- **WHEN** the user picks a PDF on the ticket edit form
- **THEN** it is uploaded immediately and listed among the ticket's attachments

### Requirement: Ticket rows show the attachment indicator
The tickets list SHALL show the clickable attachment indicator defined by
`frontend/attachments` on rows whose `attachmentCount` is above zero,
alongside any inactive or deregistered car badge.

#### Scenario: Ticket with attachments is listed
- **WHEN** the tickets list contains a ticket with two attachments
- **THEN** its row shows the attachment button with the count 2
