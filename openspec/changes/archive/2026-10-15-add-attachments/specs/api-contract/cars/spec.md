## ADDED Requirements

### Requirement: Car representations carry the attachment count
Every car representation returned by the cars operations SHALL contain a
read-only integer `attachmentCount`: the number of attachments of the car
itself, not of its expenses. `CarInput` and `CarUpdate` SHALL NOT accept it.

#### Scenario: Car with attachments is listed
- **WHEN** a caller lists their cars and one car has two attachments
- **THEN** that car's `attachmentCount` is `2` and every other car's is `0`

#### Scenario: Client supplies the count
- **WHEN** a car creation or update request contains `attachmentCount`
- **THEN** the response is `400` with the common validation error identifying
  `attachmentCount`
