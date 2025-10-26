package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/deicon/funwithflags/internal/flag"
)

type evaluateRequest struct {
	Context map[string]any `json:"context"`
}

type evaluateResponse struct {
	FlagKey       string `json:"flagKey"`
	VariationKey  string `json:"variationKey"`
	VariationType string `json:"variationType"`
	Value         any    `json:"value"`
	Reason        string `json:"reason"`
}

func newEvaluateHandler(service *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeError(w, http.StatusInternalServerError, "evaluation service not configured")
			return
		}

		project := r.PathValue("project")
		if project == "" {
			writeError(w, http.StatusBadRequest, "project missing from path")
			return
		}

		stage := r.PathValue("stage")
		if stage == "" {
			writeError(w, http.StatusBadRequest, "stage missing from path")
			return
		}

		flagKey := r.PathValue("key")
		if flagKey == "" {
			writeError(w, http.StatusBadRequest, "flag key missing from path")
			return
		}

		var req evaluateRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				req.Context = map[string]any{}
			} else {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
		}

		ctx := flag.EvaluationContext(req.Context)
		result, err := service.EvaluateFlag(r.Context(), project, stage, flagKey, ctx)
		if err != nil {
			switch {
			case errors.Is(err, flag.ErrFlagNotFound):
				writeError(w, http.StatusNotFound, "flag not found")
			default:
				writeError(w, http.StatusInternalServerError, "evaluation failed")
			}
			return
		}

		resp := evaluateResponse{
			FlagKey:       flagKey,
			VariationKey:  result.Variation.Key,
			VariationType: string(result.Variation.Type),
			Value:         result.Variation.Value,
			Reason:        result.Reason,
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
