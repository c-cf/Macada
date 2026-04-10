package sandbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/c-cf/macada/internal/domain"
	rtctx "github.com/c-cf/macada/internal/runtime/context"
)

// ---------------------------------------------------------------------------
// Mock dependencies
// ---------------------------------------------------------------------------

type mockSessionRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{data: make(map[string]*domain.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[s.ID] = s
	return nil
}

func (m *mockSessionRepo) GetByID(_ context.Context, id string) (*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return s, nil
}

func (m *mockSessionRepo) List(_ context.Context, _ domain.SessionListParams) ([]*domain.Session, *string, error) {
	return nil, nil, nil
}

func (m *mockSessionRepo) Update(_ context.Context, s *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[s.ID] = s
	return nil
}

func (m *mockSessionRepo) UpdateStatus(_ context.Context, id string, status domain.SessionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.data[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	s.Status = status
	return nil
}

func (m *mockSessionRepo) UpdateUsage(_ context.Context, _ string, _ domain.SessionUsage) error {
	return nil
}

func (m *mockSessionRepo) UpdateMemory(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

func (m *mockSessionRepo) Archive(_ context.Context, _ string) (*domain.Session, error) {
	return nil, nil
}

func (m *mockSessionRepo) getStatus(id string) domain.SessionStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.data[id]; ok {
		return s.Status
	}
	return ""
}

type mockEventRepo struct {
	mu     sync.Mutex
	events []*domain.Event
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{}
}

func (m *mockEventRepo) Create(_ context.Context, event *domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockEventRepo) ListBySession(_ context.Context, sessionID string, _ domain.EventListParams) ([]*domain.Event, *string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Event
	for _, e := range m.events {
		if e.SessionID == sessionID {
			result = append(result, e)
		}
	}
	return result, nil, nil
}

func (m *mockEventRepo) findByType(eventType string) []*domain.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Event
	for _, e := range m.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

type mockEventBus struct{}

func (m *mockEventBus) Publish(_ context.Context, _ string, _ *domain.Event) error { return nil }
func (m *mockEventBus) Subscribe(_ context.Context, _ string) (<-chan *domain.Event, func(), error) {
	ch := make(chan *domain.Event)
	return ch, func() { close(ch) }, nil
}

// newTestOrchestrator creates an Orchestrator with mock dependencies for testing.
func newTestOrchestrator(sessionRepo domain.SessionRepository, eventRepo domain.EventRepository, eventBus domain.EventBus) *Orchestrator {
	return &Orchestrator{
		sessionRepo: sessionRepo,
		eventRepo:   eventRepo,
		eventBus:    eventBus,
		compressor:  rtctx.NewCompressor(rtctx.DefaultCompressionConfig()),
		sandboxes:   make(map[string]*SandboxInfo),
	}
}

// ---------------------------------------------------------------------------
// Tests: RecordHeartbeat
// ---------------------------------------------------------------------------

func TestRecordHeartbeat_UpdatesTimestamp(t *testing.T) {
	o := newTestOrchestrator(newMockSessionRepo(), newMockEventRepo(), &mockEventBus{})

	created := time.Now().UTC().Add(-5 * time.Minute)
	o.sandboxes["sesn_01"] = &SandboxInfo{
		ID:              "sandbox-sesn_01",
		SessionID:       "sesn_01",
		Status:          SandboxStatusRunning,
		CreatedAt:       created,
		LastHeartbeatAt: &created,
	}

	before := time.Now().UTC()
	o.RecordHeartbeat("sesn_01")
	after := time.Now().UTC()

	sbx := o.sandboxes["sesn_01"]
	if sbx.LastHeartbeatAt == nil {
		t.Fatal("LastHeartbeatAt should not be nil after heartbeat")
	}
	if sbx.LastHeartbeatAt.Before(before) || sbx.LastHeartbeatAt.After(after) {
		t.Errorf("LastHeartbeatAt = %v, want between %v and %v", *sbx.LastHeartbeatAt, before, after)
	}
}

func TestRecordHeartbeat_UnknownSession_NoOp(t *testing.T) {
	o := newTestOrchestrator(newMockSessionRepo(), newMockEventRepo(), &mockEventBus{})

	// Should not panic for unknown session
	o.RecordHeartbeat("sesn_nonexistent")

	if len(o.sandboxes) != 0 {
		t.Error("should not create sandbox entry for unknown session")
	}
}

func TestRecordHeartbeat_MultipleCalls_UpdatesEachTime(t *testing.T) {
	o := newTestOrchestrator(newMockSessionRepo(), newMockEventRepo(), &mockEventBus{})

	created := time.Now().UTC().Add(-5 * time.Minute)
	o.sandboxes["sesn_01"] = &SandboxInfo{
		ID:              "sandbox-sesn_01",
		SessionID:       "sesn_01",
		Status:          SandboxStatusRunning,
		CreatedAt:       created,
		LastHeartbeatAt: &created,
	}

	o.RecordHeartbeat("sesn_01")
	first := *o.sandboxes["sesn_01"].LastHeartbeatAt

	time.Sleep(1 * time.Millisecond)
	o.RecordHeartbeat("sesn_01")
	second := *o.sandboxes["sesn_01"].LastHeartbeatAt

	if !second.After(first) {
		t.Errorf("second heartbeat (%v) should be after first (%v)", second, first)
	}
}

// ---------------------------------------------------------------------------
// Tests: checkStaleHeartbeats
// ---------------------------------------------------------------------------

func TestCheckStaleHeartbeats_DetectsStale(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	o := newTestOrchestrator(sessionRepo, eventRepo, &mockEventBus{})

	sessionRepo.data["sesn_stale"] = &domain.Session{
		ID:     "sesn_stale",
		Status: domain.SessionStatusRunning,
	}

	staleTime := time.Now().UTC().Add(-2 * time.Minute) // well past 90s threshold
	o.sandboxes["sesn_stale"] = &SandboxInfo{
		ID:              "sandbox-sesn_stale",
		SessionID:       "sesn_stale",
		Status:          SandboxStatusRunning,
		CreatedAt:       staleTime,
		LastHeartbeatAt: &staleTime,
	}

	o.checkStaleHeartbeats(context.Background())

	// Sandbox should be removed from memory
	if _, ok := o.sandboxes["sesn_stale"]; ok {
		t.Error("stale sandbox should be removed from in-memory map")
	}

	// runtime.stopped event should have been emitted
	stoppedEvents := eventRepo.findByType(domain.EventTypeRuntimeStopped)
	if len(stoppedEvents) != 1 {
		t.Fatalf("expected 1 runtime.stopped event, got %d", len(stoppedEvents))
	}
	if stoppedEvents[0].SessionID != "sesn_stale" {
		t.Errorf("event session_id = %q, want sesn_stale", stoppedEvents[0].SessionID)
	}

	// Verify event payload contains reason
	var payload map[string]string
	if err := json.Unmarshal(stoppedEvents[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["reason"] != "heartbeat_timeout" {
		t.Errorf("reason = %q, want heartbeat_timeout", payload["reason"])
	}

	// Session status should be idle
	if status := sessionRepo.getStatus("sesn_stale"); status != domain.SessionStatusIdle {
		t.Errorf("session status = %q, want %q", status, domain.SessionStatusIdle)
	}
}

func TestCheckStaleHeartbeats_SkipsFreshSandboxes(t *testing.T) {
	eventRepo := newMockEventRepo()
	o := newTestOrchestrator(newMockSessionRepo(), eventRepo, &mockEventBus{})

	freshTime := time.Now().UTC()
	o.sandboxes["sesn_fresh"] = &SandboxInfo{
		ID:              "sandbox-sesn_fresh",
		SessionID:       "sesn_fresh",
		Status:          SandboxStatusRunning,
		CreatedAt:       freshTime,
		LastHeartbeatAt: &freshTime,
	}

	o.checkStaleHeartbeats(context.Background())

	// Sandbox should still be present
	if _, ok := o.sandboxes["sesn_fresh"]; !ok {
		t.Error("fresh sandbox should not be removed")
	}

	if events := eventRepo.findByType(domain.EventTypeRuntimeStopped); len(events) != 0 {
		t.Errorf("expected 0 runtime.stopped events, got %d", len(events))
	}
}

func TestCheckStaleHeartbeats_SkipsNonRunningSandboxes(t *testing.T) {
	eventRepo := newMockEventRepo()
	o := newTestOrchestrator(newMockSessionRepo(), eventRepo, &mockEventBus{})

	staleTime := time.Now().UTC().Add(-2 * time.Minute)
	o.sandboxes["sesn_stopped"] = &SandboxInfo{
		ID:              "sandbox-sesn_stopped",
		SessionID:       "sesn_stopped",
		Status:          SandboxStatusStopped,
		CreatedAt:       staleTime,
		LastHeartbeatAt: &staleTime,
	}

	o.checkStaleHeartbeats(context.Background())

	if events := eventRepo.findByType(domain.EventTypeRuntimeStopped); len(events) != 0 {
		t.Errorf("expected 0 events for stopped sandbox, got %d", len(events))
	}
}

func TestCheckStaleHeartbeats_NilHeartbeatFallsBackToCreatedAt(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	o := newTestOrchestrator(sessionRepo, eventRepo, &mockEventBus{})

	sessionRepo.data["sesn_new"] = &domain.Session{
		ID:     "sesn_new",
		Status: domain.SessionStatusRunning,
	}

	staleCreated := time.Now().UTC().Add(-2 * time.Minute)
	o.sandboxes["sesn_new"] = &SandboxInfo{
		ID:              "sandbox-sesn_new",
		SessionID:       "sesn_new",
		Status:          SandboxStatusRunning,
		CreatedAt:       staleCreated,
		LastHeartbeatAt: nil,
	}

	o.checkStaleHeartbeats(context.Background())

	if _, ok := o.sandboxes["sesn_new"]; ok {
		t.Error("sandbox with nil heartbeat and stale CreatedAt should be removed")
	}
}

func TestCheckStaleHeartbeats_MixedSandboxes(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	o := newTestOrchestrator(sessionRepo, eventRepo, &mockEventBus{})

	sessionRepo.data["sesn_stale"] = &domain.Session{ID: "sesn_stale", Status: domain.SessionStatusRunning}
	sessionRepo.data["sesn_fresh"] = &domain.Session{ID: "sesn_fresh", Status: domain.SessionStatusRunning}

	staleTime := time.Now().UTC().Add(-2 * time.Minute)
	freshTime := time.Now().UTC()

	o.sandboxes["sesn_stale"] = &SandboxInfo{
		ID: "sandbox-sesn_stale", SessionID: "sesn_stale",
		Status: SandboxStatusRunning, CreatedAt: staleTime, LastHeartbeatAt: &staleTime,
	}
	o.sandboxes["sesn_fresh"] = &SandboxInfo{
		ID: "sandbox-sesn_fresh", SessionID: "sesn_fresh",
		Status: SandboxStatusRunning, CreatedAt: freshTime, LastHeartbeatAt: &freshTime,
	}

	o.checkStaleHeartbeats(context.Background())

	if _, ok := o.sandboxes["sesn_stale"]; ok {
		t.Error("stale sandbox should be removed")
	}
	if _, ok := o.sandboxes["sesn_fresh"]; !ok {
		t.Error("fresh sandbox should remain")
	}

	stoppedEvents := eventRepo.findByType(domain.EventTypeRuntimeStopped)
	if len(stoppedEvents) != 1 {
		t.Fatalf("expected 1 runtime.stopped event, got %d", len(stoppedEvents))
	}
}

// ---------------------------------------------------------------------------
// Tests: StartWatcher lifecycle
// ---------------------------------------------------------------------------

func TestStartWatcher_StopsOnContextCancel(t *testing.T) {
	o := newTestOrchestrator(newMockSessionRepo(), newMockEventRepo(), &mockEventBus{})

	ctx, cancel := context.WithCancel(context.Background())
	o.StartWatcher(ctx)

	// Cancel immediately — watcher goroutine should exit cleanly
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// ---------------------------------------------------------------------------
// Tests: Run — forwardPayload failure emits events
// ---------------------------------------------------------------------------

func TestRun_ForwardFailure_EmitsErrorAndCleansUp(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	o := newTestOrchestrator(sessionRepo, eventRepo, &mockEventBus{})

	sessionRepo.data["sesn_fwd"] = &domain.Session{
		ID:     "sesn_fwd",
		Status: domain.SessionStatusRunning,
		Agent:  json.RawMessage(`{"id":"agent_01","model":{"id":"claude-sonnet-4-6"}}`),
		Memory: json.RawMessage(`{}`),
	}

	// Pre-seed sandbox with unreachable IP (RFC 5737 TEST-NET)
	now := time.Now().UTC()
	o.sandboxes["sesn_fwd"] = &SandboxInfo{
		ID:              "sandbox-sesn_fwd",
		SessionID:       "sesn_fwd",
		ContainerID:     "container123",
		ContainerIP:     "192.0.2.1",
		Status:          SandboxStatusRunning,
		CreatedAt:       now,
		LastHeartbeatAt: &now,
	}

	err := o.Run(context.Background(), "sesn_fwd", []domain.SendEventParams{
		{Type: "user.message", Content: json.RawMessage(`"hello"`)},
	})

	if err == nil {
		t.Fatal("Run should fail when forward fails")
	}

	// runtime.error events should have been emitted (forward_failed)
	errEvents := eventRepo.findByType(domain.EventTypeRuntimeError)
	var forwardFailed *domain.Event
	for _, e := range errEvents {
		var p map[string]string
		if json.Unmarshal(e.Payload, &p) == nil && p["reason"] == "forward_failed" {
			forwardFailed = e
			break
		}
	}
	if forwardFailed == nil {
		t.Fatal("expected runtime.error event with reason=forward_failed")
	}
	if forwardFailed.SessionID != "sesn_fwd" {
		t.Errorf("error event session_id = %q, want sesn_fwd", forwardFailed.SessionID)
	}

	// Session should be set back to idle
	if status := sessionRepo.getStatus("sesn_fwd"); status != domain.SessionStatusIdle {
		t.Errorf("session status = %q, want %q", status, domain.SessionStatusIdle)
	}

	// Sandbox should be removed from in-memory map
	if _, ok := o.sandboxes["sesn_fwd"]; ok {
		t.Error("sandbox should be removed after forward failure")
	}
}
