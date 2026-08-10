-- Migration 002: Create idempotency_records table

BEGIN;

CREATE TABLE IF NOT EXISTS idempotency_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(256) NOT NULL,
    operation       VARCHAR(64) NOT NULL DEFAULT 'create_payment',
    payload_hash    VARCHAR(128) NOT NULL,
    status          VARCHAR(32) NOT NULL DEFAULT 'PROCESSING',
    response        JSONB,
    http_code       INT,
    transaction_id  VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

-- Unique constraint ensures only one record per key+operation combination
CREATE UNIQUE INDEX IF NOT EXISTS uq_idempotency_key_operation
    ON idempotency_records(idempotency_key, operation);

-- Index for cleanup of expired records
CREATE INDEX IF NOT EXISTS idx_idempotency_expires
    ON idempotency_records(expires_at)
    WHERE status = 'PROCESSING';

COMMIT;
