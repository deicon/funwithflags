package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("project repository is required")
	}
	return &Service{repo: repo}, nil
}

func (s *Service) ListProjects(ctx context.Context) ([]Project, error) {
	projects, err := s.repo.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	if projects == nil {
		return []Project{}, nil
	}
	return projects, nil
}

func (s *Service) GetProject(ctx context.Context, key string) (Project, error) {
	if strings.TrimSpace(key) == "" {
		return Project{}, ErrInvalidProjectKey
	}
	return s.repo.GetProject(ctx, key)
}

func (s *Service) CreateProject(ctx context.Context, project Project) error {
	project.Key = strings.TrimSpace(project.Key)
	project.Name = strings.TrimSpace(project.Name)
	if project.Key == "" {
		return ErrInvalidProjectKey
	}
	if project.Name == "" {
		return ErrInvalidProjectName
	}
	return s.repo.CreateProject(ctx, project)
}

func (s *Service) UpdateProject(ctx context.Context, project Project) error {
	project.Key = strings.TrimSpace(project.Key)
	project.Name = strings.TrimSpace(project.Name)
	if project.Key == "" {
		return ErrInvalidProjectKey
	}
	if project.Name == "" {
		return ErrInvalidProjectName
	}
	return s.repo.UpdateProject(ctx, project)
}

func (s *Service) DeleteProject(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrInvalidProjectKey
	}

	hasFlags, err := s.repo.ProjectHasFlags(ctx, key)
	if err != nil {
		return err
	}
	if hasFlags {
		return ErrProjectHasFlags
	}
	return s.repo.DeleteProject(ctx, key)
}

func (s *Service) ListStages(ctx context.Context, projectKey string) ([]Stage, error) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return nil, ErrInvalidProjectKey
	}
	stages, err := s.repo.ListStages(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if stages == nil {
		return []Stage{}, nil
	}
	return stages, nil
}

func (s *Service) GetStage(ctx context.Context, projectKey, stageKey string) (Stage, error) {
	projectKey = strings.TrimSpace(projectKey)
	stageKey = strings.TrimSpace(stageKey)
	if projectKey == "" {
		return Stage{}, ErrInvalidProjectKey
	}
	if stageKey == "" {
		return Stage{}, ErrInvalidStageKey
	}
	return s.repo.GetStage(ctx, projectKey, stageKey)
}

func (s *Service) CreateStage(ctx context.Context, stage Stage) error {
	stage.ProjectKey = strings.TrimSpace(stage.ProjectKey)
	stage.Key = strings.TrimSpace(stage.Key)
	stage.Name = strings.TrimSpace(stage.Name)
	if stage.ProjectKey == "" {
		return ErrInvalidProjectKey
	}
	if stage.Key == "" {
		return ErrInvalidStageKey
	}
	if stage.Name == "" {
		return ErrInvalidStageName
	}

	if _, err := s.repo.GetProject(ctx, stage.ProjectKey); err != nil {
		return err
	}

	return s.repo.CreateStage(ctx, stage)
}

func (s *Service) UpdateStage(ctx context.Context, stage Stage) error {
	stage.ProjectKey = strings.TrimSpace(stage.ProjectKey)
	stage.Key = strings.TrimSpace(stage.Key)
	stage.Name = strings.TrimSpace(stage.Name)
	if stage.ProjectKey == "" {
		return ErrInvalidProjectKey
	}
	if stage.Key == "" {
		return ErrInvalidStageKey
	}
	if stage.Name == "" {
		return ErrInvalidStageName
	}

	return s.repo.UpdateStage(ctx, stage)
}

func (s *Service) DeleteStage(ctx context.Context, projectKey, stageKey string) error {
	projectKey = strings.TrimSpace(projectKey)
	stageKey = strings.TrimSpace(stageKey)
	if projectKey == "" {
		return ErrInvalidProjectKey
	}
	if stageKey == "" {
		return ErrInvalidStageKey
	}

	hasFlags, err := s.repo.StageHasFlags(ctx, projectKey, stageKey)
	if err != nil {
		return err
	}
	if hasFlags {
		return ErrStageHasFlags
	}

	return s.repo.DeleteStage(ctx, projectKey, stageKey)
}

// EnsureProjectAndStage creates the project and stage if they do not already exist.
func (s *Service) EnsureProjectAndStage(ctx context.Context, projectKey, projectName, stageKey, stageName string) error {
	project := Project{Key: projectKey, Name: projectName}
	if _, err := s.repo.GetProject(ctx, projectKey); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			if err := s.repo.CreateProject(ctx, project); err != nil && !errors.Is(err, ErrProjectExists) {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	stage := Stage{
		ProjectKey: projectKey,
		Key:        stageKey,
		Name:       stageName,
	}

	if _, err := s.repo.GetStage(ctx, projectKey, stageKey); err != nil {
		if errors.Is(err, ErrStageNotFound) {
			if err := s.repo.CreateStage(ctx, stage); err != nil && !errors.Is(err, ErrStageExists) {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	return nil
}
