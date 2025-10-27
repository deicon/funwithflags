package project

import (
	"context"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu       sync.RWMutex
	projects map[string]Project
	stages   map[string]map[string]Stage
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		projects: make(map[string]Project),
		stages:   make(map[string]map[string]Stage),
	}
}

func (r *InMemoryRepository) CreateProject(_ context.Context, project Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.projects[project.Key]; exists {
		return ErrProjectExists
	}

	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	r.projects[project.Key] = project
	return nil
}

func (r *InMemoryRepository) GetProject(_ context.Context, key string) (Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.projects[key]
	if !ok {
		return Project{}, ErrProjectNotFound
	}
	return project, nil
}

func (r *InMemoryRepository) ListProjects(_ context.Context) ([]Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.projects))
	for key := range r.projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]Project, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.projects[key])
	}
	return result, nil
}

func (r *InMemoryRepository) UpdateProject(_ context.Context, project Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.projects[project.Key]
	if !ok {
		return ErrProjectNotFound
	}
	project.CreatedAt = existing.CreatedAt
	project.UpdatedAt = time.Now()
	r.projects[project.Key] = project
	return nil
}

func (r *InMemoryRepository) DeleteProject(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[key]; !ok {
		return ErrProjectNotFound
	}
	delete(r.projects, key)
	delete(r.stages, key)
	return nil
}

func (r *InMemoryRepository) CreateStage(_ context.Context, stage Stage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[stage.ProjectKey]; !ok {
		return ErrProjectNotFound
	}

	projectStages, ok := r.stages[stage.ProjectKey]
	if !ok {
		projectStages = make(map[string]Stage)
		r.stages[stage.ProjectKey] = projectStages
	}

	if _, exists := projectStages[stage.Key]; exists {
		return ErrStageExists
	}

	now := time.Now()
	stage.CreatedAt = now
	stage.UpdatedAt = now
	projectStages[stage.Key] = stage
	return nil
}

func (r *InMemoryRepository) GetStage(_ context.Context, projectKey, key string) (Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.projects[projectKey]; !ok {
		return Stage{}, ErrProjectNotFound
	}

	projectStages, ok := r.stages[projectKey]
	if !ok {
		return Stage{}, ErrStageNotFound
	}

	stage, ok := projectStages[key]
	if !ok {
		return Stage{}, ErrStageNotFound
	}

	return stage, nil
}

func (r *InMemoryRepository) ListStages(_ context.Context, projectKey string) ([]Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.projects[projectKey]; !ok {
		return nil, ErrProjectNotFound
	}

	projectStages, ok := r.stages[projectKey]
	if !ok {
		return []Stage{}, nil
	}

	keys := make([]string, 0, len(projectStages))
	for key := range projectStages {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]Stage, 0, len(keys))
	for _, key := range keys {
		result = append(result, projectStages[key])
	}
	return result, nil
}

func (r *InMemoryRepository) UpdateStage(_ context.Context, stage Stage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	projectStages, ok := r.stages[stage.ProjectKey]
	if !ok {
		return ErrStageNotFound
	}

	existing, ok := projectStages[stage.Key]
	if !ok {
		return ErrStageNotFound
	}

	stage.CreatedAt = existing.CreatedAt
	stage.UpdatedAt = time.Now()
	projectStages[stage.Key] = stage
	return nil
}

func (r *InMemoryRepository) DeleteStage(_ context.Context, projectKey, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	projectStages, ok := r.stages[projectKey]
	if !ok {
		return ErrStageNotFound
	}
	if _, ok := projectStages[key]; !ok {
		return ErrStageNotFound
	}

	delete(projectStages, key)
	return nil
}

func (r *InMemoryRepository) ProjectHasFlags(context.Context, string) (bool, error) {
	return false, nil
}

func (r *InMemoryRepository) StageHasFlags(context.Context, string, string) (bool, error) {
	return false, nil
}
