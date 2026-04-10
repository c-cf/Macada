package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/c-cf/macada/internal/api/handler"
	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/internal/sandbox"
	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// Helpers specific to InternalHandler tests
// ---------------------------------------------------------------------------

// mockHeartbeatRecorder tracks RecordHeartbeat calls.
type mockHeartbeatRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (m *mockHeartbeatRecorder) RecordHeartbeat(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, sessionID)
}

func (m *mockHeartbeatRecorder) getCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.calls))
	copy(cp, m.calls)
	return cp
}

// newInternalTestHandler creates an InternalHandler wired to mocks.
func newInternalTestHandler(hbr sandbox.HeartbeatRecorder) (*handler.InternalHandler, *mockEventRepo, *mockSessionRepo) {
	eventRepo := newMockEventRepo()
	sessionRepo := newMockSessionRepo()
	eventBus := &mockEventBus{}
	tokenGen := sandbox.NewTokenGenerator("test-secret")

	h := handler.NewInternalHandler(eventRepo, sessionRepo, eventBus, nil, tokenGen, nil, hbr)
	return h, eventRepo, sessionRepo
}

// serveInternal invokes InternalHandler.IngestEvents via a chi router
// to inject the {session_id} URL parameter, using the given sandbox token.
func serveInternal(h *handler.InternalHandler, sessionID, token string, body interface{}) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/sandbox/"+sessionID+"/events", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()

	// Use chi router to inject URL params
	r := chi.NewRouter()
	r.Post("/internal/v1/sandbox/{session_id}/events", h.IngestEvents)
	r.ServeHTTP(rr, req)
	return rr
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestIngestEvents_HeartbeatCallsRecorder(t *testing.T) {
	hbr := &mockHeartbeatRecorder{}
	h, _, _ := newInternalTestHandler(hbr)

	sessionID := "sesn_hb01"
	token := sandbox.NewTokenGenerator("test-secret").Generate(sessionID)

	body := map[string]interface{}{
		"events": []map[string]interface{}{
			{"type": domain.EventTypeRuntimeHeartbeat},
		},
	}

	rr := serveInternal(h, sessionID, token, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	calls := hbr.getCalls()
	if len(calls) != 1 || calls[0] != sessionID {
		t.Errorf("RecordHeartbeat calls = %v, want [%s]", calls, sessionID)
	}
}

func TestIngestEvents_HeartbeatNotPersisted(t *testing.T) {
	hbr := &mockHeartbeatRecorder{}
	h, eventRepo, _ := newInternalTestHandler(hbr)

	sessionID := "sesn_hb02"
	token := sandbox.NewTokenGenerator("test-secret").Generate(sessionID)

	body := map[string]interface{}{
		"events": []map[string]interface{}{
			{"type": domain.EventTypeRuntimeHeartbeat},
		},
	}

	rr := serveInternal(h, sessionID, token, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// Heartbeat events should NOT be persisted
	events, _, _ := eventRepo.ListBySession(context.Background(), sessionID, domain.EventListParams{})
	if len(events) != 0 {
		t.Errorf("expected 0 persisted events, got %d", len(events))
	}
}

func TestIngestEvents_MixedEventsWithHeartbeat(t *testing.T) {
	hbr := &mockHeartbeatRecorder{}
	h, eventRepo, sessionRepo := newInternalTestHandler(hbr)

	sessionID := "sesn_hb03"
	token := sandbox.NewTokenGenerator("test-secret").Generate(sessionID)

	// Pre-create session for status update
	_ = sessionRepo.Create(context.Background(), &domain.Session{
		ID:     sessionID,
		Status: domain.SessionStatusRunning,
	})

	body := map[string]interface{}{
		"events": []map[string]interface{}{
			{"type": domain.EventTypeRuntimeHeartbeat},
			{"type": domain.EventTypeSessionIdle},
			{"type": domain.EventTypeRuntimeHeartbeat},
		},
	}

	rr := serveInternal(h, sessionID, token, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	// 2 heartbeats should call recorder twice
	calls := hbr.getCalls()
	if len(calls) != 2 {
		t.Errorf("RecordHeartbeat calls = %d, want 2", len(calls))
	}

	// Only session.status_idle should be persisted
	events, _, _ := eventRepo.ListBySession(context.Background(), sessionID, domain.EventListParams{})
	if len(events) != 1 {
		t.Fatalf("expected 1 persisted event, got %d", len(events))
	}
	if events[0].Type != domain.EventTypeSessionIdle {
		t.Errorf("persisted event type = %q, want %q", events[0].Type, domain.EventTypeSessionIdle)
	}
}

func TestIngestEvents_NilHeartbeatRecorder_NoError(t *testing.T) {
	// Passing nil heartbeat recorder should not panic
	h, _, _ := newInternalTestHandler(nil)

	sessionID := "sesn_hb04"
	token := sandbox.NewTokenGenerator("test-secret").Generate(sessionID)

	body := map[string]interface{}{
		"events": []map[string]interface{}{
			{"type": domain.EventTypeRuntimeHeartbeat},
		},
	}

	rr := serveInternal(h, sessionID, token, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
}

func TestIngestEvents_InvalidToken_Rejected(t *testing.T) {
	h, _, _ := newInternalTestHandler(nil)

	body := map[string]interface{}{
		"events": []map[string]interface{}{
			{"type": domain.EventTypeRuntimeHeartbeat},
		},
	}

	rr := serveInternal(h, "sesn_hb05", "bad-token", body)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}
