## ADDED Requirements

### Requirement: Refuels list shows the newest refuel first
The `/refuels` table SHALL list the refuels in the order the paginated
refuels operation returns them, which is newest first. Rows loaded with the
load-more action SHALL append below the rows already shown, so the list
stays newest first from top to bottom.

#### Scenario: Recently added refuel is at the top
- **WHEN** the user opens `/refuels` and the selected car and range include
  a refuel dated later than every other matching refuel
- **THEN** that refuel is the first row of the table

#### Scenario: Further page continues with older refuels
- **WHEN** the user activates the load-more action
- **THEN** the appended rows are all dated no later than the last row shown
  before loading
