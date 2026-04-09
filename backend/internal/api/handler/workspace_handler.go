package handler

import (
	"net/http"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/go-chi/chi/v5"
)

type WorkspaceHandler struct {
	repo domain.WorkspaceRepository
}

func NewWorkspaceHandler(repo domain.WorkspaceRepository) *WorkspaceHandler {
	return &WorkspaceHandler{repo: repo}
}

type createWorkspaceRequest struct {
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (h *WorkspaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	now := time.Now().UTC()
	ws := &domain.Workspace{
		ID:          domain.NewWorkspaceID(),
		Name:        req.Name,
		Description: "",
		Metadata:    map[string]string{},
		CreatedAt:   now,
		UpdatedAt:   now,
		Type:        "workspace",
	}
	if req.Description != nil {
		ws.Description = *req.Description
	}
	if req.Metadata != nil {
		ws.Metadata = req.Metadata
	}

	if err := h.repo.Create(r.Context(), ws); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}

	writeJSON(w, http.StatusOK, ws)
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)
	workspaces, nextPage, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}
	if workspaces == nil {
		workspaces = []*domain.Workspace{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Workspace]{
		Data:     workspaces,
		NextPage: nextPage,
	})
}

func (h *WorkspaceHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "workspace_id")
	wsID := workspaceIDFromCtx(r)
	if id != wsID {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}

	ws, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

func (h *WorkspaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "workspace_id")
	wsID := workspaceIDFromCtx(r)
	if id != wsID {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}

	ws, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}

	var req createWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		ws.Name = req.Name
	}
	if req.Description != nil {
		ws.Description = *req.Description
	}
	if req.Metadata != nil {
		ws.Metadata = req.Metadata
	}

	if err := h.repo.Update(r.Context(), ws); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update workspace")
		return
	}

	writeJSON(w, http.StatusOK, ws)
}
