# persistence/schema Specification

## Purpose

Defines durable, ownership-scoped persistence for the v1 account, vehicle, and
expense data model while preserving the established hard-deletion behavior.

## Requirements

### Requirement: Core records have durable ownership and integrity
The system SHALL persist accounts, cars, refuels, repairs, and tickets with
UUID identifiers. Each car SHALL belong to exactly one account and each expense
SHALL belong to exactly one car. Account emails SHALL be normalized and unique.
Cars and expenses SHALL retain the API's calendar-date, timestamp, nullable,
and decimal-value semantics, and required numeric values SHALL reject negative
values; refuel litres SHALL be greater than zero. Odometer readings MAY be
absent but, when supplied, SHALL be non-negative.

#### Scenario: An expense is associated with an owned car
- **WHEN** a valid refuel, repair, or ticket is persisted for a car
- **THEN** it is associated with that car and cannot exist without it

#### Scenario: An absent odometer reading is stored
- **WHEN** a valid expense is persisted without an odometer reading
- **THEN** the expense remains valid and its odometer reading is absent

### Requirement: Cars and expenses use hard-deletion lifecycle semantics
The system SHALL not soft-delete cars, refuels, repairs, or tickets. Deleting a
car SHALL permanently delete its associated expenses; deleting an individual
expense SHALL permanently remove that expense.

#### Scenario: A car is deleted
- **WHEN** a car is permanently deleted
- **THEN** its refuels, repairs, and tickets are permanently deleted and are no
  longer retrievable

### Requirement: Category values are stable enumerations
The system SHALL persist vehicle fuel, refuel fuel grade, repair type, and
ticket type as the documented stable category codes. These stored codes SHALL
be suitable for client-side localization and SHALL reject values outside their
respective documented sets.

#### Scenario: An unsupported category is submitted
- **WHEN** a persistence operation supplies a category value outside its
  documented set
- **THEN** the value is rejected and no invalid category is stored

### Requirement: Legacy import remains a separate, reviewable operation
The initial schema change SHALL not import legacy data. A later one-time import
MUST validate source data before writing it, map legacy identifiers and category
labels explicitly, and report rows that cannot be represented without a
documented transformation.

#### Scenario: Legacy data contains an unknown category
- **WHEN** an import preflight encounters a legacy category outside the
  documented mapping
- **THEN** it reports the affected source rows for review without silently
  storing a replacement value

### Requirement: Car registration details are optional
The system SHALL persist a car without a first registration date, without a
license plate, or without both. A stored license plate SHALL NOT be blank.

#### Scenario: A car without registration details is stored
- **WHEN** a car is persisted with no first registration date and no license
  plate
- **THEN** the car is stored and both values are absent

#### Scenario: A blank license plate is stored
- **WHEN** a car is persisted with a license plate consisting only of
  whitespace
- **THEN** the write is rejected and no car is stored or changed

### Requirement: Cars record deregistration and free-form information
The system SHALL persist for each car an optional deregistration date and an
optional information text, independent of the car's active state. A stored
information text SHALL NOT be blank, and a stored deregistration date SHALL NOT
precede the car's first registration date when both are present.

#### Scenario: A car without deregistration or information is stored
- **WHEN** a car is persisted with no deregistration date and no information
- **THEN** the car is stored and both values are absent

#### Scenario: A car with both is stored
- **WHEN** a car is persisted with a deregistration date and an information text
- **THEN** both values are stored and returned unchanged

#### Scenario: A blank information text is stored
- **WHEN** a car is persisted with an information text consisting only of
  whitespace
- **THEN** the write is rejected and no car is stored or changed

#### Scenario: Deregistration precedes first registration
- **WHEN** a car is persisted with a deregistration date earlier than its first
  registration date
- **THEN** the write is rejected and no car is stored or changed
