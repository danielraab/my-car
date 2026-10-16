# One-time Legacy Data Import

`db-schema-v1` deliberately creates no importer and does not read the legacy
code. A future, separately approved one-time import runs only after the new
application is working.

Before writing any rows, that importer must produce a reviewable preflight
report and stop on data it cannot represent. The report must include source row
identifiers for duplicate normalized emails, missing required fields, negative
or non-finite legacy float values, and category values outside the mappings.

## Expected mappings

| Legacy field | New field |
| --- | --- |
| `Users.email` | normalized `accounts.email` |
| `Users.firstname` / `lastname` | `accounts.first_name` / `last_name` |
| legacy numeric IDs | generated UUIDs, held in the importer's mapping state |
| `Cars.carMake` | `cars.make` |
| `Refuels.liter` | `refuels.quantity` |
| `Others`, `Diesel`, `Gasoline`, `Electric` | `other`, `diesel`, `gasoline`, `electric` |
| `Normal`, `Special`, `Others` | `normal`, `special`, `other` (`ac`, `dc`, `other` for refuels of an electric car) |
| `Check`, `Service`, `Wearing part`, `Crash repair` | `check`, `service`, `wearing_part`, `crash_repair` |
| `Parking`, `Velocity`, `Others` | `parking`, `velocity`, `other` |

Legacy password hashes, refresh tokens, and verification/reset tokens are not
imported: the replacement authentication model is OIDC plus magic link.

The source timezone for legacy Sequelize `DATE` values remains an explicit
decision for that future change. It must be chosen and recorded before any
timestamp conversion is run; it must not be guessed by the importer.
