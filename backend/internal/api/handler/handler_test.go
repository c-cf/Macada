package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/c-cf/macada/internal/api"
	"github.com/c-cf/macada/internal/api/handler"
	"github.com/c-cf/macada/internal/domain"
)

// ---------------------------------------------------------------------------
// Mock repositories
// ---------------------------------------------------------------------------

// mockEnvironmentRepo is an in-memory implementation of domain.EnvironmentRepository.
type mockEnvironmentRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.Environment
}

func newMockEnvironmentRepo() *mockEnvironmentRepo {
	return &mockEnvironmentRepo{data: make(map[string]*domain.Environment)}
}

func (m *mockEnvironmentRepo) Create(_ context.Context, env *domain.Environment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[env.ID] = env
	return nil
}

func (m *mockEnvironmentRepo) GetByID(_ context.Context, id string) (*domain.Environment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	env, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return env, nil
}

func (m *mockEnvironmentRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Environment, *string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Environment, 0, len(m.data))
	for _, env := range m.data {
		result = append(result, env)
	}
	return result, nil, nil
}

func (m *mockEnvironmentRepo) Update(_ context.Context, env *domain.Environment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[env.ID]; !ok {
		return fmt.Errorf("not found")
	}
	m.data[env.ID] = env
	return nil
}

func (m *mockEnvironmentRepo) Archive(_ context.Context, id string) (*domain.Environment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	env, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	now := time.Now().UTC()
	archived := *env
	archived.ArchivedAt = &now
	m.data[id] = &archived
	return &archived, nil
}

func (m *mockEnvironmentRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[id]; !ok {
		return fmt.Errorf("not found")
	}
	delete(m.data, id)
	return nil
}

// mockAgentRepo is an in-memory implementation of domain.AgentRepository.
type mockAgentRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.Agent
}

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{data: make(map[string]*domain.Agent)}
}

func (m *mockAgentRepo) Create(_ context.Context, agent *domain.Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[agent.ID] = agent
	return nil
}

func (m *mockAgentRepo) GetByID(_ context.Context, id string) (*domain.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	agent, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return agent, nil
}

func (m *mockAgentRepo) List(_ context.Context, _ domain.AgentListParams) ([]*domain.Agent, *string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Agent, 0, len(m.data))
	for _, agent := range m.data {
		result = append(result, agent)
	}
	return result, nil, nil
}

func (m *mockAgentRepo) Update(_ context.Context, agent *domain.Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[agent.ID]; !ok {
		return fmt.Errorf("not found")
	}
	m.data[agent.ID] = agent
	return nil
}

func (m *mockAgentRepo) Archive(_ context.Context, id string) (*domain.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	agent, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	now := time.Now().UTC()
	archived := *agent
	archived.ArchivedAt = &now
	m.data[id] = &archived
	return &archived, nil
}

// mockSessionRepo is an in-memory implementation of domain.SessionRepository.
type mockSessionRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{data: make(map[string]*domain.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, session *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[session.ID] = session
	return nil
}

func (m *mockSessionRepo) GetByID(_ context.Context, id string) (*domain.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return session, nil
}

func (m *mockSessionRepo) List(_ context.Context, _ domain.SessionListParams) ([]*domain.Session, *string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*domain.Session, 0, len(m.data))
	for _, session := range m.data {
		result = append(result, session)
	}
	return result, nil, nil
}

func (m *mockSessionRepo) Update(_ context.Context, session *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[session.ID]; !ok {
		return fmt.Errorf("not found")
	}
	m.data[session.ID] = session
	return nil
}

