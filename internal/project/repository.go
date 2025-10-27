package project

import "context"

type Repository interface {
	CreateProject(ctx context.Context, project Project) error
	GetProject(ctx context.Context, key string) (Project, error)
	ListProjects(ctx context.Context) ([]Project, error)
	UpdateProject(ctx context.Context, project Project) error
	DeleteProject(ctx context.Context, key string) error

	CreateStage(ctx context.Context, stage Stage) error
	GetStage(ctx context.Context, projectKey, key string) (Stage, error)
	ListStages(ctx context.Context, projectKey string) ([]Stage, error)
	UpdateStage(ctx context.Context, stage Stage) error
	DeleteStage(ctx context.Context, projectKey, key string) error

	ProjectHasFlags(ctx context.Context, projectKey string) (bool, error)
	StageHasFlags(ctx context.Context, projectKey, stageKey string) (bool, error)
}
