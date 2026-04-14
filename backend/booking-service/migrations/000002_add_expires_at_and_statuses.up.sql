ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

UPDATE bookings
SET status = 'PENDING_PAYMENT'
WHERE status = 'PENDING';

UPDATE bookings
SET expires_at = created_at + INTERVAL '15 minutes'
WHERE expires_at IS NULL;

ALTER TABLE bookings
    ALTER COLUMN status SET DEFAULT 'PENDING_PAYMENT',
    ALTER COLUMN expires_at SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'bookings_status_check'
    ) THEN
        ALTER TABLE bookings
            ADD CONSTRAINT bookings_status_check
                CHECK (status IN ('PENDING_PAYMENT', 'CONFIRMED', 'CANCELLED', 'EXPIRED'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_bookings_expires_at ON bookings(expires_at);
