# Spec Delta

## MODIFIED Requirements

### Requirement: Caller can manage owned cars
The system SHALL document paginated `GET /api/v1/cars`, `POST /api/v1/cars`, `GET /api/v1/cars/{carId}`, `PATCH /api/v1/cars/{carId}`, and `DELETE /api/v1/cars/{carId}` operations. Car representations SHALL cover the legacy parity fields: type, make, name, fuel, first registration, license plate, FIN, active state, purchase date, and purchase price, plus a deregistration date and an information text. The car fuel field SHALL accept only the stable codes `other`, `diesel`, `gasoline`, and `electric`. `CarInput` and `CarUpdate` SHALL reject any request member outside their documented properties with `400` and the common validation error representation identifying the offending member. First registration, license plate, FIN, purchase date, purchase price, deregistration date, and information SHALL be optional and SHALL be clearable by an update that sets them to `null`. A license plate, FIN, or information that is supplied as a string SHALL NOT be blank, and information SHALL NOT exceed 2000 characters. When both a deregistration date and a first registration date are present on the resulting car, the deregistration date SHALL NOT be earlier than the first registration date; on an update, a member that is not supplied SHALL be taken from the stored car for this check. A car representation SHALL always contain every documented member, using `null` for an optional value that is not recorded. The active state and the deregistration date SHALL be independent: neither SHALL change the other.

#### Scenario: Caller creates a car
- **WHEN** an authenticated caller submits a valid car creation request
- **THEN** the response is `201` with a car owned by that caller

#### Scenario: Caller creates a car without registration details
- **WHEN** an authenticated caller submits a car creation request that omits
  `firstRegistration` and `licensePlate`, or sets them to `null`
- **THEN** the response is `201` and both members are `null` in the
  returned car

#### Scenario: Caller creates a car with deregistration and information
- **WHEN** an authenticated caller submits a car creation request with
  `deregistrationDate`, `information`, and `isActive` set to `false`
- **THEN** the response is `201` and the returned car carries all three values
  as submitted

#### Scenario: Caller supplies a blank license plate
- **WHEN** an authenticated caller submits a car creation or update request
  with `licensePlate` set to a blank string
- **THEN** the response is `400` with `fields.licensePlate` set to `empty`

#### Scenario: Caller supplies a blank or oversized information text
- **WHEN** an authenticated caller submits a car creation or update request
  with `information` blank, or longer than 2000 characters
- **THEN** the response is `400` with `fields.information` set to `empty` or
  `too_long` respectively

#### Scenario: Deregistration precedes first registration
- **WHEN** an authenticated caller submits a car creation request, or an update
  of a car, such that the resulting `deregistrationDate` is earlier than the
  resulting `firstRegistration`
- **THEN** the response is `400` with `fields.deregistrationDate` set to
  `before_first_registration` and no car is created or changed

#### Scenario: Only one of the two dates is validated against the stored value
- **WHEN** an authenticated caller updates a car with only `deregistrationDate`
  set earlier than the stored `firstRegistration`
- **THEN** the response is `400` with `fields.deregistrationDate` set to
  `before_first_registration`

#### Scenario: Caller supplies an unsupported car fuel
- **WHEN** an authenticated caller submits a car creation or update request
  with a fuel value outside the documented codes
- **THEN** the response is `400` with the common validation error representation

#### Scenario: Caller supplies an undocumented field
- **WHEN** an authenticated caller submits a car creation or update request
  with a member outside `CarInput`/`CarUpdate`'s documented properties
- **THEN** the response is `400` with the common validation error
  representation identifying that member, and no car is created or changed

#### Scenario: Caller clears an optional car field
- **WHEN** an authenticated caller updates one of their cars with
  `firstRegistration`, `licensePlate`, `fin`, `purchaseDate`, `purchasePrice`,
  `deregistrationDate`, or `information` set to `null`
- **THEN** the response is `200` and that member is `null` in the returned car

#### Scenario: Caller changes the active state
- **WHEN** an authenticated caller updates one of their cars with only
  `isActive` set to `false`
- **THEN** the response is `200`, `isActive` is `false`, and
  `deregistrationDate` is unchanged

#### Scenario: Caller deletes a car
- **WHEN** an authenticated caller deletes one of their cars
- **THEN** the response is `204` and its associated expenses are no longer accessible

#### Scenario: Caller requests another caller's car
- **WHEN** an authenticated caller retrieves, updates, or deletes a car
  owned by a different account, or one that does not exist
- **THEN** the response is `404` with the common error representation, with
  no distinction between the two cases