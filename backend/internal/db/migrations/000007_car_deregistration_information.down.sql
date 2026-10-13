ALTER TABLE cars
    DROP CONSTRAINT cars_deregistration_after_first_registration,
    DROP CONSTRAINT cars_information_not_blank,
    DROP COLUMN information,
    DROP COLUMN deregistration_date;
