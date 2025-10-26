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
	mux.Handle("/healthz", http.HandlerFunc(livenessHandler))
	mux.Handle("/readyz", http.HandlerFunc(readinessHandler))
	mux.HandleFunc("POST /api/v1/flags/{key}/evaluate", newEvaluateHandler(cfg.FlagService))

	return mux, nil
}
