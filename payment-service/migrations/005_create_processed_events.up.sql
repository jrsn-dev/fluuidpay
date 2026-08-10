-- Migration 005: Create processed_events table

BEGIN;

-- Tracks which events have been consumed by each consumer,
-- enabling idempotent message processing.
CREATE TABLE IF NOT EXISTS processed_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     VARCHAR(256) NOT NULL,
    consumer     VARCHAR(128) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique constraint prevents double-processing
CREATE UNIQUE INDEX IF NOT EXISTS uq_processed_event
    ON processed_events(event_id, consumer);

COMMIT;
