## ADDED Requirements

### Requirement: Car forms manage attachments
The car create form SHALL include the attachment upload component in queueing
mode, and the car edit form in immediate mode, as defined by
`frontend/attachments`. After a car is created and its queued files are
uploaded, the user SHALL be taken to the car's detail screen as before.

#### Scenario: Car created with its registration document
- **WHEN** the user queues a PDF on the car create form and submits a valid form
- **THEN** the car is created, the PDF is uploaded, and the user lands on the
  car's detail screen

### Requirement: Details tab shows the car's attachments
The car detail Details tab SHALL show an attachments section listing the car's
own attachments with type icons, each opening in a new tab, or a localized
"no attachments" message when there are none. It SHALL NOT list the attachments
of the car's expenses.

#### Scenario: Car with attachments
- **WHEN** the user opens the Details tab of a car with one image and one PDF
  attachment
- **THEN** both are listed with their type icons

#### Scenario: Car without attachments
- **WHEN** the user opens the Details tab of a car without attachments
- **THEN** the attachments section shows the localized "no attachments" message

### Requirement: Car rows show the attachment indicator
The cars list and the Expenses tab tables SHALL show the clickable attachment
indicator defined by `frontend/attachments` on rows whose `attachmentCount` is
above zero, alongside any inactive or deregistered badge.

#### Scenario: Deregistered car with attachments
- **WHEN** the cars list contains a deregistered car with one attachment
- **THEN** its row is muted with its badge and also shows the attachment button
