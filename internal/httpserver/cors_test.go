package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCORSHandler_AllowsConfiguredOrigin(t *testing.T) {
	var called bool
	h := newCORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}), []string{"https://example.com/"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if !called {
		t.Fatalf("expected downstream handler to be called")
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func TestNewCORSHandler_HandlesPreflight(t *testing.T) {
	h := newCORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("preflight request should not reach next handler")
	}), []string{"https://example.com"})

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, res.Code)
	}
}

func TestNewCORSHandler_IgnoresUnknownOrigin(t *testing.T) {
	var called bool
	h := newCORSHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}), []string{"https://allowed.example"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://denied.example")
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)

	if !called {
		t.Fatalf("expected downstream handler to be called for unknown origin")
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow origin header, got %q", got)
	}
}
