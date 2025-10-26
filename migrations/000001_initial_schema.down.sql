-- Drop trigger
DROP TRIGGER IF EXISTS update_feature_flags_updated_at ON feature_flags;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop audit logs table
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_flag_key;
DROP TABLE IF EXISTS audit_logs;

-- Drop feature flags table
DROP INDEX IF EXISTS idx_feature_flags_config;
DROP INDEX IF EXISTS idx_feature_flags_enabled;
DROP TABLE IF EXISTS feature_flags;
