package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/pkg/sse"
	"github.com/go-chi/chi/v5"
)

type EventHandler struct {
	eventRepo   domain.EventRepository
	sessionRepo domain.SessionRepository
	eventBus    domain.EventBus
	runner      domain.SessionRunner
}

type sendEventsRequest struct {
	Events []domain.SendEventParams `json:"events"`
}

func NewEventHandler(
	eventRepo domain.EventRepository,
	sessionRepo domain.SessionRepository,
	eventBus domain.EventBus,
	runner domain.SessionRunner,
) *EventHandler {
	return &EventHandler{
		eventRepo:   eventRepo,
		sessionRepo: sessionRepo,
		eventBus:    eventBus,
		runner:      runner,
	}
}

func (h *EventHandler) Send(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")

	// Verify session exists and belongs to the workspace
	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	var req sendEventsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "events array is required")
		return
	}

	// Store user events and return them
	var storedEvents []*domain.Event
	for _, ep := range req.Events {
		payload, _ := json.Marshal(ep)

		evt := &domain.Event{
			ID:          domain.NewEventID(),
			SessionID:   sessionID,
			Type:        ep.Type,
			ProcessedAt: time.Now().UTC(),
			Payload:     payload,
		}

		if err := h.eventRepo.Create(r.Context(), evt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to store event")
			return
		}

		// Publish to event bus for SSE subscribers
		_ = h.eventBus.Publish(r.Context(), sessionID, evt)
		storedEvents = append(storedEvents, evt)
	}

	// If there's a user.message, trigger the runner asynchronously
	for _, ep := range req.Events {
		if ep.Type == domain.EventTypeUserMessage {
			go func() { _ = h.runner.Run(r.Context(), sessionID, req.Events) }()
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": storedEvents,
	})
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")

	// Verify session belongs to the workspace
	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	params := domain.EventListParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			params.Limit = &n
		}
	}
	if v := r.URL.Query().Get("page"); v != "" {
		params.Page = &v
	}
	if v := r.URL.Query().Get("order"); v != "" {
		params.Order = &v
	}

	events, nextPage, err := h.eventRepo.ListBySession(r.Context(), sessionID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	if events == nil {
		events = []*domain.Event{}
	}

	writeJSON(w, http.StatusOK, domain.ListResponse[*domain.Event]{
		Data:     events,
		NextPage: nextPage,
	})
}

func (h *EventHandler) Stream(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")

	// Verify session exists and belongs to the workspace
	session, err := h.sessionRepo.GetByID(r.Context(), sessionID)
	if err != nil || session.WorkspaceID != workspaceIDFromCtx(r) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	events, cancel, err := h.eventBus.Subscribe(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to subscribe to events")
		return
	}
	defer cancel()

	for {
		select {
		case evt, ok := <-events:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			if err := sseWriter.WriteEvent(evt.Type, data); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
