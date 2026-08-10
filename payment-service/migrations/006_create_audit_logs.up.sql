CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id VARCHAR(128) NOT NULL,
    entity_type VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    actor VARCHAR(128) NOT NULL,
    old_state JSONB,
    new_state JSONB,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_id, entity_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
