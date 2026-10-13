ALTER TABLE cars
    ADD COLUMN deregistration_date DATE,
    ADD COLUMN information TEXT,
    ADD CONSTRAINT cars_information_not_blank CHECK (information IS NULL OR information ~ '[^[:space:]]'),
    ADD CONSTRAINT cars_deregistration_after_first_registration CHECK (deregistration_date IS NULL OR first_registration IS NULL OR deregistration_date >= first_registration);
