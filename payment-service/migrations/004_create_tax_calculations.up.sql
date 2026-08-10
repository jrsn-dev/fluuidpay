-- Migration 004: Create tax_calculations table

BEGIN;

CREATE TABLE IF NOT EXISTS tax_calculations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id        UUID NOT NULL REFERENCES payments(id),
    ibs_amount_minor  BIGINT NOT NULL DEFAULT 0,
    cbs_amount_minor  BIGINT NOT NULL DEFAULT 0,
    total_tax_minor   BIGINT NOT NULL DEFAULT 0,
    base_amount_minor BIGINT NOT NULL,
    currency          CHAR(3) NOT NULL,
    rule_version      VARCHAR(64) NOT NULL,
    jurisdiction      VARCHAR(32),
    country_code      CHAR(2),
    state_code        CHAR(2),
    city_code         VARCHAR(16),
    product_type      VARCHAR(128),
    customer_type     VARCHAR(32),
    calculated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tax_calc_payment ON tax_calculations(payment_id);
CREATE INDEX IF NOT EXISTS idx_tax_calc_rule ON tax_calculations(rule_version);

COMMIT;
