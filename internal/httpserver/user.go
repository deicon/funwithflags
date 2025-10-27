package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/deicon/funwithflags/internal/auth"
)

type userCreateRequest struct {
	Username string    `json:"username"`
	Password string    `json:"password"`
	Role     auth.Role `json:"role"`
}

type userUpdateRequest struct {
	Password *string    `json:"password"`
	Role     *auth.Role `json:"role"`
}

func newListUsersHandler(service *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := service.ListUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list users: %v", err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users})
	}
}

func newCreateUserHandler(service *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req userCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		user, err := service.CreateUser(r.Context(), req.Username, req.Password, req.Role)
		if err != nil {
			switch err {
			case auth.ErrUserExists:
				writeError(w, http.StatusConflict, "user already exists")
			default:
				writeError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"user": user})
	}
}

func newUpdateUserHandler(service *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			writeError(w, http.StatusBadRequest, "username is required")
			return
		}

		var req userUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}

		if req.Password != nil && *req.Password != "" {
			if err := service.UpdatePassword(r.Context(), username, *req.Password); err != nil {
				if err == auth.ErrUserNotFound {
					writeError(w, http.StatusNotFound, "user not found")
					return
				}
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		if req.Role != nil {
			if err := service.UpdateRole(r.Context(), username, *req.Role); err != nil {
				if err == auth.ErrUserNotFound {
					writeError(w, http.StatusNotFound, "user not found")
					return
				}
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message":  "user updated",
			"username": username,
		})
	}
}

func newDeleteUserHandler(service *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.PathValue("username")
		if username == "" {
			writeError(w, http.StatusBadRequest, "username is required")
			return
		}

		if err := service.DeleteUser(r.Context(), username); err != nil {
			if err == auth.ErrUserNotFound {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete user: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message":  "user deleted",
			"username": username,
		})
	}
}
