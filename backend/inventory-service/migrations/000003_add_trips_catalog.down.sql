ALTER TABLE inventory_reservations
    DROP CONSTRAINT IF EXISTS inventory_reservations_quantity_check;
ALTER TABLE inventory_reservations
    DROP COLUMN IF EXISTS quantity;

ALTER TABLE inventory
    DROP CONSTRAINT IF EXISTS fk_inventory_trip;

DROP TABLE IF EXISTS trips;
