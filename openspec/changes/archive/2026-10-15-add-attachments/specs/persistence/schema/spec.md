## ADDED Requirements

### Requirement: Attachments belong to exactly one entry and its account
The system SHALL persist attachment metadata (original filename, detected
content type, size in bytes, storage key, creation time) in a record owned by
exactly one car, refuel, repair or ticket, and by the account owning that entry.
The account SHALL be taken from the owning entry when the record is created. The
storage key SHALL be unique.

#### Scenario: Attachment for a refuel
- **WHEN** an attachment is stored for a refuel
- **THEN** its record references that refuel and the refuel's car's account,
  and no car, repair or ticket

#### Scenario: Attachment without or with two owners
- **WHEN** an attachment record is written that references no entry, or both a
  refuel and a ticket
- **THEN** the write is rejected

### Requirement: Attachments are deleted with their owner
Deleting a car, refuel, repair or ticket SHALL permanently delete its
attachment records, including those of a deleted car's expenses. Every
deleted attachment record SHALL queue its stored file for removal in the same
transaction, and the backend SHALL remove queued files, keeping a file queued
until its removal succeeds. A failure to remove a stored file SHALL NOT undo
the deletion; an attachment record SHALL never point to a missing file.

#### Scenario: Car with attached expenses is deleted
- **WHEN** a car is deleted that has one attachment itself and one on a refuel
- **THEN** both attachment records are deleted and both stored files are
  removed

#### Scenario: Stored file cannot be removed
- **WHEN** an entry is deleted and removing one of its stored files fails
- **THEN** the entry and its attachment records stay deleted, the failure is
  logged, and the removal is retried later
