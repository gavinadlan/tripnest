DROP TABLE IF EXISTS outbox_dead_letters;
DROP INDEX IF EXISTS idx_outbox_pending_next_attempt;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS next_attempt_at;
