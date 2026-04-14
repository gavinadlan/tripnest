DROP TABLE IF EXISTS processed_webhooks;

DROP INDEX IF EXISTS idx_payments_midtrans_order_id;

ALTER TABLE payments
    DROP COLUMN IF EXISTS midtrans_order_id,
    DROP COLUMN IF EXISTS snap_token,
    DROP COLUMN IF EXISTS redirect_url,
    DROP COLUMN IF EXISTS payment_type,
    DROP COLUMN IF EXISTS raw_notification,
    DROP COLUMN IF EXISTS transaction_status,
    DROP COLUMN IF EXISTS midtrans_status_code,
    DROP COLUMN IF EXISTS midtrans_gross_amount,
    DROP COLUMN IF EXISTS midtrans_fraud_status,
    DROP COLUMN IF EXISTS midtrans_transaction_time;
