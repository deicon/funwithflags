package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/deicon/funwithflags/internal/project"
)

type projectRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type stageRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func newListProjectsHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := service.ListProjects(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list projects: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
	}
}

func newGetProjectHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("project")
		projectObj, err := service.GetProject(r.Context(), key)
		if err != nil {
			if errors.Is(err, project.ErrProjectNotFound) {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get project: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, projectObj)
	}
}

func newCreateProjectHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req projectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		projectObj := project.Project{
			Key:         req.Key,
			Name:        req.Name,
			Description: req.Description,
		}

		if err := service.CreateProject(r.Context(), projectObj); err != nil {
			switch {
			case errors.Is(err, project.ErrInvalidProjectKey),
				errors.Is(err, project.ErrInvalidProjectName):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, project.ErrProjectExists):
				writeError(w, http.StatusConflict, "project with that key already exists")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create project: %v", err))
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{
			"message": "project created",
			"key":     projectObj.Key,
		})
	}
}

func newUpdateProjectHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("project")
		var req projectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		projectObj := project.Project{
			Key:         key,
			Name:        req.Name,
			Description: req.Description,
		}

		if err := service.UpdateProject(r.Context(), projectObj); err != nil {
			switch {
			case errors.Is(err, project.ErrInvalidProjectKey),
				errors.Is(err, project.ErrInvalidProjectName):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, project.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update project: %v", err))
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "project updated",
			"key":     key,
		})
	}
}

func newDeleteProjectHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("project")
		if err := service.DeleteProject(r.Context(), key); err != nil {
			switch {
			case errors.Is(err, project.ErrInvalidProjectKey):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, project.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			case errors.Is(err, project.ErrProjectHasFlags):
				writeError(w, http.StatusConflict, "project has associated flags")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete project: %v", err))
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "project deleted",
			"key":     key,
		})
	}
}

func newListStagesHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectKey := r.PathValue("project")
		stages, err := service.ListStages(r.Context(), projectKey)
		if err != nil {
			switch {
			case errors.Is(err, project.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list stages: %v", err))
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"stages": stages})
	}
}

func newGetStageHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectKey := r.PathValue("project")
		stageKey := r.PathValue("stage")
		stageObj, err := service.GetStage(r.Context(), projectKey, stageKey)
		if err != nil {
			switch {
			case errors.Is(err, project.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			case errors.Is(err, project.ErrStageNotFound):
				writeError(w, http.StatusNotFound, "stage not found")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get stage: %v", err))
			}
			return
		}
		writeJSON(w, http.StatusOK, stageObj)
	}
}

func newCreateStageHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectKey := r.PathValue("project")
		var req stageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		stageObj := project.Stage{
			ProjectKey:  projectKey,
			Key:         req.Key,
			Name:        req.Name,
			Description: req.Description,
		}

		if err := service.CreateStage(r.Context(), stageObj); err != nil {
			switch {
			case errors.Is(err, project.ErrInvalidProjectKey),
				errors.Is(err, project.ErrInvalidStageKey),
				errors.Is(err, project.ErrInvalidStageName):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, project.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			case errors.Is(err, project.ErrStageExists):
				writeError(w, http.StatusConflict, "stage with that key already exists")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create stage: %v", err))
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{
			"message": "stage created",
			"key":     stageObj.Key,
		})
	}
}

func newUpdateStageHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectKey := r.PathValue("project")
		stageKey := r.PathValue("stage")
		var req stageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		stageObj := project.Stage{
			ProjectKey:  projectKey,
			Key:         stageKey,
			Name:        req.Name,
			Description: req.Description,
		}

		if err := service.UpdateStage(r.Context(), stageObj); err != nil {
			switch {
			case errors.Is(err, project.ErrInvalidProjectKey),
				errors.Is(err, project.ErrInvalidStageKey),
				errors.Is(err, project.ErrInvalidStageName):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, project.ErrStageNotFound):
				writeError(w, http.StatusNotFound, "stage not found")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update stage: %v", err))
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "stage updated",
			"key":     stageKey,
		})
	}
}

func newDeleteStageHandler(service *project.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectKey := r.PathValue("project")
		stageKey := r.PathValue("stage")
		if err := service.DeleteStage(r.Context(), projectKey, stageKey); err != nil {
			switch {
			case errors.Is(err, project.ErrInvalidProjectKey),
				errors.Is(err, project.ErrInvalidStageKey):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, project.ErrStageNotFound):
				writeError(w, http.StatusNotFound, "stage not found")
			case errors.Is(err, project.ErrStageHasFlags):
				writeError(w, http.StatusConflict, "stage has associated flags")
			default:
				writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete stage: %v", err))
			}
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "stage deleted",
			"key":     stageKey,
		})
	}
}
