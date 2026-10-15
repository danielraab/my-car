# frontend/profile Specification

## Purpose

Defines the authenticated profile screen where a user reviews their account
email and maintains their first and last name and their currency and unit
display settings, in German and English.

## Requirements

### Requirement: Profile screen shows the caller's account details
The frontend SHALL render the `/profile` route inside the authenticated
application shell and SHALL load the caller's profile from the documented
profile operation. It SHALL display the email address as read-only and
SHALL explain, in the active locale, that the email address cannot be
changed. It SHALL NOT display session, token, or identity-provider details.
While loading, it SHALL show a localized loading state; a load failure SHALL
produce a localized error with a retry action rather than an empty form.

#### Scenario: User opens the profile screen
- **WHEN** an authenticated user navigates to `/profile`
- **THEN** the screen shows their email as a non-editable value with an
  explanation, and editable first-name and last-name fields prefilled with
  their current values

#### Scenario: Profile cannot be loaded
- **WHEN** the profile operation fails with a transport or server error
- **THEN** the screen shows a localized error with a retry action and no
  editable form

### Requirement: User can update their name from the profile screen
The frontend SHALL let the user edit and save their first and last name,
together with the display settings defined below, submitting only those
fields; it SHALL NOT submit the email address. While a save is in progress
the submit action SHALL be disabled. After a successful save the screen
SHALL show a localized, accessibly announced confirmation, and every part of
the application that displays the caller's profile SHALL reflect the saved
values without a page reload. A validation rejection SHALL be shown as a
localized message associated with the affected field; any other failure
SHALL be shown as a localized error that keeps the user's unsaved input.

#### Scenario: User saves a new name
- **WHEN** the user changes their first name and submits the form
- **THEN** the frontend sends only name and display-setting fields, shows a
  success confirmation, and displays the saved name

#### Scenario: Backend rejects a name
- **WHEN** the backend responds `400` with field details for `lastName`
- **THEN** the frontend shows a localized validation message on the last-name
  field and does not report success

#### Scenario: Save fails unexpectedly
- **WHEN** the save fails with a transport or server error
- **THEN** the frontend shows a localized error and the entered values remain
  in the form

### Requirement: Profile screen is fully localized
Every user-facing string on the profile screen, including validation and
error messages, SHALL be available in German and English and SHALL follow the
active locale.

#### Scenario: User switches language on the profile screen
- **WHEN** the user switches the language from English to German on `/profile`
- **THEN** all labels, hints, and messages on the screen are shown in German

### Requirement: User can choose display settings on the profile screen
The profile screen SHALL show the caller's currency, volume unit and distance
unit as editable choices prefilled with their current values. The currency
choice SHALL offer every ISO 4217 currency the browser knows, each labelled
with its localized name and code, and SHALL always include the caller's
current currency. The volume unit choice SHALL offer litres and gallons and
the distance unit choice kilometres and miles, localized. The screen SHALL
explain, in the active locale, that changing these settings only changes how
values are labelled and does not convert existing amounts, volumes or
distances. Saving SHALL submit the changed settings together with the name
fields through the profile update operation and SHALL follow the same
in-progress, success, validation-error and failure behaviour as saving a
name; after a successful save, every screen SHALL use the new settings
without a page reload.

#### Scenario: User opens the settings
- **WHEN** an account with currency EUR, litres and kilometres opens
  `/profile`
- **THEN** the currency choice shows the euro, the volume choice litres and
  the distance choice kilometres, with a hint that values are not converted

#### Scenario: User switches to US dollars and miles
- **WHEN** the user selects US dollar and miles and saves
- **THEN** the frontend sends `currency` `USD` and `distanceUnit` `mile`,
  shows the success confirmation, and amounts and odometer readings on other
  screens are labelled in US dollars and miles without a reload

#### Scenario: Backend rejects the currency
- **WHEN** the backend responds `400` with field details for `currency`
- **THEN** the frontend shows a localized validation message on the currency
  choice and does not report success

#### Scenario: Settings in German
- **WHEN** the active locale is German
- **THEN** the currency names, unit names, labels and the no-conversion hint
  are shown in German
