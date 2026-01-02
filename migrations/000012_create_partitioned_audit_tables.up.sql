-- Migration: Create category-specific audit tables
-- Enables different retention policies and optimized schemas per audit category

-- =============================================================================
-- Compliance audit table (long retention, tamper-evident)
-- Retention: 7 years (regulatory requirement)
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_compliance (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id             UUID NOT NULL,
    subject             VARCHAR(255) NOT NULL DEFAULT '',
    action              VARCHAR(100) NOT NULL,
    purpose             VARCHAR(100) NOT NULL DEFAULT '',
    decision            VARCHAR(50) NOT NULL DEFAULT '',
    subject_id_hash     VARCHAR(64) NOT NULL DEFAULT '',
    request_id          VARCHAR(255) NOT NULL DEFAULT '',
    actor_id            VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX idx_audit_compliance_user_id ON audit_compliance(user_id);
CREATE INDEX idx_audit_compliance_timestamp ON audit_compliance(timestamp DESC);
CREATE INDEX idx_audit_compliance_action ON audit_compliance(action);
CREATE INDEX idx_audit_compliance_request_id ON audit_compliance(request_id) WHERE request_id != '';

COMMENT ON TABLE audit_compliance IS 'Compliance audit events. Retention: 7 years. DO NOT DELETE.';
COMMENT ON COLUMN audit_compliance.subject_id_hash IS 'SHA-256 hash of subject identifier for traceability without PII.';

-- =============================================================================
-- Security audit table (medium retention, SIEM-optimized)
-- Retention: 90 days
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_security (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    subject             VARCHAR(255) NOT NULL DEFAULT '',
    action              VARCHAR(100) NOT NULL,
    reason              VARCHAR(500) NOT NULL DEFAULT '',
    ip                  VARCHAR(45) NOT NULL DEFAULT '',
    request_id          VARCHAR(255) NOT NULL DEFAULT '',
    actor_id            VARCHAR(255) NOT NULL DEFAULT '',
    severity            VARCHAR(20) NOT NULL DEFAULT 'info'
);

CREATE INDEX idx_audit_security_timestamp ON audit_security(timestamp DESC);
CREATE INDEX idx_audit_security_action ON audit_security(action);
CREATE INDEX idx_audit_security_severity ON audit_security(severity);
CREATE INDEX idx_audit_security_ip ON audit_security(ip) WHERE ip != '';
CREATE INDEX idx_audit_security_subject ON audit_security(subject) WHERE subject != '';

COMMENT ON TABLE audit_security IS 'Security audit events. Retention: 90 days. SIEM integration target.';
COMMENT ON COLUMN audit_security.severity IS 'Event severity: info, warning, critical.';

-- =============================================================================
-- Operations audit table (short retention, high volume, partitioned)
-- Retention: 30 days
-- Partitioned by timestamp for easy retention management
-- =============================================================================
CREATE TABLE IF NOT EXISTS audit_ops (
    id                  UUID NOT NULL DEFAULT gen_random_uuid(),
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    subject             VARCHAR(255) NOT NULL DEFAULT '',
    action              VARCHAR(100) NOT NULL,
    request_id          VARCHAR(255) NOT NULL DEFAULT '',
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create initial monthly partitions (extend as needed)
-- Using YYYYMM naming convention for clarity
CREATE TABLE IF NOT EXISTS audit_ops_202501 PARTITION OF audit_ops
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE IF NOT EXISTS audit_ops_202502 PARTITION OF audit_ops
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE IF NOT EXISTS audit_ops_202503 PARTITION OF audit_ops
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE IF NOT EXISTS audit_ops_202504 PARTITION OF audit_ops
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE IF NOT EXISTS audit_ops_202505 PARTITION OF audit_ops
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE IF NOT EXISTS audit_ops_202506 PARTITION OF audit_ops
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

-- Indexes on partitioned table
CREATE INDEX idx_audit_ops_timestamp ON audit_ops(timestamp DESC);
CREATE INDEX idx_audit_ops_action ON audit_ops(action);

COMMENT ON TABLE audit_ops IS 'Operational audit events. Retention: 30 days. Partitioned monthly.';
COMMENT ON TABLE audit_ops_202501 IS 'Partition for January 2025. Drop after Feb 2025.';

-- =============================================================================
-- Outbox tables per category (optional, for topic isolation)
-- =============================================================================
-- Note: These can share the main outbox table with a 'topic' column,
-- or use separate tables for complete isolation. Using topic column for now.

ALTER TABLE outbox ADD COLUMN IF NOT EXISTS topic VARCHAR(50) NOT NULL DEFAULT 'audit.events';

CREATE INDEX IF NOT EXISTS idx_outbox_topic ON outbox(topic);

COMMENT ON COLUMN outbox.topic IS 'Kafka topic: audit.compliance, audit.security, audit.ops';
