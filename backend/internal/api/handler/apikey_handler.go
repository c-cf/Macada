package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/cchu-code/managed-agents/internal/api/middleware"
	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/go-chi/chi/v5"
)

type APIKeyHandler struct {
	apiKeyRepo    domain.APIKeyRepository
	workspaceRepo domain.WorkspaceRepository
}

func NewAPIKeyHandler(apiKeyRepo domain.APIKeyRepository, workspaceRepo domain.WorkspaceRepository) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyRepo:    apiKeyRepo,
		workspaceRepo: workspaceRepo,
	}
}

type createAPIKeyRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	wsID := workspaceIDFromCtx(r)

	var req createAPIKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Generate a random plaintext key
	plaintext, err := generatePlaintextKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	keyHash := middleware.SHA256Hex(plaintext)
	now := time.Now().UTC()

	key := &domain.APIKey{
		ID:          domain.NewAPIKeyID(),
		WorkspaceID: wsID,
		Name:        req.Name,
		KeyPrefix:   plaintext[:12],
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   now,
		Type:        "api_key",
	}

	if err := h.apiKeyRepo.Create(r.Context(), key, keyHash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	writeJSON(w, http.StatusOK, domain.APIKeyCreateResult{
		APIKey: *key,
		Key:    plaintext,
	})
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	wsID := workspaceIDFromCtx(r)
	params := parseListParams(r)

	keys, nextPage, err := h.apiKeyRepo.ListByWorkspace(r.Context(), wsID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}
	if keys == nil {
		keys = []*domain.APIKey{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.APIKey]{
		Data:     keys,
		NextPage: nextPage,
	})
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	wsID := workspaceIDFromCtx(r)
	keyID := chi.URLParam(r, "key_id")

	key, err := h.apiKeyRepo.Revoke(r.Context(), keyID, wsID)
	if err != nil {
		writeError(w, http.StatusNotFound, "API key not found")
		return
	}

	writeJSON(w, http.StatusOK, key)
}

func (h *APIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	wsID := workspaceIDFromCtx(r)
	keyID := chi.URLParam(r, "key_id")

	if err := h.apiKeyRepo.Delete(r.Context(), keyID, wsID); err != nil {
		writeError(w, http.StatusNotFound, "API key not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      keyID,
		"deleted": true,
	})
}

// BootstrapHandler creates the first workspace + API key pair.
// Requires ADMIN_SECRET for authentication.
type BootstrapHandler struct {
	adminSecret   string
	workspaceRepo domain.WorkspaceRepository
	apiKeyRepo    domain.APIKeyRepository
}

func NewBootstrapHandler(adminSecret string, workspaceRepo domain.WorkspaceRepository, apiKeyRepo domain.APIKeyRepository) *BootstrapHandler {
	return &BootstrapHandler{
		adminSecret:   adminSecret,
		workspaceRepo: workspaceRepo,
		apiKeyRepo:    apiKeyRepo,
	}
}

type bootstrapRequest struct {
	WorkspaceName string `json:"workspace_name"`
	KeyName       string `json:"key_name"`
}

func (h *BootstrapHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	if h.adminSecret == "" {
		writeError(w, http.StatusForbidden, "bootstrap is disabled (ADMIN_SECRET not configured)")
		return
	}

	secret := r.Header.Get("X-Admin-Secret")
	if secret != h.adminSecret {
		writeError(w, http.StatusUnauthorized, "invalid admin secret")
		return
	}

	var req bootstrapRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.WorkspaceName == "" {
		req.WorkspaceName = "Default"
	}

	now := time.Now().UTC()

	// Create workspace
	ws := &domain.Workspace{
		ID:          domain.NewWorkspaceID(),
		Name:        req.WorkspaceName,
		Description: "Auto-created via bootstrap",
		Metadata:    map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Type:        "workspace",
	}
	if err := h.workspaceRepo.Create(r.Context(), ws); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}

	// Generate API key
	plaintext, err := generatePlaintextKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate API key")
		return
	}

	keyHash := middleware.SHA256Hex(plaintext)
	keyName := req.KeyName
	if keyName == "" {
		keyName = "bootstrap-key"
	}

	key := &domain.APIKey{
		ID:          domain.NewAPIKeyID(),
		WorkspaceID: ws.ID,
		Name:        keyName,
		KeyPrefix:   plaintext[:12],
		CreatedAt:   now,
		Type:        "api_key",
	}

	if err := h.apiKeyRepo.Create(r.Context(), key, keyHash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"workspace": ws,
		"api_key": domain.APIKeyCreateResult{
			APIKey: *key,
			Key:    plaintext,
		},
	})
}

func generatePlaintextKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ma_" + hex.EncodeToString(bytes), nil
}