func (m *mockSessionRepo) UpdateStatus(_ context.Context, id string, status domain.SessionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.data[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	session.Status = status
	return nil
}

func (m *mockSessionRepo) UpdateUsage(_ context.Context, id string, usage domain.SessionUsage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.data[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	session.Usage = usage
	return nil
}

func (m *mockSessionRepo) UpdateMemory(_ context.Context, id string, memory json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.data[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	session.Memory = memory
	return nil
}

func (m *mockSessionRepo) Archive(_ context.Context, id string) (*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	now := time.Now().UTC()
	archived := *session
	archived.ArchivedAt = &now
	m.data[id] = &archived
	return &archived, nil
}

// mockEventRepo is an in-memory implementation of domain.EventRepository.
type mockEventRepo struct {
	mu   sync.RWMutex
	data map[string][]*domain.Event // keyed by session ID
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{data: make(map[string][]*domain.Event)}
}

func (m *mockEventRepo) Create(_ context.Context, event *domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[event.SessionID] = append(m.data[event.SessionID], event)
	return nil
}

func (m *mockEventRepo) ListBySession(_ context.Context, sessionID string, _ domain.EventListParams) ([]*domain.Event, *string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	events := m.data[sessionID]
	if events == nil {
		events = []*domain.Event{}
	}
	return events, nil, nil
}

// mockEventBus is a no-op implementation of domain.EventBus.
type mockEventBus struct{}

func (m *mockEventBus) Publish(_ context.Context, _ string, _ *domain.Event) error {
	return nil
}

func (m *mockEventBus) Subscribe(_ context.Context, _ string) (<-chan *domain.Event, func(), error) {
	ch := make(chan *domain.Event)
	return ch, func() { close(ch) }, nil
}

// mockSessionRunner is a no-op implementation of domain.SessionRunner.
type mockSessionRunner struct{}

func (m *mockSessionRunner) Run(_ context.Context, _ string, _ []domain.SendEventParams) error {
	return nil
}

// mockSkillRepo is an in-memory implementation of domain.SkillRepository.
type mockSkillRepo struct {
	mu   sync.Mutex
	data map[string]*domain.Skill
}

func newMockSkillRepo() *mockSkillRepo {
	return &mockSkillRepo{data: make(map[string]*domain.Skill)}
}

func (m *mockSkillRepo) Create(_ context.Context, skill *domain.Skill) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[skill.ID] = skill
	return nil
}

func (m *mockSkillRepo) GetByID(_ context.Context, id string) (*domain.Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return s, nil
}

func (m *mockSkillRepo) GetByName(_ context.Context, _ string, name string) (*domain.Skill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.data {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockSkillRepo) List(_ context.Context, _ domain.SkillListParams) ([]*domain.Skill, *string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*domain.Skill, 0, len(m.data))
	for _, s := range m.data {
		result = append(result, s)
	}
	return result, nil, nil
}

func (m *mockSkillRepo) Update(_ context.Context, skill *domain.Skill) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[skill.ID] = skill
	return nil
}

func (m *mockSkillRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, id)
	return nil
}

// mockWorkspaceRepo is a minimal mock for domain.WorkspaceRepository.
type mockWorkspaceRepo struct{}

func (m *mockWorkspaceRepo) Create(_ context.Context, ws *domain.Workspace) error { return nil }
func (m *mockWorkspaceRepo) GetByID(_ context.Context, _ string) (*domain.Workspace, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockWorkspaceRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Workspace, *string, error) {
	return nil, nil, nil
}
func (m *mockWorkspaceRepo) Update(_ context.Context, _ *domain.Workspace) error { return nil }
func (m *mockWorkspaceRepo) Archive(_ context.Context, _ string) (*domain.Workspace, error) {
	return nil, fmt.Errorf("not found")
}

// mockAPIKeyRepo is a minimal mock for domain.APIKeyRepository.
// It always returns a valid API key for any hash, scoped to testWorkspaceID.
type mockAPIKeyRepo struct{}

const testWorkspaceID = "ws_TEST000000000000000000000000"

func (m *mockAPIKeyRepo) Create(_ context.Context, _ *domain.APIKey, _ string) error { return nil }
func (m *mockAPIKeyRepo) GetByHash(_ context.Context, _ string) (*domain.APIKey, error) {
	return &domain.APIKey{
		ID:          "key_TEST000000000000000000000000",
		WorkspaceID: testWorkspaceID,
		Name:        "test-key",
		KeyPrefix:   "ma_test12345",
		CreatedAt:   time.Now(),
		Type:        "api_key",
	}, nil
}
func (m *mockAPIKeyRepo) ListByWorkspace(_ context.Context, _ string, _ domain.ListParams) ([]*domain.APIKey, *string, error) {
	return nil, nil, nil
}
func (m *mockAPIKeyRepo) Revoke(_ context.Context, _ string, _ string) (*domain.APIKey, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockAPIKeyRepo) Delete(_ context.Context, _ string, _ string) error { return nil }
func (m *mockAPIKeyRepo) TouchLastUsed(_ context.Context, _ string) error    { return nil }

type mockFileRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.File
}

func newMockFileRepo() *mockFileRepo {
	return &mockFileRepo{data: make(map[string]*domain.File)}
}

func (m *mockFileRepo) Create(_ context.Context, f *domain.File) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[f.ID] = f
	return nil
}
func (m *mockFileRepo) GetByID(_ context.Context, id string) (*domain.File, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return f, nil
}
func (m *mockFileRepo) List(_ context.Context, _ domain.FileListParams) ([]*domain.File, *string, error) {
	return nil, nil, nil
}
func (m *mockFileRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, id)
	return nil
}

type mockResourceRepo struct {
	mu   sync.RWMutex
	data map[string]*domain.SessionResource
}

func newMockResourceRepo() *mockResourceRepo {
	return &mockResourceRepo{data: make(map[string]*domain.SessionResource)}
}

func (m *mockResourceRepo) Create(_ context.Context, r *domain.SessionResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[r.ID] = r
	return nil
}
func (m *mockResourceRepo) GetByID(_ context.Context, id string) (*domain.SessionResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return r, nil
}
func (m *mockResourceRepo) ListBySession(_ context.Context, _ string, _ domain.ListParams) ([]*domain.SessionResource, *string, error) {
	return nil, nil, nil
}
func (m *mockResourceRepo) Update(_ context.Context, _ *domain.SessionResource) error { return nil }
func (m *mockResourceRepo) Delete(_ context.Context, _ string) error                  { return nil }
func (m *mockResourceRepo) ListFileResourcesBySession(_ context.Context, _ string) ([]*domain.SessionResource, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Test setup helpers
// ---------------------------------------------------------------------------

type testHarness struct {
	router      http.Handler
	envRepo     *mockEnvironmentRepo
	agentRepo   *mockAgentRepo
	sessionRepo *mockSessionRepo
	eventRepo   *mockEventRepo
}

func newTestHarness() *testHarness {
	envRepo := newMockEnvironmentRepo()
	agentRepo := newMockAgentRepo()
	sessionRepo := newMockSessionRepo()
	eventRepo := newMockEventRepo()
	eventBus := &mockEventBus{}
	runner := &mockSessionRunner{}
	skillRepo := newMockSkillRepo()
	fileRepo := newMockFileRepo()
	resourceRepo := newMockResourceRepo()

	wsRepo := &mockWorkspaceRepo{}
	apiKeyRepo := &mockAPIKeyRepo{}
	bootstrapHandler := handler.NewBootstrapHandler("test-secret", wsRepo, apiKeyRepo)
	wsHandler := handler.NewWorkspaceHandler(wsRepo)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyRepo, wsRepo)

	router := api.NewRouter(api.Deps{
		EnvironmentHandler: handler.NewEnvironmentHandler(envRepo),
		AgentHandler:       handler.NewAgentHandler(agentRepo),
		SessionHandler:     handler.NewSessionHandler(sessionRepo, agentRepo, envRepo, resourceRepo, fileRepo),
		EventHandler:       handler.NewEventHandler(eventRepo, sessionRepo, eventBus, runner),
		SkillHandler:       handler.NewSkillHandler(skillRepo),
		WorkspaceHandler:   wsHandler,
		APIKeyHandler:      apiKeyHandler,
		BootstrapHandler:   bootstrapHandler,
		APIKeyRepo:         apiKeyRepo,
	})

	return &testHarness{
		router:      router,
		envRepo:     envRepo,
		agentRepo:   agentRepo,
		sessionRepo: sessionRepo,
		eventRepo:   eventRepo,
	}
}

func (h *testHarness) doRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", "test-api-key")
	rr := httptest.NewRecorder()
	h.router.ServeHTTP(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response body: %v\nbody: %s", err, rr.Body.String())
	}
	return result
}

// ---------------------------------------------------------------------------
// Environment handler tests
// ---------------------------------------------------------------------------

