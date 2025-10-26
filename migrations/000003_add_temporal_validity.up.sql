-- Add ID column as primary key (temporal ranges mean we can have multiple rows per flag)
ALTER TABLE feature_flags DROP CONSTRAINT feature_flags_pkey;
ALTER TABLE feature_flags ADD COLUMN id BIGSERIAL;
ALTER TABLE feature_flags ADD PRIMARY KEY (id);

-- Add temporal validity columns
ALTER TABLE feature_flags ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE feature_flags ADD COLUMN valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
ALTER TABLE feature_flags ADD COLUMN valid_to TIMESTAMP WITH TIME ZONE;

-- Add constraint to ensure valid_from < valid_to
ALTER TABLE feature_flags ADD CONSTRAINT chk_valid_range
    CHECK (valid_to IS NULL OR valid_from < valid_to);

-- Create unique constraint on project, stage, key, valid_from to prevent duplicate ranges
CREATE UNIQUE INDEX idx_feature_flags_unique_range
    ON feature_flags(project, stage, key, valid_from);

-- Create index for querying active flags within time ranges
CREATE INDEX idx_feature_flags_active_time
    ON feature_flags(project, stage, key, active)
    WHERE active = true;

-- Create index for temporal queries
CREATE INDEX idx_feature_flags_temporal
    ON feature_flags(project, stage, key, valid_from, valid_to);

-- Update audit_logs to reference flag ID instead of just key
ALTER TABLE audit_logs ADD COLUMN flag_id BIGINT;
CREATE INDEX idx_audit_logs_flag_id ON audit_logs(flag_id);
