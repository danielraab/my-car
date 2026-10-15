# frontend/attachments Specification

## Purpose
Defines the shared attachment UI: the upload component with in-browser downscaling and limits, queueing on create forms and immediate changes on edit forms, the type-icon file list, and the list rows' attachment popover, in German and English.

## Requirements
### Requirement: Upload component accepts images and PDFs within the limits
The frontend SHALL provide one attachment upload component used by every car,
refuel, repair and ticket form. It SHALL let the user pick several files at
once, with the file picker restricted to images and PDF documents, and on
mobile devices it SHALL allow taking a photo. It SHALL read the limits from the
documented upload-limits operation and SHALL NOT start an upload for a file
whose type is not allowed, that is still larger than the maximum size after
downscaling, or that would bring the entry above the maximum number of files.
Each refused file SHALL be named with a localized reason.

#### Scenario: Unsupported file is refused
- **WHEN** the user picks a `.docx` file
- **THEN** the component shows a localized message that only images and PDF
  documents are allowed, and nothing is uploaded

#### Scenario: Oversized PDF is refused
- **WHEN** the user picks a 14 MB PDF and the maximum size is 10 MB
- **THEN** the component shows a localized "file too large (max. 10 MB)"
  message for it, and nothing is uploaded

#### Scenario: Eleventh file is refused
- **WHEN** an entry already has or has queued 10 files and the user picks one
  more
- **THEN** the component shows a localized limit message and does not accept it

### Requirement: Large images are downscaled before upload
Before an upload, the frontend SHALL scale down any JPEG, PNG or WebP image whose
longer edge exceeds 2000 pixels so that its longer edge is 2000 pixels,
preserving its aspect ratio and applying its EXIF orientation. A JPEG or WebP
SHALL be re-encoded in its own format at quality 0.85, and a PNG SHALL stay
PNG. Images within the size, GIFs and PDFs SHALL be uploaded unchanged. The size
limit SHALL be checked against the result of downscaling. The component SHALL
always show a localized hint stating that large images are scaled down before
upload, together with the allowed types, the maximum file size and the maximum
number of files.

#### Scenario: Phone photo is downscaled
- **WHEN** the user picks a 4000×3000 pixel, 12 MB JPEG and the maximum size is
  10 MB
- **THEN** a 2000×1500 pixel JPEG is uploaded and accepted

#### Scenario: Small image is untouched
- **WHEN** the user picks a 1200×800 pixel PNG
- **THEN** the original file is uploaded unchanged

#### Scenario: GIF is untouched
- **WHEN** the user picks a 3000×2000 pixel GIF within the size limit
- **THEN** the original file is uploaded unchanged

#### Scenario: Hint is visible
- **WHEN** the user opens any create or edit form with the upload component
- **THEN** the component shows the localized downscaling hint with the allowed
  types and limits

### Requirement: Create forms queue files until the entry is saved
On a create form the component SHALL keep picked files in a local queue, each
listed with its type icon, name and size and removable before submission.
After the entry has been created, the frontend SHALL upload the queued files one
after another, showing progress. If all uploads succeed, the form SHALL continue
as it does without attachments. If any upload fails, the frontend SHALL navigate
to the new entry's edit screen and show a localized message naming the files
that were not uploaded; the entry itself SHALL stay saved.

#### Scenario: Refuel created with two receipts
- **WHEN** the user queues two images on the refuel create form and submits a
  valid form
- **THEN** the refuel is created, both files are uploaded, and the user returns
  to the refuels list

#### Scenario: One upload fails after create
- **WHEN** the entry is created but one of the queued uploads fails
- **THEN** the user lands on the entry's edit screen with a localized message
  naming the file that was not uploaded

#### Scenario: Queued file removed
- **WHEN** the user removes a queued file before submitting
- **THEN** that file is not uploaded

### Requirement: Edit forms upload and delete files immediately
On an edit form the component SHALL list the entry's attachments and SHALL
upload a picked file immediately, independent of saving the form. Deleting an
attachment SHALL use the two-step confirmation used for deleting entries and
SHALL take effect immediately. The attachment section SHALL be visually separate
from the form fields, so that cancelling the form is not mistaken for undoing
attachment changes.

#### Scenario: File added on edit
- **WHEN** the user picks a PDF on a repair's edit form
- **THEN** it is uploaded at once and listed, even if the user then leaves
  without saving

#### Scenario: File deleted on edit
- **WHEN** the user confirms deletion of an attachment in both steps
- **THEN** the attachment is deleted and disappears from the list

### Requirement: Attachments are shown with a type icon, not a thumbnail
Wherever attachments or queued files are listed, the frontend SHALL show an
image icon for image types and a document icon for PDFs, followed by the file
name and size; it SHALL NOT load image contents to render previews. Opening an
attachment SHALL open its download in a new browser tab.

#### Scenario: Mixed attachments
- **WHEN** an entry has one JPEG and one PDF attachment
- **THEN** they are listed with the image icon and the document icon
  respectively, and no image content is requested until one is opened

### Requirement: List rows offer a clickable attachment indicator
In the cars, refuels, repairs and tickets lists, and in the car Expenses tab
tables, a row whose `attachmentCount` is above zero SHALL show an attachment
button with that count. Activating it SHALL open a popover that loads the
entry's attachments and lists them with type icons; each opens in a new tab. The
button SHALL be keyboard accessible and SHALL carry a localized accessible
label. Rows without attachments SHALL show no button.

#### Scenario: User opens a refuel's attachments from the list
- **WHEN** the user activates the attachment button of a refuel with two
  attachments
- **THEN** a popover lists both files, and activating one opens it in a new tab

#### Scenario: Row without attachments
- **WHEN** a ticket has no attachments
- **THEN** its row shows no attachment button

### Requirement: Attachment strings are localized
Every string of the upload component, the attachment list and the popover,
including the downscaling hint, limit and error messages and accessible labels,
SHALL be available in German and English.

#### Scenario: German user sees the hint
- **WHEN** the user's locale is German and they open a create form
- **THEN** the downscaling hint and all attachment labels are German
