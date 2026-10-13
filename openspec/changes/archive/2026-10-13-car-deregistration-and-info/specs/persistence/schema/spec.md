# Spec Delta

## ADDED Requirements

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