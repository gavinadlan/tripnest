ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS midtrans_order_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS snap_token TEXT,
    ADD COLUMN IF NOT EXISTS redirect_url TEXT,
    ADD COLUMN IF NOT EXISTS payment_type VARCHAR(100),
    ADD COLUMN IF NOT EXISTS raw_notification JSONB,
    ADD COLUMN IF NOT EXISTS transaction_status VARCHAR(100),
    ADD COLUMN IF NOT EXISTS midtrans_status_code VARCHAR(50),
    ADD COLUMN IF NOT EXISTS midtrans_gross_amount VARCHAR(100),
    ADD COLUMN IF NOT EXISTS midtrans_fraud_status VARCHAR(100),
    ADD COLUMN IF NOT EXISTS midtrans_transaction_time VARCHAR(100);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_midtrans_order_id ON payments(midtrans_order_id);

CREATE TABLE IF NOT EXISTS processed_webhooks (
    id BIGSERIAL PRIMARY KEY,
    message_key VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
