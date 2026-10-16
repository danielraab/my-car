-- Per-account display settings. They only label values; stored amounts,
-- volumes and distances are never converted when they change.
ALTER TABLE accounts
    ADD COLUMN currency TEXT NOT NULL DEFAULT 'EUR' CHECK (currency ~ '^[A-Z]{3}$'),
    ADD COLUMN volume_unit TEXT NOT NULL DEFAULT 'liter' CHECK (volume_unit IN ('liter', 'gallon')),
    ADD COLUMN distance_unit TEXT NOT NULL DEFAULT 'kilometer' CHECK (distance_unit IN ('kilometer', 'mile'));
