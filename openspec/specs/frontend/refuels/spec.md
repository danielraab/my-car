# frontend/refuels Specification

## Purpose
Defines authenticated, localized refuel management and fuel-price views.

## Requirements

### Requirement: Refuels list is filterable, paginated, and charted
The frontend SHALL render `/refuels` in the authenticated shell with a paginated table of the caller's refuels showing the date (without time of day), car, station, odometer reading, quantity, unit price, amount, and consumption when available, labelled in the account's currency and in the units of each refuel's car as defined by `frontend/display-units`. It SHALL provide a localized caller-car filter and a localized date-range control with a "from" date and an optional "to" date, and SHALL apply both to the collection and chart queries. It SHALL show a fuel-price chart from the complete chart series for the selected car and range rather than from table pages, and a running amount total for loaded table rows. The screen SHALL provide localized loading, retryable-error, empty, load-more, edit, and create states.

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

#### Scenario: Cells in gallons
- **WHEN** the account volume unit is `gallon` and the table shows a diesel car's refuel
- **THEN** that row's quantity and unit price cells are in gallons rather than litres

#### Scenario: Diesel and electric rows together
- **WHEN** the table shows refuels of a diesel car and charges of an electric car
- **THEN** the quantity, unit price and consumption headers name no unit, and each row's cells carry the unit of its own car

### Requirement: Fuel-price chart draws one line per car fuel and refuel fuel
The `/refuels` fuel-price chart SHALL draw one line for each pair of car fuel and refuel subtype that occurs in the chart series, where a refuel's car fuel is the current fuel of the car it belongs to. Pairs with no refuels SHALL NOT be drawn or listed in the legend. Each line SHALL be labelled in the legend and popover with the localized car fuel followed by the localized subtype (for example "Diesel · Premium" or "Electric · DC fast"). A line's hue SHALL be determined by the car fuel and its shade by the subtype: for non-electric fuels normal is the middle shade, premium the darkest and other the lightest; for electric, DC fast is the darkest, followed by AC, home, and other as the lightest. A pair's colour SHALL NOT depend on which other pairs are present, the selected car, or the selected range, and every pair SHALL stay distinguishable from the others against the chart background.

Lines of non-electric cars SHALL be plotted against the left value axis, in price per account volume unit. Lines of electric cars SHALL be plotted against a second value axis on the right, in price per kWh, with its own round ticks. When only electric lines are present, they SHALL use the left axis instead and no right axis SHALL be shown; when no electric line is present, no right axis SHALL be shown. Each axis SHALL be labelled with its unit whenever both axes are shown. The popover SHALL show each point's unit price with its own unit.

The chart SHALL be drawn only once both the chart series and the caller's cars have loaded, and SHALL show the localized retryable chart error if either fails.

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

#### Scenario: Fuel and charging prices together
- **WHEN** the selected range holds normal refuels of a diesel car and home and DC charges of an electric car
- **THEN** "Diesel · Normal" is plotted against a left axis in price per litre, "Electric · Home" and "Electric · DC fast" against a right axis in price per kWh, and both axes name their unit

#### Scenario: Only an electric car selected
- **WHEN** the user selects a single electric car
- **THEN** its lines are plotted against the left axis in price per kWh and no right axis is shown

#### Scenario: Car list still loading
- **WHEN** the chart series has loaded but the caller's cars have not
- **THEN** the chart shows its localized loading state instead of lines

#### Scenario: Car list fails to load
- **WHEN** loading the caller's cars fails
- **THEN** the chart shows its localized retryable error, and retrying reloads the cars

#### Scenario: German labels
- **WHEN** the interface language is German
- **THEN** the legend and popover use the German names of both fuels and subtypes

### Requirement: User can create a refuel
The frontend SHALL render `/refuels/create` with fields for car, date/time, station, optional odometer reading, subtype, quantity, and amount, with the odometer, quantity and amount fields showing their units as defined by `frontend/display-units`. It SHALL offer caller-owned cars, accept `carId` to preselect one, offer caller-owned station suggestions, and offer only the localized subtypes that fit the selected car's fuel: normal, premium and other for a non-electric car; home, AC, DC fast and other for an electric car. When the selected car changes to one whose subtypes do not include the chosen subtype, the subtype SHALL reset to the new car's default (normal or AC). A valid submission SHALL create the refuel and navigate to the list; a field validation error SHALL remain on the form with its values preserved.

#### Scenario: User creates from a car context
- **WHEN** the user visits `/refuels/create?carId=<id>` for their car
- **THEN** that car is preselected

#### Scenario: Electric car offers charging subtypes
- **WHEN** the user selects an electric car
- **THEN** the subtype field offers Home, AC, DC fast and Other, and the quantity field shows "kWh"

