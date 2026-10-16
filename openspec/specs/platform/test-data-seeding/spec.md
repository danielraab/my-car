# platform/test-data-seeding Specification

## Purpose
Provides developers with realistic, isolated sample account data in a resettable database for exercising the application over time.

## Requirements

### Requirement: Seed command requires account emails
The backend SHALL expose a `seed` CLI subcommand accepting one or more email addresses. It MUST reject missing or invalid addresses before modifying the database, normalize address case and whitespace, and avoid creating duplicate accounts for duplicate inputs.

#### Scenario: One or multiple users
- **WHEN** the operator supplies one or more valid email addresses
- **THEN** the command creates one account for each distinct normalized address

#### Scenario: Invalid input
- **WHEN** the operator omits addresses or supplies an invalid address
- **THEN** the command exits with an error without deleting any data

### Requirement: Atomic full database reseed
The command SHALL clear application data from all domain and authentication tables, retaining the migration history and schema, and replace it with seeded data in one transaction. A failure MUST leave the prior data intact.

#### Scenario: Successful reseed
- **WHEN** the command runs against a migrated database containing existing accounts, sessions, and expenses
- **THEN** only newly seeded application data remains and the schema migration state is preserved

#### Scenario: Seed fails
- **WHEN** a database error occurs while generating or writing the seed
- **THEN** none of the deletions or inserts are committed

### Requirement: Multi-year per-account data
Each seeded account SHALL have 3–5 cars, 500 refuels, 50–100 repairs, and 80–100 tickets, assigned to its own cars and distributed throughout the previous five years. At least one of each account's cars SHALL be electric, and its refuels SHALL be charges with charging subtypes and plausible kWh quantities and per-kWh prices; the refuels of every other car SHALL use fuel subtypes. Amounts, dates, fuel types, and odometer readings MUST be plausible and conform to the database constraints and to the subtype-to-car-fuel rule.

#### Scenario: Multiple accounts
- **WHEN** the command seeds two distinct addresses
- **THEN** each account independently has the specified numbers of cars and expenses and no expense refers to the other account's cars

#### Scenario: Timeline
- **WHEN** a seeded account's expenses are inspected
- **THEN** their dates fall within the past five years and cover multiple years

#### Scenario: Electric car
- **WHEN** a seeded account's cars are inspected
- **THEN** at least one is electric, and every refuel of it has subtype `home`, `ac`, `dc`, or `other`
