-- PENDING_SEAT adalah keadaan booking sebelum kursinya benar-benar
-- tertahan di catalog-service.
--
-- Sebelumnya booking langsung lahir sebagai PENDING_PAYMENT, jadi
-- payment-service ikut membuat transaksi Snap begitu booking.created
-- terbit -- sebelum ada yang tahu kursinya masih ada. Booking yang
-- kalah balapan tetap sempat punya token, dan uang bisa masuk untuk
-- booking yang beberapa detik kemudian dibatalkan.
ALTER TABLE bookings
    DROP CONSTRAINT IF EXISTS bookings_status_check;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_status_check
        CHECK (status IN ('PENDING_SEAT', 'PENDING_PAYMENT', 'CONFIRMED', 'CANCELLED', 'EXPIRED'));

-- Booking baru masuk lewat PENDING_SEAT. Baris lama tidak diubah:
-- statusnya sudah mencerminkan kursi yang memang sudah tertahan.
ALTER TABLE bookings
    ALTER COLUMN status SET DEFAULT 'PENDING_SEAT';
