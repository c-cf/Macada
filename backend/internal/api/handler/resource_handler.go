package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/go-chi/chi/v5"
)

type ResourceHandler struct {
	resourceRepo domain.ResourceRepository
	sessionRepo  domain.SessionRepository
	fileRepo     domain.FileRepository
}

func NewResourceHandler(resourceRepo domain.ResourceRepository, sessionRepo domain.SessionRepository, fileRepo domain.FileRepository) *ResourceHandler {
	return &ResourceHandler{
		resourceRepo: resourceRepo,
		sessionRepo:  sessionRepo,
		fileRepo:     fileRepo,
	}
}

type addResourceRequest struct {
	Type      string  `json:"type"`                // "file"
	FileID    string  `json:"file_id"`             // required for type "file"
	MountPath *string `json:"mount_path,omitempty"` // optional
}

// Add handles POST /v1/sessions/{session_id}/resources.
func (h *ResourceHandler) Add(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	wsID := workspaceIDFromCtx(r)

	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	var req addResourceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type != "file" {
		writeError(w, http.StatusBadRequest, "only 'file' resource type is currently supported")
		return
	}

	if req.FileID == "" {
		writeError(w, http.StatusBadRequest, "file_id is required for file resources")
		return
	}

	// Validate file exists and belongs to same workspace
	file, err := h.fileRepo.GetByID(r.Context(), req.FileID)
	if err != nil || file.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	mountPath := fmt.Sprintf("/mnt/session/uploads/%s", req.FileID)
	if req.MountPath != nil && *req.MountPath != "" {
		if err := validateMountPath(*req.MountPath); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		mountPath = *req.MountPath
	}

	now := time.Now().UTC()
	resource := &domain.SessionResource{
		ID:        domain.NewResourceID(),
		SessionID: sessionID,
		Type:      "file",
		FileID:    &req.FileID,
		MountPath: mountPath,
		Config:    json.RawMessage("{}"),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.resourceRepo.Create(r.Context(), resource); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create resource")
		return
	}

	writeJSON(w, http.StatusOK, resource)
}

// List handles GET /v1/sessions/{session_id}/resources.
func (h *ResourceHandler) List(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	wsID := workspaceIDFromCtx(r)

	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	lp := parseListParams(r)
	resources, nextPage, err := h.resourceRepo.ListBySession(r.Context(), sessionID, lp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list resources")
		return
	}
	if resources == nil {
		resources = []*domain.SessionResource{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.SessionResource]{
		Data:     resources,
		NextPage: nextPage,
	})
}

// Retrieve handles GET /v1/sessions/{session_id}/resources/{resource_id}.
func (h *ResourceHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	resourceID := chi.URLParam(r, "resource_id")
	wsID := workspaceIDFromCtx(r)

	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	resource, err := h.resourceRepo.GetByID(r.Context(), resourceID)
	if err != nil || resource.SessionID != sessionID {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}

	writeJSON(w, http.StatusOK, resource)
}

type updateResourceRequest struct {
	AuthorizationToken *string `json:"authorization_token,omitempty"`
	MountPath          *string `json:"mount_path,omitempty"`
}

// Update handles POST /v1/sessions/{session_id}/resources/{resource_id}.
func (h *ResourceHandler) Update(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	resourceID := chi.URLParam(r, "resource_id")
	wsID := workspaceIDFromCtx(r)

	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	resource, err := h.resourceRepo.GetByID(r.Context(), resourceID)
	if err != nil || resource.SessionID != sessionID {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}

	var req updateResourceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.MountPath != nil && *req.MountPath != "" {
		if err := validateMountPath(*req.MountPath); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		resource.MountPath = *req.MountPath
	}

	// For github_repository type, store authorization_token in config
	if req.AuthorizationToken != nil && resource.Type == "github_repository" {
		var cfg map[string]interface{}
		if err := json.Unmarshal(resource.Config, &cfg); err != nil {
			cfg = map[string]interface{}{}
		}
		cfg["authorization_token"] = *req.AuthorizationToken
		resource.Config, _ = json.Marshal(cfg)
	}

	if err := h.resourceRepo.Update(r.Context(), resource); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update resource")
		return
	}

	writeJSON(w, http.StatusOK, resource)
}

// Delete handles DELETE /v1/sessions/{session_id}/resources/{resource_id}.
func (h *ResourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")
	resourceID := chi.URLParam(r, "resource_id")
	wsID := workspaceIDFromCtx(r)

	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	resource, err := h.resourceRepo.GetByID(r.Context(), resourceID)
	if err != nil || resource.SessionID != sessionID {
		writeError(w, http.StatusNotFound, "resource not found")
		return
	}

	if err := h.resourceRepo.Delete(r.Context(), resourceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete resource")
		return
	}

	writeJSON(w, http.StatusOK, domain.DeleteResourceResponse{
		ID:   resourceID,
		Type: "session_resource_deleted",
	})
}

func validateMountPath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("mount_path must not contain '..'")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("mount_path must be an absolute path")
	}
	return nil
}
