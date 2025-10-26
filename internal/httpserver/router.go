package httpserver

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(livenessHandler))
	mux.Handle("/readyz", http.HandlerFunc(readinessHandler))
	return mux
}
