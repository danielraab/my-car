# Spec Delta

## ADDED Requirements

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