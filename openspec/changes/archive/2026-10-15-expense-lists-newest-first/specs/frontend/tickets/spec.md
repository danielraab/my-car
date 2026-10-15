## ADDED Requirements

### Requirement: Tickets list shows the newest ticket first
The `/tickets` table SHALL list the tickets in the order the paginated
tickets operation returns them, which is newest first. Rows loaded with the
load-more action SHALL append below the rows already shown, so the list
stays newest first from top to bottom.

#### Scenario: Recently added ticket is at the top
- **WHEN** the user opens `/tickets` and the selected car and range include
  a ticket dated later than every other matching ticket
- **THEN** that ticket is the first row of the table

#### Scenario: Further page continues with older tickets
- **WHEN** the user activates the load-more action
- **THEN** the appended rows are all dated no later than the last row shown
  before loading
