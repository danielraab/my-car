-- Files attached to exactly one car, refuel, repair or ticket. The contents
-- live in the configured file storage under storage_key; account_id is copied
-- from the owning entry so ownership checks need no join.
CREATE TABLE attachments (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    car_id UUID REFERENCES cars(id) ON DELETE CASCADE,
    refuel_id UUID REFERENCES refuels(id) ON DELETE CASCADE,
    repair_id UUID REFERENCES repairs(id) ON DELETE CASCADE,
    ticket_id UUID REFERENCES tickets(id) ON DELETE CASCADE,
    storage_key TEXT NOT NULL UNIQUE CHECK (storage_key <> ''),
    filename TEXT NOT NULL CHECK (filename <> '' AND char_length(filename) <= 255),
    content_type TEXT NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp', 'image/gif', 'application/pdf')),
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT attachments_exactly_one_owner CHECK (num_nonnulls(car_id, refuel_id, repair_id, ticket_id) = 1)
);

CREATE INDEX attachments_car_idx ON attachments (car_id, created_at, id) WHERE car_id IS NOT NULL;
CREATE INDEX attachments_refuel_idx ON attachments (refuel_id, created_at, id) WHERE refuel_id IS NOT NULL;
CREATE INDEX attachments_repair_idx ON attachments (repair_id, created_at, id) WHERE repair_id IS NOT NULL;
CREATE INDEX attachments_ticket_idx ON attachments (ticket_id, created_at, id) WHERE ticket_id IS NOT NULL;

-- Stored files whose record is gone, including records removed by a cascade
-- from a deleted entry or car. The backend removes each file and then its
-- row, so a failed removal is retried instead of leaving the file behind.
CREATE TABLE attachment_file_deletions (
    storage_key TEXT PRIMARY KEY,
    queued_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE FUNCTION queue_attachment_file_deletion() RETURNS trigger AS $$
BEGIN
    INSERT INTO attachment_file_deletions (storage_key) VALUES (OLD.storage_key)
    ON CONFLICT (storage_key) DO NOTHING;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER attachments_queue_file_deletion
    AFTER DELETE ON attachments
    FOR EACH ROW EXECUTE FUNCTION queue_attachment_file_deletion();
