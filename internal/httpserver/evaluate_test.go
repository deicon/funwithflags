package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deicon/funwithflags/internal/auth"
	"github.com/deicon/funwithflags/internal/flag"
	"github.com/deicon/funwithflags/internal/project"
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

	router, manager := newTestRouter(t, service)

	payload := `{"context":{"country":"DE"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-project/dev/flags/checkout/evaluate", bytes.NewBufferString(payload))
	addAuthHeader(t, manager, req)
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

	router, manager := newTestRouter(t, service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-project/dev/flags/missing/evaluate", bytes.NewBufferString(`{}`))
	addAuthHeader(t, manager, req)
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

	router, manager := newTestRouter(t, service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-project/dev/flags/test/evaluate", bytes.NewBufferString(`{"context":`))
	addAuthHeader(t, manager, req)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestNewRouterRequiresService(t *testing.T) {
	projectService, authManager := newTestDependencies(t)
	if _, err := NewRouter(Config{ProjectService: projectService, AuthManager: authManager}); err == nil {
		t.Fatalf("expected error when service missing")
	}
}

func newTestRouter(t *testing.T, flagService *flag.Service) (http.Handler, *auth.Manager) {
	t.Helper()

	projectService, manager, authService := newTestDependencies(t)

	router, err := NewRouter(Config{
		FlagService:    flagService,
		ProjectService: projectService,
		AuthManager:    manager,
		AuthService:    authService,
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	return router, manager
}

func addAuthHeader(t *testing.T, manager *auth.Manager, req *http.Request) {
	t.Helper()

	user, err := manager.Authenticate(context.Background(), "tester", "password123")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	tokens, err := manager.IssueTokens(user)
	if err != nil {
		t.Fatalf("IssueTokens: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
}

func newTestDependencies(t *testing.T) (*project.Service, *auth.Manager, *auth.Service) {
	t.Helper()

	projectRepo := project.NewInMemoryRepository()
	projectService, err := project.NewService(projectRepo)
	if err != nil {
		t.Fatalf("NewService (project): %v", err)
	}

	ctx := context.Background()
	if err := projectService.CreateProject(ctx, project.Project{Key: "test-project", Name: "Test Project"}); err != nil && !errors.Is(err, project.ErrProjectExists) {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := projectService.CreateStage(ctx, project.Stage{ProjectKey: "test-project", Key: "dev", Name: "Development"}); err != nil && !errors.Is(err, project.ErrStageExists) {
		t.Fatalf("CreateStage: %v", err)
	}

	authRepo := auth.NewInMemoryRepository()
	authService, err := auth.NewService(authRepo)
	if err != nil {
		t.Fatalf("AuthService: %v", err)
	}
	if err := authService.CreateUser(ctx, "tester", "password123", auth.RoleUser); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	manager, err := auth.NewManager(auth.Config{
		Secret:          "test-secret",
		Repository:      authRepo,
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	return projectService, manager, authService
}
