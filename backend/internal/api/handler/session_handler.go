package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/go-chi/chi/v5"
)

type SessionHandler struct {
	sessionRepo domain.SessionRepository
	agentRepo   domain.AgentRepository
	envRepo     domain.EnvironmentRepository
}

func NewSessionHandler(sessionRepo domain.SessionRepository, agentRepo domain.AgentRepository, envRepo domain.EnvironmentRepository) *SessionHandler {
	return &SessionHandler{
		sessionRepo: sessionRepo,
		agentRepo:   agentRepo,
		envRepo:     envRepo,
	}
}

type createSessionRequest struct {
	Agent         json.RawMessage   `json:"agent"`
	EnvironmentID string            `json:"environment_id"`
	Title         *string           `json:"title,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Resources     json.RawMessage   `json:"resources,omitempty"`
	VaultIDs      []string          `json:"vault_ids,omitempty"`
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.EnvironmentID == "" {
		writeError(w, http.StatusBadRequest, "environment_id is required")
		return
	}

	// Resolve agent - can be a string ID or an object {id, type, version}
	agentID, err := resolveAgentID(req.Agent)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid agent parameter")
		return
	}

	wsID := workspaceIDFromCtx(r)

	agent, err := h.agentRepo.GetByID(r.Context(), agentID)
	if err != nil || agent.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	// Verify environment belongs to the same workspace
	env, err := h.envRepo.GetByID(r.Context(), req.EnvironmentID)
	if err != nil || env.WorkspaceID != wsID {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}

	// Create agent snapshot
	agentSnapshot, err := json.Marshal(map[string]interface{}{
		"id":          agent.ID,
		"version":     agent.Version,
		"type":        "agent",
		"name":        agent.Name,
		"description": agent.Description,
		"model":       agent.Model,
		"system":      agent.System,
		"tools":       json.RawMessage(agent.Tools),
		"mcp_servers": json.RawMessage(agent.MCPServers),
		"skills":      json.RawMessage(agent.Skills),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create agent snapshot")
		return
	}

	now := time.Now().UTC()
	session := &domain.Session{
		ID:            domain.NewSessionID(),
		WorkspaceID:   wsID,
		Agent:         agentSnapshot,
		EnvironmentID: req.EnvironmentID,
		Title:         "",
		Status:        domain.SessionStatusIdle,
		Stats:         domain.SessionStats{},
		Usage:         domain.SessionUsage{},
		Resources:     defaultJSON(req.Resources, "[]"),
		Metadata:      map[string]string{},
		VaultIDs:      []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
		Type:          "session",
	}
	if req.Title != nil {
		session.Title = *req.Title
	}
	if req.Metadata != nil {
		session.Metadata = req.Metadata
	}
	if req.VaultIDs != nil {
		session.VaultIDs = req.VaultIDs
	}

	if err := h.sessionRepo.Create(r.Context(), session); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	lp := parseListParams(r)
	lp.WorkspaceID = workspaceIDFromCtx(r)
	params := domain.SessionListParams{
		ListParams: lp,
	}
	if v := r.URL.Query().Get("agent_id"); v != "" {
		params.AgentID = &v
	}
	if v := r.URL.Query().Get("order"); v != "" {
		params.Order = &v
	}
	if v := r.URL.Query().Get("created_at[gt]"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			params.CreatedAtGT = &t
		}
	}
	if v := r.URL.Query().Get("created_at[gte]"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			params.CreatedAtGTE = &t
		}
	}
	if v := r.URL.Query().Get("created_at[lt]"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			params.CreatedAtLT = &t
		}
	}
	if v := r.URL.Query().Get("created_at[lte]"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			params.CreatedAtLTE = &t
		}
	}

	sessions, nextPage, err := h.sessionRepo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}
	if sessions == nil {
		sessions = []*domain.Session{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Session]{
		Data:     sessions,
		NextPage: nextPage,
	})
}

func (h *SessionHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "session_id")
	session, err := h.sessionRepo.GetByID(r.Context(), id)
	if err != nil || session.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (h *SessionHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "session_id")
	session, err := h.sessionRepo.GetByID(r.Context(), id)
	if err != nil || session.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	session, err = h.sessionRepo.Archive(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func resolveAgentID(raw json.RawMessage) (string, error) {
	if raw == nil {
		return "", nil
	}

	// Try string first
	var agentID string
	if err := json.Unmarshal(raw, &agentID); err == nil {
		return agentID, nil
	}

	// Try object
	var agentRef struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &agentRef); err != nil {
		return "", err
	}
	return agentRef.ID, nil
}