func TestEnvironmentCreate(t *testing.T) {
	h := newTestHarness()

	t.Run("creates environment with valid body", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/environments", map[string]interface{}{
			"name": "test-env",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["type"] != "environment" {
			t.Errorf("expected type %q, got %q", "environment", body["type"])
		}
		if body["name"] != "test-env" {
			t.Errorf("expected name %q, got %q", "test-env", body["name"])
		}
		id, ok := body["id"].(string)
		if !ok || id == "" {
			t.Error("expected non-empty id")
		}
	})

	t.Run("returns error without name", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/environments", map[string]interface{}{})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("creates environment with description and metadata", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/environments", map[string]interface{}{
			"name":        "full-env",
			"description": "a test environment",
			"metadata":    map[string]string{"team": "backend"},
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["description"] != "a test environment" {
			t.Errorf("expected description %q, got %q", "a test environment", body["description"])
		}
		meta, ok := body["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("expected metadata to be an object")
		}
		if meta["team"] != "backend" {
			t.Errorf("expected metadata.team %q, got %q", "backend", meta["team"])
		}
	})
}

func TestEnvironmentList(t *testing.T) {
	h := newTestHarness()

	t.Run("returns empty data array when no environments", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/environments", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data, ok := body["data"].([]interface{})
		if !ok {
			t.Fatal("expected data to be an array")
		}
		if len(data) != 0 {
			t.Errorf("expected empty data array, got %d items", len(data))
		}
		// next_page should be null (nil in Go JSON)
		if body["next_page"] != nil {
			t.Errorf("expected next_page to be null, got %v", body["next_page"])
		}
	})

	t.Run("returns created environments", func(t *testing.T) {
		// Create two environments
		h.doRequest("POST", "/v1/environments", map[string]interface{}{"name": "env-1"})
		h.doRequest("POST", "/v1/environments", map[string]interface{}{"name": "env-2"})

		rr := h.doRequest("GET", "/v1/environments", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data, ok := body["data"].([]interface{})
		if !ok {
			t.Fatal("expected data to be an array")
		}
		if len(data) != 2 {
			t.Errorf("expected 2 environments, got %d", len(data))
		}
	})
}

