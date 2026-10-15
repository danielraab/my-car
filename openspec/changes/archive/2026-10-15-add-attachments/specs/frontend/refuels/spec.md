## ADDED Requirements

### Requirement: Refuel forms manage attachments
The refuel create form SHALL include the attachment upload component in queueing
mode, and the refuel edit form in immediate mode, as defined by
`frontend/attachments`. After a refuel is created and its queued files are
uploaded, the user SHALL return to the refuels list as before.

#### Scenario: Refuel created with a file
- **WHEN** the user queues an image on the refuel create form and submits a valid
  form
- **THEN** the refuel is created, the image is uploaded, and the user returns to
  the refuels list

#### Scenario: File added while editing
- **WHEN** the user picks a PDF on the refuel edit form
- **THEN** it is uploaded immediately and listed among the refuel's attachments

### Requirement: Refuel rows show the attachment indicator
The refuels list SHALL show the clickable attachment indicator defined by
`frontend/attachments` on rows whose `attachmentCount` is above zero,
alongside any inactive or deregistered car badge.

#### Scenario: Refuel with attachments is listed
- **WHEN** the refuels list contains a refuel with two attachments
- **THEN** its row shows the attachment button with the count 2
