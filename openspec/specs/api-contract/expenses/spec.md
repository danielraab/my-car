# api-contract/expenses Specification

## Purpose

Defines caller-scoped expense records, autocomplete data, and complete chart data that preserve legacy expense and consumption behaviour under pagination.

## Requirements

### Requirement: Caller can manage refuels, repairs, and tickets
The system SHALL document caller-scoped collection, create, retrieve, update, and delete operations for refuels, repairs, and tickets. Refuels SHALL include date-time, station, an optional odometer reading, fuel subtype, `quantity`, amount, and the derived `unitPrice`; repairs and tickets SHALL include their legacy parity fields. A refuel's `quantity` SHALL be in kilowatt-hours when its car's fuel is `electric` and in the account volume unit otherwise, and its `unitPrice` SHALL be the amount per that unit; the API SHALL NOT convert either. Refuel fuel subtype SHALL accept only `normal`, `special`, `home`, `ac`, `dc`, and `other`, subject to the car-fuel rule of "Refuel subtype fits the car's fuel"; repair type SHALL accept only `check`, `service`, `wearing_part`, and `crash_repair`; ticket type SHALL accept only `parking`, `velocity`, and `other`. Refuel `quantity` MUST be greater than zero. Updates SHALL apply every supplied valid field, including `0`, `false`, and empty optional text, rather than ignoring falsy values. `RefuelInput`, `RefuelUpdate`, `RepairInput`, `RepairUpdate`, `TicketInput`, and `TicketUpdate` SHALL reject request members outside their documented properties with `400` and the common validation error representation identifying the offending member. The refuel, repair, and ticket collections SHALL each return an explicit cursor page ordered newest first: by date descending, then by ID descending as the tie-breaker. The refuel station suggestion operation and the ticket location suggestion operation SHALL each have their own operation identifier and a string-array response. A ticket representation SHALL always contain every documented member, with an empty description when none was recorded.

#### Scenario: Caller updates an amount to zero
- **WHEN** an authenticated caller patches an owned expense amount to `0`
- **THEN** the response contains the updated amount of `0`

#### Scenario: Caller creates an expense for an owned car
- **WHEN** an authenticated caller submits a valid expense creation request for an owned car
- **THEN** the response is `201` with that expense linked to the requested car

#### Scenario: Caller omits an odometer reading
- **WHEN** an authenticated caller submits a valid expense without an odometer reading
- **THEN** the response represents the expense with an absent odometer reading

#### Scenario: Caller supplies an unsupported expense category
- **WHEN** an authenticated caller submits or updates an expense with a category value outside its documented codes
- **THEN** the response is `400` with the common validation error representation

#### Scenario: Caller supplies zero quantity
- **WHEN** an authenticated caller creates or updates a refuel with `quantity` equal to zero
- **THEN** the response is `400` with `fields.quantity` set to `non_positive`

#### Scenario: Caller uses the former member names
- **WHEN** an authenticated caller submits a refuel creation or update request containing `liters`
- **THEN** the response is `400` identifying `liters` as an undocumented member, and no refuel is created or changed

#### Scenario: Unit price of a charge
- **WHEN** an authenticated caller retrieves a refuel of an electric car with `quantity` `40` and amount `14.00`
- **THEN** the representation contains `unitPrice` `0.350`, and no `liters` or `perLiter` member

#### Scenario: Caller supplies an undocumented refuel field
- **WHEN** an authenticated caller submits a refuel creation or update request with an undocumented member
- **THEN** the response is `400` identifying that member, and no refuel is created or changed

#### Scenario: Caller supplies an undocumented repair field
- **WHEN** an authenticated caller submits a repair creation or update request with a member outside `RepairInput`/`RepairUpdate`'s documented properties
- **THEN** the response is `400` with the common validation error representation identifying that member, and no repair is created or changed

#### Scenario: Caller supplies an undocumented ticket field
- **WHEN** an authenticated caller submits a ticket creation or update request with a member outside `TicketInput`/`TicketUpdate`'s documented properties
- **THEN** the response is `400` with the common validation error representation identifying that member, and no ticket is created or changed

#### Scenario: Caller pages through tickets
- **WHEN** an authenticated caller lists tickets and the response carries a non-null `nextCursor`
- **THEN** requesting the collection with that cursor returns the following, older tickets in descending date-then-ID order, without duplicates or gaps, and the last page carries a `null` `nextCursor`

#### Scenario: Caller clears a ticket description
- **WHEN** an authenticated caller patches an owned ticket's `description` to the empty string
- **THEN** the response contains the ticket with an empty `description`, and every other member unchanged

#### Scenario: Newest expense is listed first
- **WHEN** an authenticated caller lists refuels, repairs, or tickets without a cursor
- **THEN** the first item is the caller's matching record with the latest date, and items with the same date are ordered by descending ID

### Requirement: Suggestions are restricted to the caller's data
The system SHALL document station and ticket-location suggestion operations whose results are drawn only from the authenticated caller's expense records, are distinct values, and are ordered lexically.

#### Scenario: Caller requests station suggestions
- **WHEN** an authenticated caller requests station suggestions
- **THEN** the response contains no station value contributed solely by another account

