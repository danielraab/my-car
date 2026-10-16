## MODIFIED Requirements

### Requirement: Legacy rows are mapped by explicit rules
The import SHALL map each legacy row to the v1 schema using only these
documented rules, and SHALL report any row that a rule cannot represent
instead of substituting a value:

- Users → accounts: email trimmed and lowercased (missing, blank, or a
  duplicate after normalization is a problem). Missing first and last names
  become empty. The password hash, special token and verification flag are
  discarded.
- Cars → cars, owned by the account mapped from `UserId` (missing or unknown
  is a problem). Type, make and name are required and must not be blank.
  Fuel maps `Others`→`other`, `Diesel`→`diesel`, `Gasoline`→`gasoline`,
  `Electric`→`electric`. A missing or blank license plate or FIN becomes
  `null`. A missing first registration or purchase date becomes `null`. A
  present one becomes the calendar date of its value, which must be at
  `00:00:00`. A missing active flag becomes `true`. A purchase price that is
  missing or `0` becomes `null`, and a negative one is a problem.
- Refuels → refuels of the car mapped from `CarId` (missing or unknown is a
  problem). Station is required and must not be blank. For a car whose
  mapped fuel is not `electric`, fuel maps `Normal`→`normal`,
  `Special`→`special`, `Others`→`other`; for an electric car, fuel maps
  `Normal`→`ac`, `Special`→`dc`, `Others`→`other`. Litres must be present
  and greater than zero and become the quantity. Amount must be present and
  non-negative. A missing odometer reading stays absent.
- Repairs → repairs of the mapped car. Station is required and must not be
  blank. Type maps `Check`→`check`, `Service`→`service`,
  `Wearing part`→`wearing_part`, `Crash repair`→`crash_repair`. Amount must
  be present and non-negative. A blank description becomes `null`.
- Tickets → tickets of the mapped car. Location is required and must not be
  blank. Type maps `Parking`→`parking`, `Velocity`→`velocity`,
  `Others`→`other`. Amount must be present and non-negative. A blank
  description becomes `null`.

Category labels SHALL match exactly; any other label, including a missing
one, is a problem. Date-time values SHALL be interpreted as UTC. Decimal
values SHALL be taken from the dump's literal text, and a literal that is
not a plain decimal number is a problem. `createdAt` and `updatedAt` SHALL be
preserved on every imported row.

#### Scenario: Legacy data contains an unknown category
- **WHEN** a refuel row has fuel `Premium`
- **THEN** the row is reported as a problem naming the table, legacy id,
  column and value, and nothing is imported

#### Scenario: Refuel of a legacy electric car
- **WHEN** a refuel row with fuel `Special` belongs to a car with fuel
  `Electric`
- **THEN** it is imported with subtype `dc`, and its litres value becomes its
  quantity

#### Scenario: Car without registration details
- **WHEN** a car row has no first registration and no license plate
- **THEN** the car is imported with both values `null` and is not reported

#### Scenario: Zero purchase price
- **WHEN** a car row has a purchase price of `0`
- **THEN** the car is imported with no purchase price and is not reported

#### Scenario: Expense of an unknown car
- **WHEN** a repair row's `CarId` does not match any imported car
- **THEN** the row is reported as a problem and nothing is imported
