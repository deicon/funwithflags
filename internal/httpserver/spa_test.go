package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSPAHandler(t *testing.T) {
	// Set up a temporary directory with test files
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('app')"), 0644); err != nil {
		t.Fatal(err)
	}

	handler := NewSPAHandler(dir)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "serves index.html at root",
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "<html>spa</html>",
		},
		{
			name:       "serves static file",
			path:       "/assets/app.js",
			wantStatus: http.StatusOK,
			wantBody:   "console.log('app')",
		},
		{
			name:       "falls back to index.html for unknown paths",
			path:       "/some/client/route",
			wantStatus: http.StatusOK,
			wantBody:   "<html>spa</html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if body := rec.Body.String(); body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}