#### Scenario: Switching from a gasoline to an electric car
- **WHEN** the user has chosen Premium for a gasoline car and then selects an electric car
- **THEN** the subtype resets to AC

#### Scenario: Other survives a car switch
- **WHEN** the user has chosen Other for a gasoline car and then selects an electric car
- **THEN** the subtype stays Other

#### Scenario: A field is rejected
- **WHEN** creation returns a field validation error
- **THEN** the form localizes it at that field and preserves entered values

### Requirement: User can edit or delete a refuel
The frontend SHALL render `/refuels/{refuelId}/edit` with the current values and creation fields except a fixed car, offering the subtypes and quantity unit of that car. It SHALL provide two-step deletion. A saved edit or confirmed deletion SHALL return the user to the refuels list. A missing or non-owned refuel SHALL render a localized not-found or error state.

#### Scenario: User edits a refuel
- **WHEN** the user saves a changed refuel
- **THEN** the list reflects the saved value

#### Scenario: User edits a charge
- **WHEN** the user opens the edit screen of an electric car's charge
- **THEN** the subtype field offers Home, AC, DC fast and Other, and the quantity field shows "kWh"

#### Scenario: User confirms deletion
- **WHEN** the user explicitly confirms deletion
- **THEN** the refuel is deleted and the user returns to the list

### Requirement: Refuel screens are fully localized
Every refuel screen string, including subtype labels, chart labels, validation, and errors, SHALL be available in German and English.

#### Scenario: User switches language
- **WHEN** the user changes from English to German on a refuel screen
- **THEN** all its labels, chart labels, hints, and messages are German

### Requirement: Refuel screens hide deregistered cars from car selections and mute their rows
The frontend SHALL NOT offer a deregistered car in the car field of the refuel creation form or in the caller-car filter of the refuels list, except a car that is currently selected through the `carId` search parameter. A car is deregistered when its deregistration date is set and is today or earlier in the user's local time zone. A `carId` that names a car not offered SHALL NOT preselect the creation form's car field. The refuels list SHALL continue to show every refuel of every car, including deregistered cars, and SHALL show each row of a car that is inactive or deregistered with a muted appearance and a localized text badge identifying it as such. The edit screen's fixed car SHALL be unaffected.

#### Scenario: Deregistered car is not selectable
- **WHEN** the caller has one active car and one deregistered car and opens
  `/refuels/create`
- **THEN** the car field offers only the active car

#### Scenario: Deregistered car is not a list filter
- **WHEN** the caller opens `/refuels` with a deregistered car among their cars
- **THEN** the car filter offers only the cars that are not deregistered

#### Scenario: Entries of a deregistered car stay visible
- **WHEN** the caller has refuels linked to a deregistered car
- **THEN** the list shows those rows, muted and badged

#### Scenario: Inactive car rows are muted
- **WHEN** a refuel's car has `isActive` set to `false`
- **THEN** its row is muted and badged even though the car remains selectable

#### Scenario: Create link for a deregistered car
- **WHEN** the user opens `/refuels/create?carId=<id>` for a deregistered car
- **THEN** the car field is not preselected

### Requirement: Refuel forms manage attachments
The refuel create form SHALL include the attachment upload component in queueing
mode, and the refuel edit form in immediate mode, as defined by
`frontend/attachments`. After a refuel is created and its queued files are
uploaded, the user SHALL return to the refuels list as before.

#### Scenario: Refuel created with a file
- **WHEN** the user queues an image on the refuel create form and submits a valid
  form
- **THEN** the refuel is created, the image is uploaded, and the user returns to
  the refuels list

#### Scenario: File added while editing
- **WHEN** the user picks a PDF on the refuel edit form
- **THEN** it is uploaded immediately and listed among the refuel's attachments

### Requirement: Refuel rows show the attachment indicator
The refuels list SHALL show the clickable attachment indicator defined by
`frontend/attachments` on rows whose `attachmentCount` is above zero,
alongside any inactive or deregistered car badge.

#### Scenario: Refuel with attachments is listed
- **WHEN** the refuels list contains a refuel with two attachments
- **THEN** its row shows the attachment button with the count 2

### Requirement: Refuels list shows the newest refuel first
The `/refuels` table SHALL list the refuels in the order the paginated
refuels operation returns them, which is newest first. Rows loaded with the
load-more action SHALL append below the rows already shown, so the list
stays newest first from top to bottom.

#### Scenario: Recently added refuel is at the top
- **WHEN** the user opens `/refuels` and the selected car and range include
  a refuel dated later than every other matching refuel
- **THEN** that refuel is the first row of the table

#### Scenario: Further page continues with older refuels
- **WHEN** the user activates the load-more action
- **THEN** the appended rows are all dated no later than the last row shown
  before loading
