-- =============================================================================
-- Migration 6: Temporal redesign — split feature_flags into three entities
--
-- Old schema (single table):
--   feature_flags(id, project, stage, key, name, description, enabled, active,
--                 valid_from, valid_to, default_key, config JSONB, version,
--                 created_at, updated_at)
--   config = {"variations": [...], "rules": [...]}
--
-- New schema (three tables):
--   feature_flags  — flag identity (project/stage/key + variations)
--   flag_ranges    — temporal validity windows per flag
--   range_versions — versioned rule sets per range (draft/published)
-- =============================================================================

-- 1. Drop existing foreign key constraints on feature_flags
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS fk_feature_flags_project;
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS fk_feature_flags_stage;

-- 2. Drop old trigger (we will recreate it)
DROP TRIGGER IF EXISTS update_feature_flags_updated_at ON feature_flags;

-- 3. Rename old table to preserve data for migration
ALTER TABLE feature_flags RENAME TO feature_flags_old;

-- 4. Create new feature_flags table (identity only)
CREATE TABLE feature_flags (
    id          BIGSERIAL PRIMARY KEY,
    project     VARCHAR(255) NOT NULL,
    stage       VARCHAR(100) NOT NULL,
    key         VARCHAR(255) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT false,
    default_key VARCHAR(255) NOT NULL,
    variations  JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_feature_flags_project_stage_key UNIQUE (project, stage, key),
    CONSTRAINT fk_feature_flags_project
        FOREIGN KEY (project) REFERENCES projects(key) ON DELETE RESTRICT,
    CONSTRAINT fk_feature_flags_stage
        FOREIGN KEY (project, stage) REFERENCES stages(project_key, key) ON DELETE RESTRICT
);

CREATE INDEX idx_feature_flags_project_stage ON feature_flags(project, stage);

-- 5. Create flag_ranges table
CREATE TABLE flag_ranges (
    id         BIGSERIAL PRIMARY KEY,
    flag_id    BIGINT NOT NULL,
    active     BOOLEAN NOT NULL DEFAULT false,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
    valid_to   TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_flag_ranges_flag
        FOREIGN KEY (flag_id) REFERENCES feature_flags(id) ON DELETE CASCADE,
    CONSTRAINT chk_flag_ranges_valid_range
        CHECK (valid_to IS NULL OR valid_from < valid_to)
);

CREATE INDEX idx_flag_ranges_flag_id ON flag_ranges(flag_id);
CREATE INDEX idx_flag_ranges_active ON flag_ranges(flag_id, active) WHERE active = true;
CREATE INDEX idx_flag_ranges_temporal ON flag_ranges(flag_id, valid_from, valid_to);

-- 6. Create range_versions table
CREATE TABLE range_versions (
    id           BIGSERIAL PRIMARY KEY,
    range_id     BIGINT NOT NULL,
    version      INTEGER NOT NULL DEFAULT 1,
    status       VARCHAR(20) NOT NULL DEFAULT 'draft',
    rules        JSONB NOT NULL DEFAULT '[]'::jsonb,
    published_at TIMESTAMP WITH TIME ZONE,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_range_versions_range
        FOREIGN KEY (range_id) REFERENCES flag_ranges(id) ON DELETE CASCADE,
    CONSTRAINT chk_range_versions_status
        CHECK (status IN ('draft', 'published'))
);

CREATE INDEX idx_range_versions_range_id ON range_versions(range_id);
CREATE INDEX idx_range_versions_published ON range_versions(range_id, status) WHERE status = 'published';

-- 7. Migrate data: insert distinct flags (deduplicated by project/stage/key)
--    We pick the row with the lowest id for each unique key as the canonical identity.
INSERT INTO feature_flags (project, stage, key, name, description, enabled, default_key, variations, created_at, updated_at)
SELECT DISTINCT ON (project, stage, key)
    project,
    stage,
    key,
    name,
    COALESCE(description, ''),
    enabled,
    default_key,
    COALESCE(config->'variations', '[]'::jsonb),
    created_at,
    updated_at
FROM feature_flags_old
ORDER BY project, stage, key, id ASC;

-- 8. Migrate data: create one flag_range per old row, mapping to the new flag id
INSERT INTO flag_ranges (flag_id, active, valid_from, valid_to, created_at, updated_at)
SELECT
    nf.id,
    o.active,
    o.valid_from,
    o.valid_to,
    o.created_at,
    o.updated_at
FROM feature_flags_old o
JOIN feature_flags nf ON nf.project = o.project AND nf.stage = o.stage AND nf.key = o.key;

-- 9. Migrate data: create one published version per range with the old rules
INSERT INTO range_versions (range_id, version, status, rules, published_at, created_at)
SELECT
    fr.id,
    1,
    'published',
    COALESCE(o.config->'rules', '[]'::jsonb),
    o.updated_at,
    o.created_at
FROM feature_flags_old o
JOIN feature_flags nf ON nf.project = o.project AND nf.stage = o.stage AND nf.key = o.key
JOIN flag_ranges fr ON fr.flag_id = nf.id
    AND fr.valid_from = o.valid_from
    AND (fr.valid_to = o.valid_to OR (fr.valid_to IS NULL AND o.valid_to IS NULL));

-- 10. Update audit_logs: add range_id column if not present
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS range_id BIGINT;

-- 11. Drop old table
DROP TABLE feature_flags_old;

-- 12. Drop old indexes that referenced the old table (they were dropped automatically with rename)
DROP INDEX IF EXISTS idx_feature_flags_unique_range;
DROP INDEX IF EXISTS idx_feature_flags_active_time;
DROP INDEX IF EXISTS idx_feature_flags_temporal;
DROP INDEX IF EXISTS idx_feature_flags_enabled;
DROP INDEX IF EXISTS idx_feature_flags_config;

-- 13. Replace the old trigger function (it referenced a version column that no longer exists)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 14. Create updated_at triggers for new tables
CREATE TRIGGER update_feature_flags_updated_at
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_flag_ranges_updated_at
    BEFORE UPDATE ON flag_ranges
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
