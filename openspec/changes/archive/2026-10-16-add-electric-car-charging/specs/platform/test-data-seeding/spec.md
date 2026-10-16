## MODIFIED Requirements

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
