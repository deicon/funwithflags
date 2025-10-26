package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deicon/funwithflags/internal/flag"
)

func TestEvaluateFlag_Success(t *testing.T) {
	repo := flag.NewInMemoryRepository()
	engine := flag.NewEngine()
	service, err := flag.NewService(repo, engine)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	flagDef := flag.FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "checkout",
		Enabled:    true,
		Active:     true,
		ValidFrom:  time.Now(),
		DefaultKey: "control",
		Variations: []flag.Variation{
			{Key: "control", Type: flag.BooleanVariation, Value: false},
			{Key: "variant", Type: flag.BooleanVariation, Value: true},
		},
		Rules: []flag.Rule{
			{
				ID:           "country-de",
				VariationKey: "variant",
				Conditions: []flag.Condition{
					{Attribute: "country", Operator: flag.MatcherEquals, Value: "DE"},
				},
			},
		},
	}
	if err := repo.UpsertFlag(context.Background(), flagDef); err != nil {
		t.Fatalf("UpsertFlag: %v", err)
	}

	router, err := NewRouter(Config{FlagService: service})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	payload := `{"context":{"country":"DE"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-project/dev/flags/checkout/evaluate", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp evaluateResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.VariationKey != "variant" {
		t.Fatalf("expected variation 'variant', got %s", resp.VariationKey)
	}
	if resp.Reason != flag.ReasonTargetMatch {
		t.Fatalf("expected reason %s, got %s", flag.ReasonTargetMatch, resp.Reason)
	}
}

func TestEvaluateFlag_NotFound(t *testing.T) {
	engine := flag.NewEngine()
	repo := flag.NewInMemoryRepository()
	service, err := flag.NewService(repo, engine)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	router, err := NewRouter(Config{FlagService: service})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-project/dev/flags/missing/evaluate", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestEvaluateFlag_InvalidJSON(t *testing.T) {
	engine := flag.NewEngine()
	repo := flag.NewInMemoryRepository()
	service, err := flag.NewService(repo, engine)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	router, err := NewRouter(Config{FlagService: service})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-project/dev/flags/test/evaluate", bytes.NewBufferString(`{"context":`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestNewRouterRequiresService(t *testing.T) {
	if _, err := NewRouter(Config{}); err == nil {
		t.Fatalf("expected error when service missing")
	}
}
