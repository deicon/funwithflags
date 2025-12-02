package httpserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/deicon/funwithflags/internal/auth"
	"github.com/deicon/funwithflags/internal/flag"
	"github.com/deicon/funwithflags/internal/project"
)

type Config struct {
	FlagService    *flag.Service
	ProjectService *project.Service
	AuthManager    *auth.Manager
	AuthService    *auth.Service
	AllowedOrigins []string
}

func NewRouter(cfg Config) (http.Handler, error) {
	if cfg.FlagService == nil {
		return nil, fmt.Errorf("flag service is required")
	}
	if cfg.ProjectService == nil {
		return nil, fmt.Errorf("project service is required")
	}
	if cfg.AuthManager == nil {
		return nil, fmt.Errorf("auth manager is required")
	}
	if cfg.AuthService == nil {
		return nil, fmt.Errorf("auth service is required")
	}

	mux := http.NewServeMux()

	// Authentication endpoints
	mux.HandleFunc("POST /api/v1/auth/login", newLoginHandler(cfg.AuthManager))
	mux.HandleFunc("POST /api/v1/auth/refresh", newRefreshHandler(cfg.AuthManager))
	mux.HandleFunc("POST /api/v1/auth/logout", newLogoutHandler(cfg.AuthManager))

// Health endpoints (unauthenticated)
mux.HandleFunc("GET /healthz", livenessHandler)
mux.HandleFunc("GET /readyz", readinessHandler)

	// Evaluation endpoint
	mux.HandleFunc("POST /api/v1/{project}/{stage}/flags/{key}/evaluate",
		wrapAuth(cfg.AuthManager, false, newEvaluateHandler(cfg.FlagService)))

	// Project endpoints
	mux.HandleFunc("GET /api/v1/admin/projects",
		wrapAuth(cfg.AuthManager, true, newListProjectsHandler(cfg.ProjectService)))
	mux.HandleFunc("POST /api/v1/admin/projects",
		wrapAuth(cfg.AuthManager, true, newCreateProjectHandler(cfg.ProjectService)))
	mux.HandleFunc("GET /api/v1/admin/projects/{project}",
		wrapAuth(cfg.AuthManager, true, newGetProjectHandler(cfg.ProjectService)))
	mux.HandleFunc("PUT /api/v1/admin/projects/{project}",
		wrapAuth(cfg.AuthManager, true, newUpdateProjectHandler(cfg.ProjectService)))
	mux.HandleFunc("DELETE /api/v1/admin/projects/{project}",
		wrapAuth(cfg.AuthManager, true, newDeleteProjectHandler(cfg.ProjectService)))

	// Stage endpoints
	mux.HandleFunc("GET /api/v1/admin/projects/{project}/stages",
		wrapAuth(cfg.AuthManager, true, newListStagesHandler(cfg.ProjectService)))
	mux.HandleFunc("POST /api/v1/admin/projects/{project}/stages",
		wrapAuth(cfg.AuthManager, true, newCreateStageHandler(cfg.ProjectService)))
	mux.HandleFunc("GET /api/v1/admin/projects/{project}/stages/{stage}",
		wrapAuth(cfg.AuthManager, true, newGetStageHandler(cfg.ProjectService)))
	mux.HandleFunc("PUT /api/v1/admin/projects/{project}/stages/{stage}",
		wrapAuth(cfg.AuthManager, true, newUpdateStageHandler(cfg.ProjectService)))
	mux.HandleFunc("DELETE /api/v1/admin/projects/{project}/stages/{stage}",
		wrapAuth(cfg.AuthManager, true, newDeleteStageHandler(cfg.ProjectService)))

	// Admin endpoints - flag operations (currently active flags)
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags",
		wrapAuth(cfg.AuthManager, true, newListFlagsHandler(cfg.FlagService)))
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}",
		wrapAuth(cfg.AuthManager, true, newGetFlagHandler(cfg.FlagService)))
	mux.HandleFunc("POST /api/v1/admin/{project}/{stage}/flags",
		wrapAuth(cfg.AuthManager, true, newCreateFlagHandler(cfg.FlagService)))
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}/audit",
		wrapAuth(cfg.AuthManager, true, newGetAuditLogsHandler(cfg.FlagService)))

	// Admin endpoints - temporal range operations
	mux.HandleFunc("GET /api/v1/admin/flags/{id}",
		wrapAuth(cfg.AuthManager, true, newGetFlagByIDHandler(cfg.FlagService)))
	mux.HandleFunc("PUT /api/v1/admin/flags/{id}",
		wrapAuth(cfg.AuthManager, true, newUpdateFlagHandler(cfg.FlagService)))
	mux.HandleFunc("DELETE /api/v1/admin/flags/{id}",
		wrapAuth(cfg.AuthManager, true, newDeleteFlagHandler(cfg.FlagService)))
	mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}/ranges",
		wrapAuth(cfg.AuthManager, true, newGetFlagRangesHandler(cfg.FlagService)))
	mux.HandleFunc("POST /api/v1/admin/flags/{id}/activate",
		wrapAuth(cfg.AuthManager, true, newActivateFlagHandler(cfg.FlagService)))
	mux.HandleFunc("POST /api/v1/admin/flags/{id}/deactivate",
		wrapAuth(cfg.AuthManager, true, newDeactivateFlagHandler(cfg.FlagService)))
	// User management endpoints
	mux.HandleFunc("GET /api/v1/admin/users",
		wrapAuth(cfg.AuthManager, true, newListUsersHandler(cfg.AuthService)))
	mux.HandleFunc("POST /api/v1/admin/users",
		wrapAuth(cfg.AuthManager, true, newCreateUserHandler(cfg.AuthService)))
	mux.HandleFunc("PUT /api/v1/admin/users/{username}",
		wrapAuth(cfg.AuthManager, true, newUpdateUserHandler(cfg.AuthService)))
	mux.HandleFunc("DELETE /api/v1/admin/users/{username}",
		wrapAuth(cfg.AuthManager, true, newDeleteUserHandler(cfg.AuthService)))

	handler := http.Handler(mux)
	if len(cfg.AllowedOrigins) > 0 {
		handler = newCORSHandler(handler, cfg.AllowedOrigins)
	}

	return handler, nil
}

func wrapAuth(manager *auth.Manager, requireAdmin bool, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}

		user, err := manager.ParseAccessToken(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if requireAdmin && user.Role != auth.RoleAdmin {
			writeError(w, http.StatusForbidden, "admin privileges required")
			return
		}

		handler(w, r.WithContext(auth.WithUser(r.Context(), user)))
	}
}

func extractBearerToken(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("authorization header required")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("authorization token missing")
	}
	return token, nil
}
