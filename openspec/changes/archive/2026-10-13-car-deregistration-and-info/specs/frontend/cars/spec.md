# Spec Delta

## MODIFIED Requirements

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