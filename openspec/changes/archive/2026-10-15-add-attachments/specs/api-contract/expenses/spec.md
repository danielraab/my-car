## ADDED Requirements

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
