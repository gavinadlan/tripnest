-- Satu booking bisa memesan lebih dari satu kursi.
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS quantity INT NOT NULL DEFAULT 1;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_quantity_check CHECK (quantity >= 1 AND quantity <= 10);

-- DECIMAL(10,2) mentok di Rp 99.999.999,99. Trip di atas seratus juta
-- akan gagal insert, jadi lebarkan sekarang selagi tabelnya masih kecil.
ALTER TABLE bookings
    ALTER COLUMN total_amount TYPE DECIMAL(14, 2);
