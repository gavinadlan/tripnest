-- Trip pindah ke sini dari MongoDB.
--
-- Sebelumnya trip hidup di search-service (Mongo) sementara kapasitasnya
-- hidup di sini (Postgres), dan tidak ada apa pun yang menghubungkan
-- keduanya: baris inventory tidak pernah dibuat, sehingga setiap
-- ReserveSlot mengenai 0 baris dan selalu gagal. Dengan trip dan
-- kapasitas berada di satu database, keduanya bisa dibuat dalam satu
-- transaksi dan tidak bisa lagi lepas satu sama lain.
CREATE TABLE IF NOT EXISTS trips (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    price NUMERIC(14, 2) NOT NULL CHECK (price >= 0),
    trip_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trips_destination ON trips(destination);
CREATE INDEX IF NOT EXISTS idx_trips_trip_date ON trips(trip_date);

-- Baris inventory yang dibuat manual lewat POST /inventory belum punya
-- trip. Buatkan trip penampung supaya foreign key di bawah bisa dipasang
-- tanpa membuang data yang sudah ada.
INSERT INTO trips (id, title, destination, price, trip_date)
SELECT i.resource_id, 'Trip ' || i.resource_id, 'Belum diisi', 0, CURRENT_DATE
FROM inventory i
ON CONFLICT (id) DO NOTHING;

ALTER TABLE inventory
    DROP CONSTRAINT IF EXISTS fk_inventory_trip;
ALTER TABLE inventory
    ADD CONSTRAINT fk_inventory_trip
    FOREIGN KEY (resource_id) REFERENCES trips(id) ON DELETE CASCADE;

-- Satu booking bisa memesan lebih dari satu kursi. Tanpa kolom ini,
-- pelepasan slot selalu mengembalikan 1 dan stok jadi melenceng.
ALTER TABLE inventory_reservations
    ADD COLUMN IF NOT EXISTS quantity INT NOT NULL DEFAULT 1;
ALTER TABLE inventory_reservations
    DROP CONSTRAINT IF EXISTS inventory_reservations_quantity_check;
ALTER TABLE inventory_reservations
    ADD CONSTRAINT inventory_reservations_quantity_check CHECK (quantity >= 1);
