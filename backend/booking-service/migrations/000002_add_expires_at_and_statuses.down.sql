ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_status_check;

DROP INDEX IF EXISTS idx_bookings_expires_at;

ALTER TABLE bookings
    ALTER COLUMN status SET DEFAULT 'PENDING',
    DROP COLUMN IF EXISTS expires_at;
