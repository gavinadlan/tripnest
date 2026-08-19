ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_quantity_check;
ALTER TABLE bookings DROP COLUMN IF EXISTS quantity;
ALTER TABLE bookings ALTER COLUMN total_amount TYPE DECIMAL(10, 2);
