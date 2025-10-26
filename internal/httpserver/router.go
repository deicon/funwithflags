package httpserver

import (
	"fmt"
	"net/http"

	"github.com/deicon/funwithflags/internal/flag"
)

type Config struct {
	FlagService *flag.Service
}

func NewRouter(cfg Config) (*http.ServeMux, error) {
	if cfg.FlagService == nil {
		return nil, fmt.Errorf("flag service is required")
	}

	mux := http.NewServeMux()

	// Health endpoints
	mux.Handle("/healthz", http.HandlerFunc(livenessHandler))
	mux.Handle("/readyz", http.HandlerFunc(readinessHandler))

	// Evaluation endpoint
	mux.HandleFunc("POST /api/v1/{project}/{stage}/flags/{key}/evaluate", newEvaluateHandler(cfg.FlagService))

	// Admin endpoints
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags", newListFlagsHandler(cfg.FlagService))
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}", newGetFlagHandler(cfg.FlagService))
	mux.HandleFunc("POST /api/v1/admin/{project}/{stage}/flags", newCreateFlagHandler(cfg.FlagService))
	mux.HandleFunc("PUT /api/v1/admin/{project}/{stage}/flags/{key}", newUpdateFlagHandler(cfg.FlagService))
	mux.HandleFunc("DELETE /api/v1/admin/{project}/{stage}/flags/{key}", newDeleteFlagHandler(cfg.FlagService))
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}/audit", newGetAuditLogsHandler(cfg.FlagService))

	return mux, nil
}
