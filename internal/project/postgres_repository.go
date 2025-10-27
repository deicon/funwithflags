package project

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateProject(ctx context.Context, project Project) error {
	query := `
		INSERT INTO projects (key, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`

	_, err := r.pool.Exec(ctx, query, project.Key, project.Name, project.Description)
	if err != nil {
		return mapProjectError("create project", err)
	}

	return nil
}

func (r *PostgresRepository) GetProject(ctx context.Context, key string) (Project, error) {
	query := `
		SELECT key, name, description, created_at, updated_at
		FROM projects
		WHERE key = $1
	`

	var project Project
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&project.Key,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Project{}, ErrProjectNotFound
		}
		return Project{}, fmt.Errorf("get project: %w", err)
	}
	return project, nil
}

func (r *PostgresRepository) ListProjects(ctx context.Context) ([]Project, error) {
	query := `
		SELECT key, name, description, created_at, updated_at
		FROM projects
		ORDER BY key
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]Project, 0)
	for rows.Next() {
		var project Project
		if err := rows.Scan(
			&project.Key,
			&project.Name,
			&project.Description,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}

func (r *PostgresRepository) UpdateProject(ctx context.Context, project Project) error {
	query := `
		UPDATE projects
		SET name = $2, description = $3, updated_at = NOW()
		WHERE key = $1
	`

	tag, err := r.pool.Exec(ctx, query, project.Key, project.Name, project.Description)
	if err != nil {
		return mapProjectError("update project", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteProject(ctx context.Context, key string) error {
	query := `DELETE FROM projects WHERE key = $1`

	tag, err := r.pool.Exec(ctx, query, key)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateStage(ctx context.Context, stage Stage) error {
	query := `
		INSERT INTO stages (project_key, key, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`

	_, err := r.pool.Exec(ctx, query, stage.ProjectKey, stage.Key, stage.Name, stage.Description)
	if err != nil {
		return mapStageError("create stage", err)
	}
	return nil
}

func (r *PostgresRepository) GetStage(ctx context.Context, projectKey, key string) (Stage, error) {
	query := `
		SELECT project_key, key, name, description, created_at, updated_at
		FROM stages
		WHERE project_key = $1 AND key = $2
	`

	var stage Stage
	err := r.pool.QueryRow(ctx, query, projectKey, key).Scan(
		&stage.ProjectKey,
		&stage.Key,
		&stage.Name,
		&stage.Description,
		&stage.CreatedAt,
		&stage.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Stage{}, ErrStageNotFound
		}
		return Stage{}, fmt.Errorf("get stage: %w", err)
	}

	return stage, nil
}

func (r *PostgresRepository) ListStages(ctx context.Context, projectKey string) ([]Stage, error) {
	query := `
		SELECT project_key, key, name, description, created_at, updated_at
		FROM stages
		WHERE project_key = $1
		ORDER BY key
	`

	rows, err := r.pool.Query(ctx, query, projectKey)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	defer rows.Close()

	stages := make([]Stage, 0)
	for rows.Next() {
		var stage Stage
		if err := rows.Scan(
			&stage.ProjectKey,
			&stage.Key,
			&stage.Name,
			&stage.Description,
			&stage.CreatedAt,
			&stage.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}
		stages = append(stages, stage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stages: %w", err)
	}

	return stages, nil
}

func (r *PostgresRepository) UpdateStage(ctx context.Context, stage Stage) error {
	query := `
		UPDATE stages
		SET name = $3, description = $4, updated_at = NOW()
		WHERE project_key = $1 AND key = $2
	`

	tag, err := r.pool.Exec(ctx, query, stage.ProjectKey, stage.Key, stage.Name, stage.Description)
	if err != nil {
		return mapStageError("update stage", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteStage(ctx context.Context, projectKey, key string) error {
	query := `DELETE FROM stages WHERE project_key = $1 AND key = $2`

	tag, err := r.pool.Exec(ctx, query, projectKey, key)
	if err != nil {
		return fmt.Errorf("delete stage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

func (r *PostgresRepository) ProjectHasFlags(ctx context.Context, projectKey string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM feature_flags WHERE project = $1)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, projectKey).Scan(&exists); err != nil {
		return false, fmt.Errorf("project has flags: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) StageHasFlags(ctx context.Context, projectKey, stageKey string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM feature_flags WHERE project = $1 AND stage = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, projectKey, stageKey).Scan(&exists); err != nil {
		return false, fmt.Errorf("stage has flags: %w", err)
	}
	return exists, nil
}

func mapProjectError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrProjectExists) {
		return err
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "duplicate key value"):
		return ErrProjectExists
	default:
		return fmt.Errorf("%s: %w", op, err)
	}
}

func mapStageError(op string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "duplicate key value"):
		return ErrStageExists
	case strings.Contains(msg, "violates foreign key constraint"):
		return ErrProjectNotFound
	default:
		return fmt.Errorf("%s: %w", op, err)
	}
}
