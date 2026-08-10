-- Migration 001: Create payments and payment_attempts tables
-- Part of Phase 2: Persistence layer

BEGIN;

-- Payments table — source of truth for all transactions
CREATE TABLE IF NOT EXISTS payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id                VARCHAR(128) NOT NULL,
    user_id                 VARCHAR(128) NOT NULL,
    amount_minor            BIGINT NOT NULL CHECK (amount_minor > 0),
    currency                CHAR(3) NOT NULL,
    status                  VARCHAR(32) NOT NULL DEFAULT 'CREATED',
    card_token_hash         VARCHAR(64),
    provider_transaction_id VARCHAR(256),
    state_code              CHAR(2),
    metadata                JSONB,
    failure_code            VARCHAR(128),
    failure_category        VARCHAR(32),
    failure_message         VARCHAR(512),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for order lookups
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);

-- Index for reconciliation worker: find stuck payments
CREATE INDEX IF NOT EXISTS idx_payments_pending ON payments(status, updated_at)
    WHERE status IN ('CREATED', 'PENDING', 'AUTHORIZED');

-- Index for user payment history
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);

-- Payment attempts — audit trail of gateway interactions
CREATE TABLE IF NOT EXISTS payment_attempts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id              UUID NOT NULL REFERENCES payments(id),
    attempt_number          INT NOT NULL DEFAULT 1,
    provider_transaction_id VARCHAR(256),
    status                  VARCHAR(32) NOT NULL,
    provider_code           VARCHAR(128),
    provider_message        VARCHAR(512),
    response_time_ms        INT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_attempts_payment ON payment_attempts(payment_id);

-- Payment status history — immutable audit log
CREATE TABLE IF NOT EXISTS payment_status_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id  UUID NOT NULL REFERENCES payments(id),
    from_status VARCHAR(32),
    to_status   VARCHAR(32) NOT NULL,
    reason      VARCHAR(512),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_status_history_payment ON payment_status_history(payment_id);

COMMIT;
