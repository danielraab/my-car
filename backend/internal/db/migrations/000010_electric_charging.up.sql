-- Electric cars record charges: the refuel subtype gains charging values
-- and the quantity column no longer names a unit. A refuel's unit (kWh or
-- the account volume unit) is implied by its car's fuel, never stored.
ALTER TYPE refuel_fuel ADD VALUE 'home';
ALTER TYPE refuel_fuel ADD VALUE 'ac';
ALTER TYPE refuel_fuel ADD VALUE 'dc';

ALTER TABLE refuels RENAME COLUMN liters TO quantity;
