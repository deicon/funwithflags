package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
)

// NewSPAHandler returns an http.Handler that serves static files from dir.
// If the requested file does not exist, it falls back to dir/index.html
// to support client-side routing.
func NewSPAHandler(dir string) http.Handler {
	return &spaHandler{dir: dir}
}

type spaHandler struct {
	dir string
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path and resolve to filesystem
	path := filepath.Join(h.dir, filepath.Clean(r.URL.Path))

	// Check if the file exists
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		// Fall back to index.html for SPA client-side routing
		http.ServeFile(w, r, filepath.Join(h.dir, "index.html"))
		return
	}

	http.ServeFile(w, r, path)
}
