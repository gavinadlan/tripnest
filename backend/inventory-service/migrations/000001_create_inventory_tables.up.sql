CREATE TABLE IF NOT EXISTS inventory (
    resource_id VARCHAR(255) PRIMARY KEY,
    total_slots INT NOT NULL CHECK (total_slots >= 0),
    available_slots INT NOT NULL CHECK (available_slots >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inventory_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL UNIQUE,
    resource_id VARCHAR(255) NOT NULL REFERENCES inventory(resource_id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('RESERVED', 'CONFIRMED', 'RELEASED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_reservations_resource_id ON inventory_reservations(resource_id);
CREATE INDEX IF NOT EXISTS idx_inventory_reservations_status ON inventory_reservations(status);
