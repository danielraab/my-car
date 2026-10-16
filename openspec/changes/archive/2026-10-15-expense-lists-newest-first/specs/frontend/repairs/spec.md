## ADDED Requirements

### Requirement: Repairs list shows the newest repair first
The `/repairs` table SHALL list the repairs in the order the paginated
repairs operation returns them, which is newest first. Rows loaded with the
load-more action SHALL append below the rows already shown, so the list
stays newest first from top to bottom.

#### Scenario: Recently added repair is at the top
- **WHEN** the user opens `/repairs` and the selected car and range include
  a repair dated later than every other matching repair
- **THEN** that repair is the first row of the table

#### Scenario: Further page continues with older repairs
- **WHEN** the user activates the load-more action
- **THEN** the appended rows are all dated no later than the last row shown
  before loading
