package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/deicon/funwithflags/internal/auth"
	"github.com/deicon/funwithflags/internal/flag"
)

// --- Flag identity handlers ---

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
			writeError(w, http.StatusInternalServerError, "failed to list flags")
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
			writeError(w, http.StatusInternalServerError, "failed to get flag")
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
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		f.Project = project
		f.Stage = stage

		performedBy := requestUser(r)

		created, err := service.CreateFlag(r.Context(), f, performedBy)
		if err != nil {
			if errors.Is(err, flag.ErrInvalidFlag) {
				writeError(w, http.StatusBadRequest, "invalid flag")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create flag")
			return
		}

		writeJSON(w, http.StatusCreated, created)
	}
}

func newGetFlagByIDHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
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
			writeError(w, http.StatusInternalServerError, "failed to get flag")
			return
		}

		writeJSON(w, http.StatusOK, f)
	}
}

func newUpdateFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		f.ID = id
		performedBy := requestUser(r)

		if err := service.UpdateFlag(r.Context(), f, performedBy); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			if errors.Is(err, flag.ErrInvalidFlag) {
				writeError(w, http.StatusBadRequest, "invalid flag")
				return
			}
			if errors.Is(err, flag.ErrFlagConflict) {
				writeError(w, http.StatusConflict, "flag conflict - version mismatch")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update flag")
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
		id, err := parseIDParam(r, "id")
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
			writeError(w, http.StatusInternalServerError, "failed to delete flag")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "flag deleted successfully",
			"id":      id,
		})
	}
}

// --- Range handlers ---

func newListRangesHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagID, err := parseIDParam(r, "flagId")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		ranges, err := service.ListRanges(r.Context(), flagID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list ranges")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ranges": ranges,
		})
	}
}

func newCreateRangeHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagID, err := parseIDParam(r, "flagId")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		var req struct {
			ValidFrom time.Time  `json:"validFrom"`
			ValidTo   *time.Time `json:"validTo,omitempty"`
			Rules     []flag.Rule `json:"rules"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		performedBy := requestUser(r)

		rng := flag.FlagRange{
			FlagID:    flagID,
			ValidFrom: req.ValidFrom,
			ValidTo:   req.ValidTo,
		}

		created, version, err := service.CreateRange(r.Context(), rng, req.Rules, performedBy)
		if err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			if errors.Is(err, flag.ErrRangeOverlap) {
				writeError(w, http.StatusConflict, "range overlaps with existing range")
				return
			}
			if errors.Is(err, flag.ErrInvalidTimeRange) {
				writeError(w, http.StatusBadRequest, "invalid time range")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create range")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"range":   created,
			"version": version,
		})
	}
}

func newGetRangeHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		rng, err := service.GetRange(r.Context(), id)
		if err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, "range not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to get range")
			return
		}

		writeJSON(w, http.StatusOK, rng)
	}
}

func newUpdateRangeHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		var rng flag.FlagRange
		if err := json.NewDecoder(r.Body).Decode(&rng); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		rng.ID = id
		performedBy := requestUser(r)

		if err := service.UpdateRange(r.Context(), rng, performedBy); err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, "range not found")
				return
			}
			if errors.Is(err, flag.ErrRangeOverlap) {
				writeError(w, http.StatusConflict, "range overlaps with existing range")
				return
			}
			if errors.Is(err, flag.ErrInvalidTimeRange) {
				writeError(w, http.StatusBadRequest, "invalid time range")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update range")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "range updated successfully",
			"id":      id,
		})
	}
}

func newDeleteRangeHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.DeleteRange(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, "range not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete range")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "range deleted successfully",
			"id":      id,
		})
	}
}

func newActivateRangeHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.ActivateRange(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, "range not found")
				return
			}
			if errors.Is(err, flag.ErrNoPublishedVersion) {
				writeError(w, http.StatusConflict, "cannot activate: no published version")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to activate range")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "range activated successfully",
			"id":      id,
		})
	}
}

func newDeactivateRangeHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.DeactivateRange(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, "range not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to deactivate range")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "range deactivated successfully",
			"id":      id,
		})
	}
}

// --- Version handlers ---

func newListVersionsHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rangeID, err := parseIDParam(r, "rangeId")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		versions, err := service.ListVersions(r.Context(), rangeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list versions")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"versions": versions,
		})
	}
}

func newCreateVersionHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rangeID, err := parseIDParam(r, "rangeId")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}

		var req struct {
			Rules []flag.Rule `json:"rules"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		performedBy := requestUser(r)

		version, err := service.CreateVersion(r.Context(), rangeID, req.Rules, performedBy)
		if err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, "range not found")
				return
			}
			if errors.Is(err, flag.ErrDraftExists) {
				writeError(w, http.StatusConflict, "a draft version already exists")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create version")
			return
		}

		writeJSON(w, http.StatusCreated, version)
	}
}

func newGetVersionHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}

		version, err := service.GetVersion(r.Context(), id)
		if err != nil {
			if errors.Is(err, flag.ErrVersionNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to get version")
			return
		}

		writeJSON(w, http.StatusOK, version)
	}
}

func newUpdateVersionHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}

		var v flag.RangeVersion
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		v.ID = id
		performedBy := requestUser(r)

		if err := service.UpdateVersion(r.Context(), v, performedBy); err != nil {
			if errors.Is(err, flag.ErrVersionNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			if errors.Is(err, flag.ErrVersionNotDraft) {
				writeError(w, http.StatusConflict, "only draft versions can be updated")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update version")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "version updated successfully",
			"id":      id,
		})
	}
}

func newDeleteVersionHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.DeleteDraftVersion(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrVersionNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			if errors.Is(err, flag.ErrCannotDeletePublished) {
				writeError(w, http.StatusConflict, "cannot delete a published version")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete version")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "version deleted successfully",
			"id":      id,
		})
	}
}

func newPublishVersionHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}

		performedBy := requestUser(r)

		if err := service.PublishVersion(r.Context(), id, performedBy); err != nil {
			if errors.Is(err, flag.ErrVersionNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			if errors.Is(err, flag.ErrVersionNotDraft) {
				writeError(w, http.StatusConflict, "only draft versions can be published")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to publish version")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"message": "version published successfully",
			"id":      id,
		})
	}
}

func newRollbackVersionHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseIDParam(r, "id")
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}

		performedBy := requestUser(r)

		version, err := service.RollbackVersion(r.Context(), id, performedBy)
		if err != nil {
			if errors.Is(err, flag.ErrVersionNotFound) {
				writeError(w, http.StatusNotFound, "version not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to rollback version")
			return
		}

		writeJSON(w, http.StatusOK, version)
	}
}

// --- Audit handler ---

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
			writeError(w, http.StatusInternalServerError, "failed to get audit logs")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"logs": logs,
		})
	}
}

// --- Helpers ---

func requestUser(r *http.Request) string {
	if user, ok := auth.UserFromContext(r.Context()); ok && user.Username != "" {
		return user.Username
	}
	return "unknown"
}

func parseIDParam(r *http.Request, name string) (int64, error) {
	s := r.PathValue(name)
	if s == "" {
		return 0, errors.New("missing path parameter")
	}
	return strconv.ParseInt(s, 10, 64)
}
