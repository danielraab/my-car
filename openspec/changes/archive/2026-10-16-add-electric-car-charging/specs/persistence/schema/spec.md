## ADDED Requirements

### Requirement: Refuels persist charging subtypes and a unit-neutral quantity
The refuel subtype enumeration SHALL contain `normal`, `special`, `home`, `ac`, `dc`, and `other`. A refuel's quantity SHALL be stored in a column named `quantity` that keeps the existing constraint of being greater than zero; its unit is implied by the car's fuel and is not stored. Introducing these SHALL NOT change any existing refuel's subtype, quantity, amount, or other value.

#### Scenario: Existing refuels after migration
- **WHEN** the migration runs against a database with existing refuels
- **THEN** every refuel keeps its subtype, and its former litres value is its quantity

#### Scenario: Charging subtype is stored
- **WHEN** a persistence operation stores a refuel with subtype `dc`
- **THEN** the refuel is stored with subtype `dc`

#### Scenario: Unsupported subtype is stored
- **WHEN** a persistence operation stores a refuel with subtype `premium`
- **THEN** the value is rejected and no refuel is stored
