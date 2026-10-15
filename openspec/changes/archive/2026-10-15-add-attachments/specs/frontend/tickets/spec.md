## ADDED Requirements

### Requirement: Ticket forms manage attachments
The ticket create form SHALL include the attachment upload component in queueing
mode, and the ticket edit form in immediate mode, as defined by
`frontend/attachments`. After a ticket is created and its queued files are
uploaded, the user SHALL return to the tickets list as before.

#### Scenario: Ticket created with a file
- **WHEN** the user queues an image on the ticket create form and submits a valid
  form
- **THEN** the ticket is created, the image is uploaded, and the user returns to
  the tickets list

#### Scenario: File added while editing
- **WHEN** the user picks a PDF on the ticket edit form
- **THEN** it is uploaded immediately and listed among the ticket's attachments

### Requirement: Ticket rows show the attachment indicator
The tickets list SHALL show the clickable attachment indicator defined by
`frontend/attachments` on rows whose `attachmentCount` is above zero,
alongside any inactive or deregistered car badge.

#### Scenario: Ticket with attachments is listed
- **WHEN** the tickets list contains a ticket with two attachments
- **THEN** its row shows the attachment button with the count 2
