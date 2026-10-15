## ADDED Requirements

### Requirement: Repair forms manage attachments
The repair create form SHALL include the attachment upload component in queueing
mode, and the repair edit form in immediate mode, as defined by
`frontend/attachments`. After a repair is created and its queued files are
uploaded, the user SHALL return to the repairs list as before.

#### Scenario: Repair created with a file
- **WHEN** the user queues an image on the repair create form and submits a valid
  form
- **THEN** the repair is created, the image is uploaded, and the user returns to
  the repairs list

#### Scenario: File added while editing
- **WHEN** the user picks a PDF on the repair edit form
- **THEN** it is uploaded immediately and listed among the repair's attachments

### Requirement: Repair rows show the attachment indicator
The repairs list SHALL show the clickable attachment indicator defined by
`frontend/attachments` on rows whose `attachmentCount` is above zero,
alongside any inactive or deregistered car badge.

#### Scenario: Repair with attachments is listed
- **WHEN** the repairs list contains a repair with two attachments
- **THEN** its row shows the attachment button with the count 2
