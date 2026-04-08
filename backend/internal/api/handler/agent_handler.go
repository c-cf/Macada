package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/go-chi/chi/v5"
)

type AgentHandler struct {
	repo domain.AgentRepository
}

func NewAgentHandler(repo domain.AgentRepository) *AgentHandler {
	return &AgentHandler{repo: repo}
}

type createAgentRequest struct {
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Model       json.RawMessage   `json:"model"`
	System      *string           `json:"system,omitempty"`
	Tools       json.RawMessage   `json:"tools,omitempty"`
	MCPServers  json.RawMessage   `json:"mcp_servers,omitempty"`
	Skills      json.RawMessage   `json:"skills,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Parse model - can be a string or an object
	model, err := parseModelConfig(req.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid model")
		return
	}

	now := time.Now().UTC()
	agent := &domain.Agent{
		ID:          domain.NewAgentID(),
		WorkspaceID: workspaceIDFromCtx(r),
		Name:        req.Name,
		Description: "",
		Model:       model,
		System:      "",
		Tools:       defaultJSON(req.Tools, "[]"),
		MCPServers:  defaultJSON(req.MCPServers, "[]"),
		Skills:      defaultJSON(req.Skills, "[]"),
		Metadata:    map[string]string{},
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		Type:        "agent",
	}
	if req.Description != nil {
		agent.Description = *req.Description
	}
	if req.System != nil {
		agent.System = *req.System
	}
	if req.Metadata != nil {
		agent.Metadata = req.Metadata
	}

	if err := h.repo.Create(r.Context(), agent); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	lp := parseListParams(r)
	lp.WorkspaceID = workspaceIDFromCtx(r)
	params := domain.AgentListParams{
		ListParams: lp,
	}
	if v := r.URL.Query().Get("created_at[gte]"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			params.CreatedAtGTE = &t
		}
	}
	if v := r.URL.Query().Get("created_at[lte]"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			params.CreatedAtLTE = &t
		}
	}

	agents, nextPage, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	if agents == nil {
		agents = []*domain.Agent{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Agent]{
		Data:     agents,
		NextPage: nextPage,
	})
}

func (h *AgentHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "agent_id")
	agent, err := h.repo.GetByID(r.Context(), id)
	if err != nil || agent.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "agent_id")
	agent, err := h.repo.GetByID(r.Context(), id)
	if err != nil || agent.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	var req createAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != nil {
		agent.Description = *req.Description
	}
	if req.Model != nil {
		model, err := parseModelConfig(req.Model)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid model")
			return
		}
		agent.Model = model
	}
	if req.System != nil {
		agent.System = *req.System
	}
	if req.Tools != nil {
		agent.Tools = req.Tools
	}
	if req.MCPServers != nil {
		agent.MCPServers = req.MCPServers
	}
	if req.Skills != nil {
		agent.Skills = req.Skills
	}
	if req.Metadata != nil {
		agent.Metadata = req.Metadata
	}

	agent.Version++

	if err := h.repo.Update(r.Context(), agent); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}

	writeJSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "agent_id")
	// Verify workspace ownership before archiving
	agent, err := h.repo.GetByID(r.Context(), id)
	if err != nil || agent.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	agent, err = h.repo.Archive(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func parseModelConfig(raw json.RawMessage) (domain.ModelConfig, error) {
	if raw == nil {
		return domain.ModelConfig{}, nil
	}

	// Try string first (e.g., "claude-sonnet-4-6")
	var modelStr string
	if err := json.Unmarshal(raw, &modelStr); err == nil {
		return domain.ModelConfig{ID: modelStr}, nil
	}

	// Try object (e.g., {"id": "claude-sonnet-4-6", "speed": "fast"})
	var model domain.ModelConfig
	if err := json.Unmarshal(raw, &model); err != nil {
		return domain.ModelConfig{}, err
	}
	return model, nil
}

func defaultJSON(raw json.RawMessage, fallback string) json.RawMessage {
	if raw == nil || len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}
