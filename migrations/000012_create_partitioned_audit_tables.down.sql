-- Rollback: Remove category-specific audit tables

-- Remove outbox topic column
ALTER TABLE outbox DROP COLUMN IF EXISTS topic;

-- Drop operations partitions and table
DROP TABLE IF EXISTS audit_ops_202506;
DROP TABLE IF EXISTS audit_ops_202505;
DROP TABLE IF EXISTS audit_ops_202504;
DROP TABLE IF EXISTS audit_ops_202503;
DROP TABLE IF EXISTS audit_ops_202502;
DROP TABLE IF EXISTS audit_ops_202501;
DROP TABLE IF EXISTS audit_ops;

-- Drop security table
DROP TABLE IF EXISTS audit_security;

-- Drop compliance table
DROP TABLE IF EXISTS audit_compliance;
