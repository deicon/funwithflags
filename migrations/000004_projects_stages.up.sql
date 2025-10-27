-- Projects table to organise feature flags
CREATE TABLE IF NOT EXISTS projects (
    key VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Stages belong to projects
CREATE TABLE IF NOT EXISTS stages (
    project_key VARCHAR(255) NOT NULL,
    key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_key, key),
    CONSTRAINT fk_stages_project
        FOREIGN KEY (project_key) REFERENCES projects(key) ON DELETE CASCADE
);

-- Backfill existing projects and stages from feature flag data
INSERT INTO projects (key, name)
SELECT DISTINCT project, project
FROM feature_flags
ON CONFLICT (key) DO NOTHING;

INSERT INTO stages (project_key, key, name)
SELECT DISTINCT project, stage, stage
FROM feature_flags
ON CONFLICT (project_key, key) DO NOTHING;

-- Ensure default entries exist
INSERT INTO projects (key, name, description)
VALUES ('default', 'Default Project', 'Seed project')
ON CONFLICT (key) DO NOTHING;

INSERT INTO stages (project_key, key, name, description)
VALUES ('default', 'production', 'Production', 'Seed stage')
ON CONFLICT (project_key, key) DO NOTHING;

-- Add referential integrity constraints to feature flags
ALTER TABLE feature_flags
    ADD CONSTRAINT fk_feature_flags_project
        FOREIGN KEY (project) REFERENCES projects(key) ON DELETE RESTRICT;

ALTER TABLE feature_flags
    ADD CONSTRAINT fk_feature_flags_stage
        FOREIGN KEY (project, stage) REFERENCES stages(project_key, key) ON DELETE RESTRICT;

