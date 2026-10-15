# frontend/attachments Specification

## Purpose
Defines the shared attachment UI: the upload component with in-browser downscaling and limits, queueing on create forms and immediate changes on edit forms, the type-icon file list, the list rows' attachment popover, and the in-app attachment viewer for images and PDFs, in German and English.

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
name and size; it SHALL NOT load image contents to render previews. Activating
a stored attachment SHALL open the attachment viewer at that attachment instead
of navigating away or opening a new tab. A queued file that isn't uploaded yet
SHALL NOT be openable.

#### Scenario: Mixed attachments
- **WHEN** an entry has one JPEG and one PDF attachment
- **THEN** they are listed with the image icon and the document icon
  respectively, and no image content is requested until one is opened

#### Scenario: User opens an attachment from a list
- **WHEN** the user activates the second of an entry's three attachments in a
  file list
- **THEN** the attachment viewer opens showing that attachment, the current
  page stays where it is, and no new tab is opened

### Requirement: List rows offer a clickable attachment indicator
In the cars, refuels, repairs and tickets lists, and in the car Expenses tab
tables, a row whose `attachmentCount` is above zero SHALL show an attachment
button with that count. Activating it SHALL open a popover that loads the
entry's attachments and lists them with type icons. Activating an attachment in
the popover SHALL close the popover and then open the attachment viewer at that
attachment, so the two are never shown together. The button SHALL be keyboard
accessible and SHALL carry a localized accessible label. Rows without
attachments SHALL show no button.

#### Scenario: User opens a refuel's attachments from the list
- **WHEN** the user activates the attachment button of a refuel with two
  attachments and then activates the first file
- **THEN** the popover listed both files, it closes, and the viewer opens
  showing the first file with "1 / 2"

#### Scenario: Row without attachments
- **WHEN** a ticket has no attachments
- **THEN** its row shows no attachment button

### Requirement: Attachment strings are localized
Every string of the upload component, the attachment list, the popover and the
attachment viewer, including the downscaling hint, limit and error messages,
the viewer's position indicator, action labels and PDF fallback message, and
all accessible labels, SHALL be available in German and English.

#### Scenario: German user sees the hint
- **WHEN** the user's locale is German and they open a create form
- **THEN** the downscaling hint and all attachment labels are German

#### Scenario: German user opens the viewer
- **WHEN** the user's locale is German and they open an attachment in the viewer
- **THEN** the viewer's position indicator, action labels and accessible labels
  are German

### Requirement: Attachment viewer steps through an entry's attachments
The frontend SHALL provide one modal attachment viewer used by every place
that lists stored attachments. It SHALL show one attachment at a time from the
entry's attachments in their listed order, starting with the activated one, and
SHALL show that attachment's file name and its position as "current / total".
The user SHALL be able to go to the previous and next attachment with buttons,
with the left and right arrow keys, and by swiping on touch devices.
Navigation SHALL stop at the first and last attachment, without wrapping
around, and the previous or next button SHALL be disabled there. The viewer
SHALL close with a close button, the Escape key, or a click on the backdrop,
SHALL trap keyboard focus while open, and SHALL return focus to the element
that opened it, or, if that element is gone (such as a closed popover's file
link), to its popover button.

#### Scenario: User steps forward with the keyboard
- **WHEN** the viewer shows the first of three attachments and the user presses
  the right arrow key twice
- **THEN** the viewer shows the third attachment with "3 / 3" and the next
  button is disabled

#### Scenario: User swipes on a phone
- **WHEN** the viewer shows the second of three attachments on a touch device
  and the user swipes from right to left
- **THEN** the viewer shows the third attachment

#### Scenario: Single attachment
- **WHEN** the user opens an entry's only attachment
- **THEN** the viewer shows "1 / 1" and both navigation buttons are disabled

#### Scenario: User closes the viewer
- **WHEN** the viewer is open and the user presses Escape
- **THEN** the viewer closes and focus returns to the control that opened it

### Requirement: Viewer shows images and renders PDFs in the app
The viewer SHALL show an image attachment from its documented download URL,
scaled to fit the viewer without cropping and without being enlarged beyond
its natural size. It SHALL load an attachment's content only when that
attachment is shown or is directly before or after the shown one. It SHALL
render a PDF attachment's pages inside the viewer, fitted to the viewer's width
and scrollable, without depending on the browser's built-in PDF viewer, so that
PDFs are viewable on mobile browsers too. The PDF rendering code SHALL be
loaded only when a PDF is first shown, SHALL render pages only as they scroll
into view, and SHALL NOT execute scripts embedded in the PDF. While a file
loads, the viewer SHALL show a localized loading state. If an image or PDF
can't be loaded or rendered, the viewer SHALL show a localized message with an
action that opens the attachment in a new tab.

#### Scenario: Image is shown
- **WHEN** the user opens a JPEG attachment
- **THEN** the viewer shows the image fitted inside the viewer

#### Scenario: Multi-page PDF on a phone
- **WHEN** the user opens a three-page PDF attachment on a mobile browser
- **THEN** the viewer shows the PDF's pages at the viewer's width, and the user
  can scroll through all three pages in the viewer

#### Scenario: No PDF code before a PDF is shown
- **WHEN** the user opens an image attachment and steps only through images
- **THEN** the PDF rendering code is not downloaded

#### Scenario: Unreadable PDF
- **WHEN** a PDF attachment can't be rendered
- **THEN** the viewer shows a localized message and an action that opens the
  attachment in a new tab

### Requirement: Viewer offers read-only file actions
The viewer SHALL offer, for the shown attachment, an action that opens it in a
new browser tab and an action that downloads it under its original file name.
The viewer SHALL NOT offer to delete, rename or otherwise change an attachment,
also when opened from an edit form's attachment card.

#### Scenario: User downloads from the viewer
- **WHEN** the viewer shows a PDF named `invoice.pdf` and the user activates the
  download action
- **THEN** the browser saves the file as `invoice.pdf`

#### Scenario: Viewer opened from an edit form
- **WHEN** the user opens an attachment from an edit form's attachment card
- **THEN** the viewer shows open-in-new-tab and download actions and no delete
  action
