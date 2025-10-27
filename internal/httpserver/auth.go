package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/deicon/funwithflags/internal/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authResponse struct {
	TokenType             string    `json:"tokenType"`
	AccessToken           string    `json:"accessToken"`
	AccessTokenExpiresAt  string    `json:"accessTokenExpiresAt"`
	RefreshToken          string    `json:"refreshToken"`
	RefreshTokenExpiresAt string    `json:"refreshTokenExpiresAt"`
	User                  auth.User `json:"user"`
}

func newLoginHandler(manager *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username and password are required")
			return
		}

		user, err := manager.Authenticate(r.Context(), req.Username, req.Password)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}

		tokens, err := manager.IssueTokens(user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to issue token: %v", err))
			return
		}

		resp := authResponse{
			TokenType:             "Bearer",
			AccessToken:           tokens.AccessToken,
			AccessTokenExpiresAt:  tokens.AccessTokenExpiresAt.UTC().Format(time.RFC3339),
			RefreshToken:          tokens.RefreshToken,
			RefreshTokenExpiresAt: tokens.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
			User:                  user,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func newRefreshHandler(manager *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		if req.RefreshToken == "" {
			writeError(w, http.StatusBadRequest, "refresh token is required")
			return
		}

		tokens, user, err := manager.Refresh(r.Context(), req.RefreshToken)
		if err != nil {
			status := http.StatusUnauthorized
			if err == auth.ErrRefreshTokenExpired {
				status = http.StatusUnauthorized
			}
			writeError(w, status, err.Error())
			return
		}

		resp := authResponse{
			TokenType:             "Bearer",
			AccessToken:           tokens.AccessToken,
			AccessTokenExpiresAt:  tokens.AccessTokenExpiresAt.UTC().Format(time.RFC3339),
			RefreshToken:          tokens.RefreshToken,
			RefreshTokenExpiresAt: tokens.RefreshTokenExpiresAt.UTC().Format(time.RFC3339),
			User:                  user,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func newLogoutHandler(manager *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req logoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
		if req.RefreshToken == "" {
			writeError(w, http.StatusBadRequest, "refresh token is required")
			return
		}

		manager.RevokeRefreshToken(req.RefreshToken)
		writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
	}
}
