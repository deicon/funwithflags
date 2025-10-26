-- Remove project and stage from audit_logs
DROP INDEX IF EXISTS idx_audit_logs_project_stage_key;
CREATE INDEX idx_audit_logs_flag_key ON audit_logs(flag_key);
ALTER TABLE audit_logs DROP COLUMN IF EXISTS stage;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS project;

-- Revert feature_flags changes
DROP INDEX IF EXISTS idx_feature_flags_enabled;
DROP INDEX IF EXISTS idx_feature_flags_project_stage;

ALTER TABLE feature_flags DROP CONSTRAINT feature_flags_pkey;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS stage;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS project;
ALTER TABLE feature_flags ADD PRIMARY KEY (key);

CREATE INDEX idx_feature_flags_enabled ON feature_flags(enabled);
