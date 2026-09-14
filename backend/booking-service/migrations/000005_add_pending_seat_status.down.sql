-- Tidak ada padanan PENDING_SEAT di skema lama. Booking yang masih
-- menunggu kursi dibatalkan saja: kursinya memang belum tertahan, jadi
-- tidak ada yang perlu dilepas.
UPDATE bookings
SET status = 'CANCELLED', updated_at = NOW()
WHERE status = 'PENDING_SEAT';

ALTER TABLE bookings
    ALTER COLUMN status SET DEFAULT 'PENDING_PAYMENT';

ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_status_check;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_status_check
        CHECK (status IN ('PENDING_PAYMENT', 'CONFIRMED', 'CANCELLED', 'EXPIRED'));
