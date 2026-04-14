package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/internal/infra/postgres"
	"github.com/c-cf/macada/internal/sandbox"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// InternalHandler handles events arriving from sandbox runtimes.
type InternalHandler struct {
	eventRepo          domain.EventRepository
	sessionRepo        domain.SessionRepository
	eventBus           domain.EventBus
	analyticsRepo      *postgres.AnalyticsRepo
	tokenGen           *sandbox.TokenGenerator
	fileHandler        *FileHandler // for internal file uploads
	heartbeatRecorder  sandbox.HeartbeatRecorder
}

// NewInternalHandler creates a new internal event handler.
func NewInternalHandler(
	eventRepo domain.EventRepository,
	sessionRepo domain.SessionRepository,
	eventBus domain.EventBus,
	analyticsRepo *postgres.AnalyticsRepo,
	tokenGen *sandbox.TokenGenerator,
	fileHandler *FileHandler,
	heartbeatRecorder sandbox.HeartbeatRecorder,
) *InternalHandler {
	return &InternalHandler{
		eventRepo:         eventRepo,
		sessionRepo:       sessionRepo,
		eventBus:          eventBus,
		analyticsRepo:     analyticsRepo,
		tokenGen:          tokenGen,
		fileHandler:       fileHandler,
		heartbeatRecorder: heartbeatRecorder,
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
		// Heartbeat events update the sandbox's last-seen timestamp but are not persisted.
		if ie.Type == domain.EventTypeRuntimeHeartbeat {
			if h.heartbeatRecorder != nil {
				h.heartbeatRecorder.RecordHeartbeat(sessionID)
			}
			continue
		}

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

		if pubErr := h.eventBus.Publish(r.Context(), sessionID, evt); pubErr != nil {
			log.Warn().Err(pubErr).Str("session_id", sessionID).Msg("failed to publish event to bus")
		}

		// Handle session status transitions
		switch ie.Type {
		case domain.EventTypeSessionIdle:
			if err := h.sessionRepo.UpdateStatus(r.Context(), sessionID, domain.SessionStatusIdle); err != nil {
				log.Error().Err(err).Str("session_id", sessionID).Msg("failed to update session status to idle")
			}

		case domain.EventTypeSessionRunning:
			if err := h.sessionRepo.UpdateStatus(r.Context(), sessionID, domain.SessionStatusRunning); err != nil {
				log.Error().Err(err).Str("session_id", sessionID).Msg("failed to update session status to running")
			}

		case domain.EventTypeSessionTerminated:
			if err := h.sessionRepo.UpdateStatus(r.Context(), sessionID, domain.SessionStatusTerminated); err != nil {
				log.Error().Err(err).Str("session_id", sessionID).Msg("failed to update session status to terminated")
			}

		case domain.EventTypeSessionError:
			log.Warn().Str("session_id", sessionID).RawJSON("error", ie.Payload).Msg("session error reported by runtime")

		case "span.model_request_end":
			h.recordAnalytics(r.Context(), sessionID, ie.Payload)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UploadFile handles internal file upload from sandbox runtime.
func (h *InternalHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")

	// Validate sandbox token
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || !h.tokenGen.Validate(sessionID, token) {
		writeError(w, http.StatusUnauthorized, "invalid sandbox token")
		return
	}

	// Look up session to get workspace ID
	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	h.fileHandler.UploadInternal(w, r, session.WorkspaceID)
}

func (h *InternalHandler) recordAnalytics(ctx context.Context, sessionID string, payload json.RawMessage) {
	var data struct {
		ModelUsage domain.ModelUsage `json:"model_usage"`
		Model      string            `json:"model"`
		IsError    bool              `json:"is_error"`
	}
	if json.Unmarshal(payload, &data) != nil || data.IsError {
		return
	}
	if data.Model == "" {
		data.Model = "unknown"
	}

	// Extract agent ID from session
	session, err := h.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return
	}
	var snap struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(session.Agent, &snap); err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("failed to extract agent ID from session snapshot")
	}

	now := time.Now().UTC()
	logEntry := postgres.LogRow{
		ID:                  domain.NewLLMLogID(),
		WorkspaceID:         session.WorkspaceID,
		SessionID:           sessionID,
		AgentID:             snap.ID,
		Model:               data.Model,
		InputTokens:         data.ModelUsage.InputTokens,
		OutputTokens:        data.ModelUsage.OutputTokens,
		CacheReadTokens:     data.ModelUsage.CacheReadInputTokens,
		CacheCreationTokens: data.ModelUsage.CacheCreationInputTokens,
		IsError:             false,
		CreatedAt:           now,
	}

	if err := h.analyticsRepo.InsertRequestLog(ctx, logEntry); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to insert analytics log")
	}
	if err := h.analyticsRepo.IncrementDailyUsage(ctx, session.WorkspaceID, now, logEntry.Model,
		logEntry.InputTokens, logEntry.OutputTokens, logEntry.CacheReadTokens, logEntry.CacheCreationTokens); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to increment daily usage")
	}

	// Incrementally update session usage totals
	usage := session.Usage
	input := valOrZero(usage.InputTokens) + data.ModelUsage.InputTokens
	output := valOrZero(usage.OutputTokens) + data.ModelUsage.OutputTokens
	cacheRead := valOrZero(usage.CacheReadInputTokens) + data.ModelUsage.CacheReadInputTokens
	usage.InputTokens = &input
	usage.OutputTokens = &output
	usage.CacheReadInputTokens = &cacheRead
	if err := h.sessionRepo.UpdateUsage(ctx, sessionID, usage); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to update session usage")
	}
}

func valOrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