### Requirement: Refuel data required for consumption and price charts is complete
The system SHALL document a chart-oriented refuel query, separately from the paginated table collection, that returns all matching caller-owned refuels in date order with server-computed `unitPrice`, `distance`, and `consumption`. `consumption` SHALL be the refuel's `quantity` per 100 units of `distance`, for electric and non-electric cars alike. `distance` and `consumption` SHALL be absent only for a car's first applicable refuel, not because a table page begins after its predecessor, and not because the predecessor lies before the query's `from` bound.

#### Scenario: Refuel follows a prior page boundary
- **WHEN** a chart query includes a refuel whose predecessor is outside a table page
- **THEN** that refuel's distance and consumption are computed from its actual prior refuel for the same car

#### Scenario: Refuel's predecessor lies before the requested range
- **WHEN** a chart query with a `from` bound includes a car's first refuel on or after `from`, and that car has an earlier refuel with an odometer reading before `from`
- **THEN** the response contains only refuels within the range, and that first refuel's distance and consumption are computed from the earlier refuel's odometer reading

#### Scenario: Car's first refuel ever
- **WHEN** a chart query includes a car's earliest refuel with an odometer reading
- **THEN** that refuel has no distance and no consumption

#### Scenario: Consumption of a charge
- **WHEN** a chart query includes a charge of an electric car with `quantity` `34.4` whose odometer reading is 200 above the car's previous refuel
- **THEN** that charge has `distance` `200` and `consumption` `17.20`

### Requirement: Dashboard expense data is bounded by an instant range
The system SHALL document an expense-statistics query that requires a `from` and a `to` date-time and accepts an optional caller-owned car filter. `from` SHALL be inclusive and `to` exclusive, the same as the expense collections' range. The response SHALL contain every refuel, repair and ticket of the caller's cars dated within the range, each as a row with its date-time, its kind (`refuel`, `repair` or `ticket`) and its amount, ordered by date, in a single unpaged response. The response SHALL NOT group rows into days or other calendar buckets, because only the client knows the viewer's time zone. A missing or malformed `from` or `to`, or a malformed car filter, SHALL be rejected with `400` and the common validation error representation identifying the parameter. A car filter naming a car the caller does not own SHALL return no rows.

#### Scenario: Caller requests a year of expenses
- **WHEN** an authenticated caller requests statistics with `from` and `to` bounding one year
- **THEN** the response contains a row for each of that caller's refuels, repairs and tickets dated on or after `from` and before `to`, ordered by date, with its kind and amount

#### Scenario: Expense at the range boundary
- **WHEN** an expense is dated exactly at `to`
- **THEN** it is not in the response, and an expense dated exactly at `from` is

#### Scenario: Caller filters by car
- **WHEN** an authenticated caller requests statistics with the ID of one of their cars
- **THEN** the response contains only that car's expenses within the range

#### Scenario: Caller filters by another account's car
- **WHEN** an authenticated caller requests statistics with the ID of a car they do not own
- **THEN** the response is `200` with no rows

#### Scenario: Caller omits the range
- **WHEN** an authenticated caller requests statistics without `from` or without `to`
- **THEN** the response is `400` with the common validation error representation identifying the missing parameter

#### Scenario: Another account's expenses are excluded
- **WHEN** another account has expenses within the requested range
- **THEN** none of them appear in the caller's response

### Requirement: Expense representations carry the attachment count
Every refuel, repair and ticket representation returned by the expense
operations SHALL contain a read-only integer `attachmentCount`: the number of
that entry's attachments. Expense input and update representations SHALL NOT
accept it.

#### Scenario: Refuel with a receipt is listed
- **WHEN** a caller lists their refuels and one refuel has one attachment
- **THEN** that refuel's `attachmentCount` is `1` and every other refuel's is
  `0`

#### Scenario: Client supplies the count
- **WHEN** a repair creation or update request contains `attachmentCount`
- **THEN** the response is `400` with the common validation error identifying
  `attachmentCount`

### Requirement: Refuel subtype fits the car's fuel
The system SHALL accept, for a refuel of a car whose fuel is `electric`, only the subtypes `home`, `ac`, `dc`, and `other`, and for a refuel of any other car only `normal`, `special`, and `other`. A creation request whose subtype does not fit the requested car's fuel, and an update whose supplied subtype does not fit the refuel's car's fuel, SHALL be rejected with `400` and `fields.fuel` set to `fuel_mismatch`, and SHALL NOT create or change a refuel. The check SHALL use the car's fuel as stored at the time of the request.

#### Scenario: Charge of an electric car
- **WHEN** an authenticated caller creates a refuel with subtype `dc` for an owned electric car
- **THEN** the response is `201` with subtype `dc`

#### Scenario: Fuel subtype on an electric car
- **WHEN** an authenticated caller creates a refuel with subtype `special` for an owned electric car
- **THEN** the response is `400` with `fields.fuel` set to `fuel_mismatch`, and no refuel is created

#### Scenario: Charging subtype on a diesel car
- **WHEN** an authenticated caller updates a refuel of a diesel car to subtype `home`
- **THEN** the response is `400` with `fields.fuel` set to `fuel_mismatch`, and the refuel is unchanged

#### Scenario: Shared other subtype
- **WHEN** an authenticated caller creates refuels with subtype `other` for an electric car and for a gasoline car
- **THEN** both are created
