ALTER TABLE feature_flags
    DROP CONSTRAINT IF EXISTS fk_feature_flags_stage;

ALTER TABLE feature_flags
    DROP CONSTRAINT IF EXISTS fk_feature_flags_project;

DROP TABLE IF EXISTS stages;
DROP TABLE IF EXISTS projects;

