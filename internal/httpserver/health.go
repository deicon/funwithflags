package httpserver

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, healthResponse{Status: "ok"})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, healthResponse{Status: "ready"})
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
