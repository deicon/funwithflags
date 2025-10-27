package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/deicon/funwithflags/internal/auth"
	"github.com/deicon/funwithflags/internal/flag"
)

func newListFlagsHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		if project == "" {
			writeError(w, http.StatusBadRequest, "project is required")
			return
		}

		stage := r.PathValue("stage")
		if stage == "" {
			writeError(w, http.StatusBadRequest, "stage is required")
			return
		}

		flags, err := service.ListFlags(r.Context(), project, stage)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list flags: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"flags": flags,
		})
	}
}

func newGetFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		if project == "" {
			writeError(w, http.StatusBadRequest, "project is required")
			return
		}

		stage := r.PathValue("stage")
		if stage == "" {
			writeError(w, http.StatusBadRequest, "stage is required")
			return
		}

		key := r.PathValue("key")
		if key == "" {
			writeError(w, http.StatusBadRequest, "flag key is required")
			return
		}

		f, err := service.GetFlag(r.Context(), project, stage, key)
		if err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, f)
	}
}

func newCreateFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		if project == "" {
			writeError(w, http.StatusBadRequest, "project is required")
			return
		}

		stage := r.PathValue("stage")
		if stage == "" {
			writeError(w, http.StatusBadRequest, "stage is required")
			return
		}

		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		// Set project and stage from path
		f.Project = project
		f.Stage = stage

		performedBy := requestUser(r)

		if err := service.CreateFlag(r.Context(), f, performedBy); err != nil {
			if errors.Is(err, flag.ErrInvalidFlag) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid flag: %v", err))
				return
			}
			if errors.Is(err, flag.ErrRangeOverlap) {
				writeError(w, http.StatusConflict, "flag range overlaps with existing range")
				return
			}
			if errors.Is(err, flag.ErrInvalidTimeRange) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid time range: %v", err))
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create flag: %v", err))
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{
			"message": "flag created successfully",
			"key":     f.Key,
		})
	}
}

func newUpdateFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "flag ID is required")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		// Set ID from path
		f.ID = id

		performedBy := requestUser(r)

		if err := service.UpdateFlag(r.Context(), f, performedBy); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			if errors.Is(err, flag.ErrInvalidFlag) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid flag: %v", err))
				return
			}
			if errors.Is(err, flag.ErrFlagConflict) {
				writeError(w, http.StatusConflict, "flag conflict - version mismatch")
				return
			}
			if errors.Is(err, flag.ErrRangeOverlap) {
				writeError(w, http.StatusConflict, "flag range overlaps with existing range")
				return
			}
			if errors.Is(err, flag.ErrInvalidTimeRange) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid time range: %v", err))
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "flag updated successfully",
			"id":      id,
		})
	}
}

func newDeleteFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "flag ID is required")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.DeleteFlag(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "flag deleted successfully",
			"id":      id,
		})
	}
}

func newGetAuditLogsHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		if project == "" {
			writeError(w, http.StatusBadRequest, "project is required")
			return
		}

		stage := r.PathValue("stage")
		if stage == "" {
			writeError(w, http.StatusBadRequest, "stage is required")
			return
		}

		key := r.PathValue("key")
		if key == "" {
			writeError(w, http.StatusBadRequest, "flag key is required")
			return
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 100
		if limitStr != "" {
			var err error
			limit, err = strconv.Atoi(limitStr)
			if err != nil || limit <= 0 {
				writeError(w, http.StatusBadRequest, "invalid limit parameter: limit must be a positive integer")
				return
			}
		}

		logs, err := service.GetAuditLogs(r.Context(), project, stage, key, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get audit logs: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"logs": logs,
		})
	}
}

// Temporal range handlers

func newGetFlagByIDHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "flag ID is required")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		f, err := service.GetFlagByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, f)
	}
}

func newGetFlagRangesHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		if project == "" {
			writeError(w, http.StatusBadRequest, "project is required")
			return
		}

		stage := r.PathValue("stage")
		if stage == "" {
			writeError(w, http.StatusBadRequest, "stage is required")
			return
		}

		key := r.PathValue("key")
		if key == "" {
			writeError(w, http.StatusBadRequest, "flag key is required")
			return
		}

		ranges, err := service.GetFlagRanges(r.Context(), project, stage, key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get flag ranges: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ranges": ranges,
		})
	}
}

func newActivateFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "flag ID is required")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.ActivateFlag(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			if errors.Is(err, flag.ErrActiveRangeOverlap) {
				writeError(w, http.StatusConflict, "cannot activate: overlaps with active range")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to activate flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "flag activated successfully",
			"id":      id,
		})
	}
}

func newDeactivateFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			writeError(w, http.StatusBadRequest, "flag ID is required")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.DeactivateFlag(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to deactivate flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "flag deactivated successfully",
			"id":      id,
		})
	}
}

func requestUser(r *http.Request) string {
	if user, ok := auth.UserFromContext(r.Context()); ok && user.Username != "" {
		return user.Username
	}
	return "unknown"
}
