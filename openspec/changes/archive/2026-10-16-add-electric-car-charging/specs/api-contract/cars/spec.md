## ADDED Requirements

### Requirement: Fuel cannot cross the electric boundary while a car has refuels
An update that changes a car's fuel from `electric` to any other code, or from any other code to `electric`, SHALL be rejected with `400` and `fields.fuel` set to `fuel_locked` while the car has at least one refuel, and SHALL NOT change any member of the car. A fuel change among `diesel`, `gasoline`, and `other`, and any fuel change of a car without refuels, SHALL be accepted. A refuel created concurrently with such a fuel change SHALL NOT leave a car whose fuel contradicts its refuels' subtypes.

#### Scenario: Gasoline car with refuels becomes electric
- **WHEN** an authenticated caller patches the fuel of an owned gasoline car that has refuels to `electric`
- **THEN** the response is `400` with `fields.fuel` set to `fuel_locked`, and the car is unchanged

#### Scenario: Electric car with refuels becomes diesel
- **WHEN** an authenticated caller patches the fuel of an owned electric car that has refuels to `diesel`
- **THEN** the response is `400` with `fields.fuel` set to `fuel_locked`

#### Scenario: Diesel car with refuels becomes gasoline
- **WHEN** an authenticated caller patches the fuel of an owned diesel car that has refuels to `gasoline`
- **THEN** the response is `200` with fuel `gasoline`

#### Scenario: Car without refuels becomes electric
- **WHEN** an authenticated caller patches the fuel of an owned gasoline car without refuels to `electric`
- **THEN** the response is `200` with fuel `electric`

#### Scenario: Unchanged fuel in an update
- **WHEN** an authenticated caller patches an owned electric car with refuels, supplying `fuel` `electric` and a new name
- **THEN** the response is `200` with the new name

### Requirement: Car representations carry the refuel count
Every car representation returned by the cars operations SHALL contain a read-only integer `refuelCount`: the number of refuels of the car. `CarInput` and `CarUpdate` SHALL NOT accept it.

#### Scenario: Car with refuels is retrieved
- **WHEN** a caller retrieves an owned car that has 23 refuels
- **THEN** the car's `refuelCount` is `23`

#### Scenario: Client supplies the refuel count
- **WHEN** a car creation or update request contains `refuelCount`
- **THEN** the response is `400` with the common validation error identifying `refuelCount`
