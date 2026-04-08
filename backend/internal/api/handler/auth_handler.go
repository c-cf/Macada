package handler

import (
	"net/http"

	"github.com/cchu-code/managed-agents/internal/api/middleware"
	"github.com/cchu-code/managed-agents/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	name := req.Name
	if name == "" {
		name = req.Email
	}

	result, err := h.authService.Register(r.Context(), req.Email, req.Password, name)
	if err != nil {
		if err.Error() == "email already registered" {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to register")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	result, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Me returns the current authenticated user info + active workspace.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "this endpoint requires JWT authentication")
		return
	}

	wsID := middleware.WorkspaceIDFromCtx(r.Context())

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":      userID,
		"workspace_id": wsID,
	})
}

// ListWorkspaces returns the workspaces the current JWT user belongs to.
// Requires JWT auth (user_id in context).
func (h *AuthHandler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "this endpoint requires JWT authentication")
		return
	}

	workspaces, err := h.authService.ListUserWorkspaces(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": workspaces,
	})
}
