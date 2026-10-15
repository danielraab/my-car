# Spec Delta

## MODIFIED Requirements

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

## ADDED Requirements

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
