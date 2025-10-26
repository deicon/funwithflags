package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/deicon/funwithflags/internal/flag"
)

func newListFlagsHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flags, err := service.ListFlags(r.Context())
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
		key := r.PathValue("key")
		if key == "" {
			writeError(w, http.StatusBadRequest, "flag key is required")
			return
		}

		f, err := service.GetFlag(r.Context(), key)
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
		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		performedBy := r.Header.Get("X-User-ID")
		if performedBy == "" {
			performedBy = "unknown"
		}

		if err := service.CreateFlag(r.Context(), f, performedBy); err != nil {
			if errors.Is(err, flag.ErrInvalidFlag) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid flag: %v", err))
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
		key := r.PathValue("key")
		if key == "" {
			writeError(w, http.StatusBadRequest, "flag key is required")
			return
		}

		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		// Ensure the key matches
		f.Key = key

		performedBy := r.Header.Get("X-User-ID")
		if performedBy == "" {
			performedBy = "unknown"
		}

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
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "flag updated successfully",
			"key":     key,
		})
	}
}

func newDeleteFlagHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		if key == "" {
			writeError(w, http.StatusBadRequest, "flag key is required")
			return
		}

		performedBy := r.Header.Get("X-User-ID")
		if performedBy == "" {
			performedBy = "unknown"
		}

		if err := service.DeleteFlag(r.Context(), key, performedBy); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, "flag not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete flag: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "flag deleted successfully",
			"key":     key,
		})
	}
}

func newGetAuditLogsHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		logs, err := service.GetAuditLogs(r.Context(), key, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get audit logs: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"logs": logs,
		})
	}
}
