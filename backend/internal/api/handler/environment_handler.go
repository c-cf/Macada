package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/go-chi/chi/v5"
)

type EnvironmentHandler struct {
	repo domain.EnvironmentRepository
}

func NewEnvironmentHandler(repo domain.EnvironmentRepository) *EnvironmentHandler {
	return &EnvironmentHandler{repo: repo}
}

type createEnvironmentRequest struct {
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Config      *json.RawMessage  `json:"config,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (h *EnvironmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEnvironmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now().UTC()
	env := &domain.Environment{
		ID:          domain.NewEnvironmentID(),
		WorkspaceID: workspaceIDFromCtx(r),
		Name:        req.Name,
		Description: "",
		Metadata:    map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Type:        "environment",
	}
	if req.Description != nil {
		env.Description = *req.Description
	}
	if req.Metadata != nil {
		env.Metadata = req.Metadata
	}
	if req.Config != nil {
		if err := json.Unmarshal(*req.Config, &env.Config); err != nil {
			writeError(w, http.StatusBadRequest, "invalid config")
			return
		}
	}
	// Default config if not provided
	if env.Config.Type == "" {
		env.Config.Type = "cloud"
	}

	if err := h.repo.Create(r.Context(), env); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create environment")
		return
	}

	writeJSON(w, http.StatusOK, env)
}

func (h *EnvironmentHandler) List(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)
	params.WorkspaceID = workspaceIDFromCtx(r)

	envs, nextPage, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list environments")
		return
	}
	if envs == nil {
		envs = []*domain.Environment{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Environment]{
		Data:     envs,
		NextPage: nextPage,
	})
}

func (h *EnvironmentHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "environment_id")
	env, err := h.repo.GetByID(r.Context(), id)
	if err != nil || env.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (h *EnvironmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "environment_id")
	env, err := h.repo.GetByID(r.Context(), id)
	if err != nil || env.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}

	var req createEnvironmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		env.Name = req.Name
	}
	if req.Description != nil {
		env.Description = *req.Description
	}
	if req.Config != nil {
		if err := json.Unmarshal(*req.Config, &env.Config); err != nil {
			writeError(w, http.StatusBadRequest, "invalid config")
			return
		}
	}
	if req.Metadata != nil {
		env.Metadata = req.Metadata
	}

	if err := h.repo.Update(r.Context(), env); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update environment")
		return
	}

	writeJSON(w, http.StatusOK, env)
}

func (h *EnvironmentHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "environment_id")
	env, err := h.repo.GetByID(r.Context(), id)
	if err != nil || env.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	env, err = h.repo.Archive(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (h *EnvironmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "environment_id")

	// Verify it exists and belongs to the workspace
	env, err := h.repo.GetByID(r.Context(), id)
	if err != nil || env.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete environment")
		return
	}

	writeJSON(w, http.StatusOK, domain.EnvironmentDeleteResponse{
		ID:   id,
		Type: "environment_deleted",
	})
}
