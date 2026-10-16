# platform/legacy-data-import Specification

## Purpose

Provides a one-time, validated import of the legacy MariaDB data, exported
as a `mysqldump` file, into a fresh v1 database.

## Requirements

### Requirement: Import command reads a legacy dump file
The backend SHALL expose an `import-legacy` CLI subcommand that takes the
path to a `mysqldump` SQL file and an optional `--dry-run` flag. It SHALL read
the legacy data by parsing that file directly, without connecting to a MySQL
or MariaDB server. It SHALL read the `Users`, `Cars`, `Refuels`, `Repairs` and
`Tickets` tables, and identify values by the column names declared in each
table's `CREATE TABLE` statement rather than by position. It SHALL ignore the
`RefreshTokens` and `SequelizeMeta` tables. A missing or unreadable file, a
missing required table or column, an unparsable `INSERT` statement, or a
dump whose session time zone is not `+00:00` SHALL be reported as a problem.

#### Scenario: Operator omits the dump path
- **WHEN** the operator runs `import-legacy` without a file path
- **THEN** the command prints its usage and exits with an error, without
  touching the database

#### Scenario: Column order differs from the expected schema
- **WHEN** the dump declares a table's columns in a different order than
  the legacy migrations
- **THEN** every value is still read into the column its `CREATE TABLE`
  statement names

#### Scenario: Escaped text values
- **WHEN** a text value in the dump contains escaped quotes, backslashes,
  or newlines
- **THEN** the imported value contains the unescaped characters

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

### Requirement: Legacy identifiers are not persisted
The import SHALL assign new UUIDs to every imported account, car and expense,
and SHALL resolve legacy foreign keys through a lookup table held only in the
memory of the running command. It SHALL NOT store legacy identifiers or the
lookup table in the database.

#### Scenario: Expenses keep their car
- **WHEN** the import completes
- **THEN** every imported refuel, repair and ticket belongs to the car that
  its legacy row referenced, and that car belongs to the account mapped from
  the legacy car's user

### Requirement: Dry run reports only problems
With `--dry-run`, the command SHALL perform all reading and validation, SHALL
NOT write to the database, and SHALL print only the problems found. Each
problem is printed on its own line with the legacy table, legacy row id
(where one exists), column, offending value and reason. When there are no
problems it SHALL print a single line saying so. The command SHALL exit with a
non-zero status when any problem is found. Problem output SHALL NOT include
password hashes or tokens.

#### Scenario: Clean dump in dry-run mode
- **WHEN** the operator runs `import-legacy dump.sql --dry-run` against a
  dump without problems
- **THEN** the command prints a single "no problems" line, exits with status
  0, and the database is unchanged

#### Scenario: Dump with problems in dry-run mode
- **WHEN** the dump contains two invalid rows
- **THEN** the command prints exactly two problem lines and exits with a
  non-zero status

### Requirement: Import runs once into a fresh database, atomically
Without `--dry-run`, the command SHALL apply pending schema migrations,
then refuse to import, before writing anything, unless the `accounts`,
`cars`, `refuels`, `repairs` and `tickets` tables are all empty. It SHALL
validate the whole dump first, and SHALL write nothing if any problem is
found, printing the problems as in dry-run mode. It SHALL write all imported
rows in a single transaction, so that a failure leaves the database
unchanged. On success it SHALL print the number of rows imported per table.

#### Scenario: Successful import
- **WHEN** the operator runs `import-legacy dump.sql` against a fresh
  database with a dump without problems
- **THEN** all accounts, cars and expenses are imported and the per-table
  counts are printed

#### Scenario: Database is not fresh
- **WHEN** the operator runs the import against a database that already
  contains an account
- **THEN** the command exits with an error and writes nothing

#### Scenario: Second run
- **WHEN** the operator runs the import again after a successful import
- **THEN** the command refuses because the database is not fresh, and no
  rows are duplicated

#### Scenario: Write fails midway
- **WHEN** a database error occurs while inserting the imported rows
- **THEN** the transaction is rolled back and no imported rows remain

#### Scenario: Imported user logs in
- **WHEN** a legacy user logs in by magic link or OIDC with their legacy
  email address in any letter case
- **THEN** they are signed in to the imported account and see their
  imported cars and expenses
