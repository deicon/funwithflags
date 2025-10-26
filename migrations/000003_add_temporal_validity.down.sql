-- Remove audit log flag_id
DROP INDEX IF EXISTS idx_audit_logs_flag_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS flag_id;

-- Remove temporal indexes
DROP INDEX IF EXISTS idx_feature_flags_temporal;
DROP INDEX IF EXISTS idx_feature_flags_active_time;
DROP INDEX IF EXISTS idx_feature_flags_unique_range;

-- Remove temporal columns and constraints
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS chk_valid_range;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS valid_to;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS valid_from;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS active;

-- Restore original primary key
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS feature_flags_pkey;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS id;
ALTER TABLE feature_flags ADD PRIMARY KEY (project, stage, key);
