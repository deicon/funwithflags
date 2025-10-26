package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()

	cases := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{name: "health", path: "/healthz", expectedStatus: http.StatusOK},
		{name: "ready", path: "/readyz", expectedStatus: http.StatusOK},
		{name: "not found", path: "/does-not-exist", expectedStatus: http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}
