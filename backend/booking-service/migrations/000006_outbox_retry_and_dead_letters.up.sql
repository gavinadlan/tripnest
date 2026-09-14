-- Outbox worker: backoff eksponensial dan dead letter.
--
-- next_attempt_at menunda event yang gagal terbit; worker hanya mengambil
-- baris yang sudah waktunya. Setelah OUTBOX_MAX_RETRY percobaan, baris
-- diberi status FAILED dan disalin ke outbox_dead_letters. Sebelumnya
-- MarkOutboxEventFailed hanya menaikkan retry_count dan event yang rusak
-- permanen dicoba lagi tiap 2 detik selamanya.
ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_outbox_pending_next_attempt
    ON outbox_events (next_attempt_at, created_at)
    WHERE status = 'PENDING';

CREATE TABLE IF NOT EXISTS outbox_dead_letters (
    id UUID PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    message_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    retry_count INT NOT NULL,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    failed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    replayed_at TIMESTAMP WITH TIME ZONE
);