func TestEnvironmentRetrieve(t *testing.T) {
	h := newTestHarness()

	t.Run("retrieves existing environment", func(t *testing.T) {
		// Create
		createRR := h.doRequest("POST", "/v1/environments", map[string]interface{}{"name": "get-me"})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		// Retrieve
		rr := h.doRequest("GET", "/v1/environments/"+id, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["id"] != id {
			t.Errorf("expected id %q, got %q", id, body["id"])
		}
		if body["name"] != "get-me" {
			t.Errorf("expected name %q, got %q", "get-me", body["name"])
		}
	})

	t.Run("returns 404 for non-existent environment", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/environments/env_nonexistent", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestEnvironmentUpdate(t *testing.T) {
	h := newTestHarness()

	t.Run("updates environment name", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/environments", map[string]interface{}{"name": "old-name"})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("POST", "/v1/environments/"+id, map[string]interface{}{"name": "new-name"})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["name"] != "new-name" {
			t.Errorf("expected name %q, got %q", "new-name", body["name"])
		}
	})

	t.Run("returns 404 for non-existent environment", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/environments/env_nonexistent", map[string]interface{}{"name": "x"})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestEnvironmentArchive(t *testing.T) {
	h := newTestHarness()

	t.Run("archives environment and sets archived_at", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/environments", map[string]interface{}{"name": "to-archive"})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("POST", "/v1/environments/"+id+"/archive", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["archived_at"] == nil {
			t.Error("expected archived_at to be set, got nil")
		}
	})

	t.Run("returns 404 for non-existent environment", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/environments/env_nonexistent/archive", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestEnvironmentDelete(t *testing.T) {
	h := newTestHarness()

	t.Run("deletes environment and returns correct response", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/environments", map[string]interface{}{"name": "to-delete"})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("DELETE", "/v1/environments/"+id, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["id"] != id {
			t.Errorf("expected id %q, got %q", id, body["id"])
		}
		if body["type"] != "environment_deleted" {
			t.Errorf("expected type %q, got %q", "environment_deleted", body["type"])
		}

		// Verify it's actually gone
		getRR := h.doRequest("GET", "/v1/environments/"+id, nil)
		if getRR.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", getRR.Code)
		}
	})

	t.Run("returns 404 for non-existent environment", func(t *testing.T) {
		rr := h.doRequest("DELETE", "/v1/environments/env_nonexistent", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Agent handler tests
// ---------------------------------------------------------------------------

func TestAgentCreate(t *testing.T) {
	h := newTestHarness()

	t.Run("creates agent with string model", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"name":  "my-agent",
			"model": "claude-sonnet-4-6",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["type"] != "agent" {
			t.Errorf("expected type %q, got %q", "agent", body["type"])
		}
		if body["name"] != "my-agent" {
			t.Errorf("expected name %q, got %q", "my-agent", body["name"])
		}
		// version starts at 1
		if body["version"] != float64(1) {
			t.Errorf("expected version 1, got %v", body["version"])
		}

		// model should have id field
		model, ok := body["model"].(map[string]interface{})
		if !ok {
			t.Fatal("expected model to be an object")
		}
		if model["id"] != "claude-sonnet-4-6" {
			t.Errorf("expected model.id %q, got %q", "claude-sonnet-4-6", model["id"])
		}
	})

	t.Run("creates agent with object model", func(t *testing.T) {
		speed := "fast"
		rr := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"name":  "fast-agent",
			"model": map[string]interface{}{"id": "claude-sonnet-4-6", "speed": speed},
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		model := body["model"].(map[string]interface{})
		if model["id"] != "claude-sonnet-4-6" {
			t.Errorf("expected model.id %q, got %q", "claude-sonnet-4-6", model["id"])
		}
		if model["speed"] != "fast" {
			t.Errorf("expected model.speed %q, got %q", "fast", model["speed"])
		}
	})

	t.Run("returns error without name", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"model": "claude-sonnet-4-6",
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("creates agent with description and system prompt", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"name":        "detailed-agent",
			"model":       "claude-sonnet-4-6",
			"description": "A helpful agent",
			"system":      "You are a helpful assistant.",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["description"] != "A helpful agent" {
			t.Errorf("expected description %q, got %q", "A helpful agent", body["description"])
		}
		if body["system"] != "You are a helpful assistant." {
			t.Errorf("expected system %q, got %q", "You are a helpful assistant.", body["system"])
		}
	})
}

func TestAgentList(t *testing.T) {
	h := newTestHarness()

	t.Run("returns empty data array when no agents", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/agents", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data, ok := body["data"].([]interface{})
		if !ok {
			t.Fatal("expected data to be an array")
		}
		if len(data) != 0 {
			t.Errorf("expected empty data array, got %d items", len(data))
		}
	})

	t.Run("returns created agents", func(t *testing.T) {
		h.doRequest("POST", "/v1/agents", map[string]interface{}{"name": "a1", "model": "claude-sonnet-4-6"})
		h.doRequest("POST", "/v1/agents", map[string]interface{}{"name": "a2", "model": "claude-sonnet-4-6"})

		rr := h.doRequest("GET", "/v1/agents", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data := body["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("expected 2 agents, got %d", len(data))
		}
	})
}

func TestAgentRetrieve(t *testing.T) {
	h := newTestHarness()

	t.Run("retrieves existing agent", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"name":  "get-agent",
			"model": "claude-sonnet-4-6",
		})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("GET", "/v1/agents/"+id, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["id"] != id {
			t.Errorf("expected id %q, got %q", id, body["id"])
		}
		if body["name"] != "get-agent" {
			t.Errorf("expected name %q, got %q", "get-agent", body["name"])
		}
	})

	t.Run("returns 404 for non-existent agent", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/agents/agent_nonexistent", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestAgentUpdate(t *testing.T) {
	h := newTestHarness()

	t.Run("updates agent and increments version", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"name":  "update-me",
			"model": "claude-sonnet-4-6",
		})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		if created["version"] != float64(1) {
			t.Fatalf("expected initial version 1, got %v", created["version"])
		}

		// First update
		rr := h.doRequest("POST", "/v1/agents/"+id, map[string]interface{}{
			"name": "updated-name",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["name"] != "updated-name" {
			t.Errorf("expected name %q, got %q", "updated-name", body["name"])
		}
		if body["version"] != float64(2) {
			t.Errorf("expected version 2, got %v", body["version"])
		}

		// Second update
		rr2 := h.doRequest("POST", "/v1/agents/"+id, map[string]interface{}{
			"name": "updated-again",
		})
		body2 := decodeBody(t, rr2)
		if body2["version"] != float64(3) {
			t.Errorf("expected version 3, got %v", body2["version"])
		}
	})

	t.Run("returns 404 for non-existent agent", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/agents/agent_nonexistent", map[string]interface{}{"name": "x"})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestAgentArchive(t *testing.T) {
	h := newTestHarness()

	t.Run("archives agent and sets archived_at", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/agents", map[string]interface{}{
			"name":  "archive-me",
			"model": "claude-sonnet-4-6",
		})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("POST", "/v1/agents/"+id+"/archive", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["archived_at"] == nil {
			t.Error("expected archived_at to be set, got nil")
		}
		if body["type"] != "agent" {
			t.Errorf("expected type %q, got %q", "agent", body["type"])
		}
	})

	t.Run("returns 404 for non-existent agent", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/agents/agent_nonexistent/archive", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Session handler tests
// ---------------------------------------------------------------------------

// createTestEnvironment is a helper that creates an environment and returns its ID.
func createTestEnvironment(t *testing.T, h *testHarness) string {
	t.Helper()
	rr := h.doRequest("POST", "/v1/environments", map[string]interface{}{
		"name": "session-test-env",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to create test environment: %d %s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	return body["id"].(string)
}

// createTestAgent is a helper that creates an agent and returns its ID.
func createTestAgent(t *testing.T, h *testHarness) string {
	t.Helper()
	rr := h.doRequest("POST", "/v1/agents", map[string]interface{}{
		"name":  "session-test-agent",
		"model": "claude-sonnet-4-6",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to create test agent: %d %s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	return body["id"].(string)
}

func TestSessionCreate(t *testing.T) {
	h := newTestHarness()
	agentID := createTestAgent(t, h)
	envID := createTestEnvironment(t, h)

	t.Run("creates session with agent string ID", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent":          agentID,
			"environment_id": envID,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["type"] != "session" {
			t.Errorf("expected type %q, got %q", "session", body["type"])
		}
		if body["status"] != "idle" {
			t.Errorf("expected status %q, got %q", "idle", body["status"])
		}
		if body["environment_id"] != envID {
			t.Errorf("expected environment_id %q, got %q", envID, body["environment_id"])
		}

		// agent should be a snapshot object
		agentSnapshot, ok := body["agent"].(map[string]interface{})
		if !ok {
			t.Fatal("expected agent to be a JSON object (snapshot)")
		}
		if agentSnapshot["id"] != agentID {
			t.Errorf("expected agent.id %q, got %q", agentID, agentSnapshot["id"])
		}
	})

	t.Run("returns error without environment_id", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent": agentID,
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("returns 404 for non-existent agent", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent":          "agent_nonexistent",
			"environment_id": envID,
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("creates session with title and metadata", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent":          agentID,
			"environment_id": envID,
			"title":          "Debug session",
			"metadata":       map[string]string{"purpose": "debugging"},
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["title"] != "Debug session" {
			t.Errorf("expected title %q, got %q", "Debug session", body["title"])
		}
	})
}

func TestSessionList(t *testing.T) {
	h := newTestHarness()

	t.Run("returns empty data array when no sessions", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/sessions", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data := body["data"].([]interface{})
		if len(data) != 0 {
			t.Errorf("expected empty data array, got %d items", len(data))
		}
	})

	t.Run("returns created sessions", func(t *testing.T) {
		agentID := createTestAgent(t, h)
		envID := createTestEnvironment(t, h)
		h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent": agentID, "environment_id": envID,
		})
		h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent": agentID, "environment_id": envID,
		})

		rr := h.doRequest("GET", "/v1/sessions", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data := body["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("expected 2 sessions, got %d", len(data))
		}
	})
}

func TestSessionRetrieve(t *testing.T) {
	h := newTestHarness()
	agentID := createTestAgent(t, h)
	envID := createTestEnvironment(t, h)

	t.Run("retrieves existing session", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent": agentID, "environment_id": envID,
		})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("GET", "/v1/sessions/"+id, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["id"] != id {
			t.Errorf("expected id %q, got %q", id, body["id"])
		}
	})

	t.Run("returns 404 for non-existent session", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/sessions/sesn_nonexistent", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestSessionArchive(t *testing.T) {
	h := newTestHarness()
	agentID := createTestAgent(t, h)
	envID := createTestEnvironment(t, h)

	t.Run("archives session and sets archived_at", func(t *testing.T) {
		createRR := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
			"agent": agentID, "environment_id": envID,
		})
		created := decodeBody(t, createRR)
		id := created["id"].(string)

		rr := h.doRequest("POST", "/v1/sessions/"+id+"/archive", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		if body["archived_at"] == nil {
			t.Error("expected archived_at to be set, got nil")
		}
	})

	t.Run("returns 404 for non-existent session", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions/sesn_nonexistent/archive", nil)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Event handler tests
// ---------------------------------------------------------------------------

// createTestSession is a helper that creates an agent + environment + session and returns the session ID.
func createTestSession(t *testing.T, h *testHarness) string {
	t.Helper()
	agentID := createTestAgent(t, h)
	envID := createTestEnvironment(t, h)
	rr := h.doRequest("POST", "/v1/sessions", map[string]interface{}{
		"agent":          agentID,
		"environment_id": envID,
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("failed to create test session: %d %s", rr.Code, rr.Body.String())
	}
	body := decodeBody(t, rr)
	return body["id"].(string)
}

func TestEventSend(t *testing.T) {
	h := newTestHarness()
	sessionID := createTestSession(t, h)

	t.Run("sends user.message event", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions/"+sessionID+"/events", map[string]interface{}{
			"events": []map[string]interface{}{
				{
					"type":    "user.message",
					"content": []map[string]string{{"type": "text", "text": "Hello!"}},
				},
			},
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		data, ok := body["data"].([]interface{})
		if !ok {
			t.Fatal("expected data to be an array")
		}
		if len(data) != 1 {
			t.Fatalf("expected 1 event, got %d", len(data))
		}

		evt := data[0].(map[string]interface{})
		if evt["type"] != "user.message" {
			t.Errorf("expected type %q, got %q", "user.message", evt["type"])
		}
		if evt["id"] == nil || evt["id"] == "" {
			t.Error("expected event to have an ID")
		}
	})

	t.Run("returns error with empty events array", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions/"+sessionID+"/events", map[string]interface{}{
			"events": []map[string]interface{}{},
		})
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("returns 404 for non-existent session", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions/sesn_nonexistent/events", map[string]interface{}{
			"events": []map[string]interface{}{
				{"type": "user.message", "content": []map[string]string{{"type": "text", "text": "Hello!"}}},
			},
		})
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("sends multiple events at once", func(t *testing.T) {
		rr := h.doRequest("POST", "/v1/sessions/"+sessionID+"/events", map[string]interface{}{
			"events": []map[string]interface{}{
				{"type": "user.message", "content": []map[string]string{{"type": "text", "text": "First"}}},
				{"type": "user.message", "content": []map[string]string{{"type": "text", "text": "Second"}}},
			},
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		body := decodeBody(t, rr)
		data := body["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("expected 2 events, got %d", len(data))
		}
	})
}

func TestEventList(t *testing.T) {
	h := newTestHarness()
	sessionID := createTestSession(t, h)

	t.Run("returns empty data array with no events", func(t *testing.T) {
		rr := h.doRequest("GET", "/v1/sessions/"+sessionID+"/events", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data := body["data"].([]interface{})
		if len(data) != 0 {
			t.Errorf("expected empty data array, got %d items", len(data))
		}
	})

	t.Run("returns events after sending", func(t *testing.T) {
		// Send an event
		h.doRequest("POST", "/v1/sessions/"+sessionID+"/events", map[string]interface{}{
			"events": []map[string]interface{}{
				{"type": "user.message", "content": []map[string]string{{"type": "text", "text": "Hello!"}}},
			},
		})

		rr := h.doRequest("GET", "/v1/sessions/"+sessionID+"/events", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		body := decodeBody(t, rr)
		data := body["data"].([]interface{})
		if len(data) < 1 {
			t.Errorf("expected at least 1 event, got %d", len(data))
		}
	})
}

// ---------------------------------------------------------------------------
// Health check test
// ---------------------------------------------------------------------------

func TestHealthEndpoint(t *testing.T) {
	h := newTestHarness()

	rr := h.doRequest("GET", "/health", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := decodeBody(t, rr)
	if body["status"] != "ok" {
		t.Errorf("expected status %q, got %q", "ok", body["status"])
	}
}
