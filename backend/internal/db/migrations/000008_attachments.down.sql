-- Stored files are not removed; delete the file storage contents manually.
DROP TRIGGER attachments_queue_file_deletion ON attachments;
DROP FUNCTION queue_attachment_file_deletion();
DROP TABLE attachment_file_deletions;
DROP TABLE attachments;
