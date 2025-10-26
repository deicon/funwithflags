-- Add project and stage columns to feature_flags
ALTER TABLE feature_flags DROP CONSTRAINT feature_flags_pkey;
ALTER TABLE feature_flags ADD COLUMN project VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE feature_flags ADD COLUMN stage VARCHAR(100) NOT NULL DEFAULT 'production';

-- Create composite primary key
ALTER TABLE feature_flags ADD PRIMARY KEY (project, stage, key);

-- Update indexes
DROP INDEX IF EXISTS idx_feature_flags_enabled;
CREATE INDEX idx_feature_flags_project_stage ON feature_flags(project, stage);
CREATE INDEX idx_feature_flags_enabled ON feature_flags(project, stage, enabled);

-- Add project and stage to audit_logs
ALTER TABLE audit_logs ADD COLUMN project VARCHAR(255);
ALTER TABLE audit_logs ADD COLUMN stage VARCHAR(100);

-- Update audit log indexes
DROP INDEX IF EXISTS idx_audit_logs_flag_key;
CREATE INDEX idx_audit_logs_project_stage_key ON audit_logs(project, stage, flag_key);
