# Spec Delta

## MODIFIED Requirements

### Requirement: Details tab shows the car's attachments
The car detail Details tab SHALL show an attachments section listing the car's
own attachments with type icons, each opening in the attachment viewer, or a
localized "no attachments" message when there are none. It SHALL NOT list the
attachments of the car's expenses.

#### Scenario: Car with attachments
- **WHEN** the user opens the Details tab of a car with one image and one PDF
  attachment
- **THEN** both are listed with their type icons

#### Scenario: User opens a car attachment
- **WHEN** the user activates the PDF listed on a car's Details tab
- **THEN** the attachment viewer opens showing that PDF, and the user can step
  to the car's image attachment

#### Scenario: Car without attachments
- **WHEN** the user opens the Details tab of a car without attachments
- **THEN** the attachments section shows the localized "no attachments" message
