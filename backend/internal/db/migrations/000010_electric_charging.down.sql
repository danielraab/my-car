-- Postgres cannot drop enum values in place, so refuel_fuel is recreated
-- without the charging values. Refuses to run once any charge uses them.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM refuels WHERE fuel::text IN ('home', 'ac', 'dc')) THEN
        RAISE EXCEPTION 'refuels use charging subtypes home/ac/dc; cannot roll back';
    END IF;
END
$$;

ALTER TABLE refuels RENAME COLUMN quantity TO liters;

ALTER TYPE refuel_fuel RENAME TO refuel_fuel_electric;
CREATE TYPE refuel_fuel AS ENUM ('normal', 'special', 'other');
ALTER TABLE refuels ALTER COLUMN fuel TYPE refuel_fuel USING fuel::text::refuel_fuel;
DROP TYPE refuel_fuel_electric;
