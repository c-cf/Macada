package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/internal/infra/postgres"
	"github.com/cchu-code/managed-agents/internal/sandbox"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// InternalHandler handles events arriving from sandbox runtimes.
type InternalHandler struct {
	eventRepo     domain.EventRepository
	sessionRepo   domain.SessionRepository
	eventBus      domain.EventBus
	analyticsRepo *postgres.AnalyticsRepo
	tokenGen      *sandbox.TokenGenerator
}

// NewInternalHandler creates a new internal event handler.
func NewInternalHandler(
	eventRepo domain.EventRepository,
	sessionRepo domain.SessionRepository,
	eventBus domain.EventBus,
	analyticsRepo *postgres.AnalyticsRepo,
	tokenGen *sandbox.TokenGenerator,
) *InternalHandler {
	return &InternalHandler{
		eventRepo:     eventRepo,
		sessionRepo:   sessionRepo,
		eventBus:      eventBus,
		analyticsRepo: analyticsRepo,
		tokenGen:      tokenGen,
	}
}

type internalEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type internalEventsRequest struct {
	Events []internalEvent `json:"events"`
}

// IngestEvents receives events from a sandbox runtime.
func (h *InternalHandler) IngestEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")

	// Validate auth token
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || !h.tokenGen.Validate(sessionID, token) {
		writeError(w, http.StatusUnauthorized, "invalid sandbox token")
		return
	}

	var req internalEventsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for _, ie := range req.Events {
		payload := ie.Payload
		if payload == nil {
			payload = json.RawMessage("{}")
		}

		evt := &domain.Event{
			ID:          domain.NewEventID(),
			SessionID:   sessionID,
			Type:        ie.Type,
			ProcessedAt: time.Now().UTC(),
			Payload:     payload,
		}

		if err := h.eventRepo.Create(r.Context(), evt); err != nil {
			log.Error().Err(err).Str("type", ie.Type).Msg("failed to persist sandbox event")
			continue
		}

		h.eventBus.Publish(r.Context(), sessionID, evt)

		// Handle session status transitions
		switch ie.Type {
		case domain.EventTypeSessionIdle:
			h.sessionRepo.UpdateStatus(r.Context(), sessionID, domain.SessionStatusIdle)

		case domain.EventTypeSessionRunning:
			h.sessionRepo.UpdateStatus(r.Context(), sessionID, domain.SessionStatusRunning)

		case "span.model_request_end":
			h.recordAnalytics(r.Context(), sessionID, ie.Payload)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *InternalHandler) recordAnalytics(ctx context.Context, sessionID string, payload json.RawMessage) {
	var data struct {
		ModelUsage domain.ModelUsage `json:"model_usage"`
		IsError    bool              `json:"is_error"`
	}
	if json.Unmarshal(payload, &data) != nil || data.IsError {
		return
	}

	// Extract agent ID from session
	session, err := h.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return
	}
	var snap struct {
		ID string `json:"id"`
	}
	json.Unmarshal(session.Agent, &snap)

	now := time.Now().UTC()
	logEntry := postgres.LogRow{
		ID:                  domain.NewLLMLogID(),
		WorkspaceID:         session.WorkspaceID,
		SessionID:           sessionID,
		AgentID:             snap.ID,
		Model:               "claude-sonnet-4-6", // TODO: extract from payload
		InputTokens:         data.ModelUsage.InputTokens,
		OutputTokens:        data.ModelUsage.OutputTokens,
		CacheReadTokens:     data.ModelUsage.CacheReadInputTokens,
		CacheCreationTokens: data.ModelUsage.CacheCreationInputTokens,
		IsError:             false,
		CreatedAt:           now,
	}

	h.analyticsRepo.InsertRequestLog(ctx, logEntry)
	h.analyticsRepo.IncrementDailyUsage(ctx, session.WorkspaceID, now, logEntry.Model,
		logEntry.InputTokens, logEntry.OutputTokens, logEntry.CacheReadTokens, logEntry.CacheCreationTokens)
}
