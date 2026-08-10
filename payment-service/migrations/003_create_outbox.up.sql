-- Migration 003: Create outbox_events table

BEGIN;

CREATE TABLE IF NOT EXISTS outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id    VARCHAR(128) NOT NULL,
    event_type      VARCHAR(64) NOT NULL,
    payload         JSONB NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    attempt_count   INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 5,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT,
    locked_at       TIMESTAMPTZ,
    locked_by       VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ
);

-- Index for publisher polling: fetch pending events ready for publishing
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, next_attempt_at)
    WHERE status IN ('PENDING', 'FAILED_RETRYABLE');

-- Index for monitoring: detect stuck events
CREATE INDEX IF NOT EXISTS idx_outbox_locked ON outbox_events(locked_at)
    WHERE status = 'PUBLISHING' AND locked_at IS NOT NULL;

-- Index for aggregate-level event history
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate ON outbox_events(aggregate_id);

COMMIT;
