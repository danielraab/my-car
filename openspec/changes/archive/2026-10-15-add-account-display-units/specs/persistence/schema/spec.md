## ADDED Requirements

### Requirement: Accounts persist display settings
The system SHALL persist a currency, a volume unit and a distance unit for
every account. They SHALL default to `EUR`, `liter` and `kilometer`, both for
accounts that existed before the settings were introduced and for newly
created or imported accounts. The volume unit SHALL reject values other than
`liter` and `gallon`, the distance unit values other than `kilometer` and
`mile`, and the currency values that are not three uppercase ASCII letters.
Changing a setting SHALL NOT modify any car or expense row.

#### Scenario: Existing account after migration
- **WHEN** the migration runs against a database with existing accounts
- **THEN** every existing account has currency `EUR`, volume unit `liter` and
  distance unit `kilometer`, and no car or expense value has changed

#### Scenario: Unsupported unit is stored
- **WHEN** a persistence operation sets an account's distance unit to a value
  other than `kilometer` or `mile`
- **THEN** the value is rejected and the stored setting is unchanged
