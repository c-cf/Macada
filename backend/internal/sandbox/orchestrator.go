package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/internal/infra/postgres"
	rtctx "github.com/cchu-code/managed-agents/internal/runtime/context"
	"github.com/rs/zerolog/log"
)

const (
	healthPollInterval = 500 * time.Millisecond
	healthPollTimeout  = 60 * time.Second
	runtimePort        = 9090
)

// OrchestratorConfig holds settings for the sandbox orchestrator.
type OrchestratorConfig struct {
	RuntimeImage    string // Docker image for the agent runtime
	ControlPlaneURL string // URL the runtime uses to call back
	AnthropicAPIKey string // API key passed to the runtime
	DockerHost      string // Docker daemon URL
	NetworkName     string // Docker network for sandbox containers
}

// Orchestrator manages sandbox containers and implements domain.SessionRunner.
type Orchestrator struct {
	config        OrchestratorConfig
	docker        *DockerClient
	deployer      *Deployer
	tokenGen      *TokenGenerator
	compressor    *rtctx.Compressor
	sessionRepo   domain.SessionRepository
	eventRepo     domain.EventRepository
	eventBus      domain.EventBus
	skillRepo     domain.SkillRepository
	envRepo       domain.EnvironmentRepository
	analyticsRepo *postgres.AnalyticsRepo

	mu        sync.Mutex
	sandboxes map[string]*SandboxInfo // sessionID -> sandbox
}

// NewOrchestrator creates a new sandbox orchestrator.
func NewOrchestrator(
	config OrchestratorConfig,
	tokenGen *TokenGenerator,
	compressor *rtctx.Compressor,
	sessionRepo domain.SessionRepository,
	eventRepo domain.EventRepository,
	eventBus domain.EventBus,
	skillRepo domain.SkillRepository,
	envRepo domain.EnvironmentRepository,
	analyticsRepo *postgres.AnalyticsRepo,
) *Orchestrator {
	return &Orchestrator{
		config:        config,
		docker:        NewDockerClient(config.DockerHost),
		deployer:      NewDeployer(),
		tokenGen:      tokenGen,
		compressor:    compressor,
		sessionRepo:   sessionRepo,
		eventRepo:     eventRepo,
		eventBus:      eventBus,
		skillRepo:     skillRepo,
		envRepo:       envRepo,
		analyticsRepo: analyticsRepo,
		sandboxes:     make(map[string]*SandboxInfo),
	}
}

// Run implements domain.SessionRunner.
// It provisions a sandbox (if needed), compresses history, and forwards to runtime.
func (o *Orchestrator) Run(ctx context.Context, sessionID string, events []domain.SendEventParams) error {
	bgCtx := context.Background()

	// Update session status
	_ = o.sessionRepo.UpdateStatus(bgCtx, sessionID, domain.SessionStatusRunning)
	o.emitEvent(bgCtx, sessionID, domain.EventTypeSessionRunning, nil)

	// Ensure sandbox is running
	sbx, err := o.ensureSandbox(bgCtx, sessionID)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to provision sandbox")
		o.emitEvent(bgCtx, sessionID, "runtime.error", map[string]string{"error": err.Error()})
		return err
	}

	// Build compressed payload for runtime
	payload, err := o.buildForwardPayload(bgCtx, sessionID, events)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to build forward payload")
		return err
	}

	// Forward to runtime
	if err := o.forwardPayload(bgCtx, sbx, payload); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to forward payload")
		return err
	}

	return nil
}

// buildForwardPayload fetches all events, compresses history, and builds the payload.
func (o *Orchestrator) buildForwardPayload(ctx context.Context, sessionID string, newEvents []domain.SendEventParams) (*ForwardPayload, error) {
	// Load session for existing memory
	session, err := o.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	// Parse existing memory
	var existingMemory *rtctx.SessionMemory
	if len(session.Memory) > 0 && string(session.Memory) != "{}" {
		existingMemory = &rtctx.SessionMemory{}
		_ = json.Unmarshal(session.Memory, existingMemory)
	}

	// Fetch all session events from DB
	allEvents, _, err := o.eventRepo.ListBySession(ctx, sessionID, domain.EventListParams{})
	if err != nil {
		return nil, fmt.Errorf("fetch events: %w", err)
	}

	// Backend-side compression: compress the full event history
	compressResult, err := o.compressor.Compress(allEvents, existingMemory)
	if err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("compression failed, sending uncompressed")
		// Fallback: send all events as messages without compression
		messages := rtctx.EventsToMessages(allEvents)
		return &ForwardPayload{
			Memory:   existingMemory,
			Messages: messages,
			NewEvents: toNewEvents(newEvents),
		}, nil
	}

	// Persist updated memory if compression produced one
	if compressResult.Memory != nil && compressResult.Tier >= 2 {
		memJSON, err := json.Marshal(compressResult.Memory)
		if err == nil {
			_ = o.sessionRepo.UpdateMemory(ctx, sessionID, memJSON)
		}
		log.Info().
			Str("session_id", sessionID).
			Int("tier", compressResult.Tier).
			Int("tokens_before", compressResult.TokensBefore).
			Int("tokens_after", compressResult.TokensAfter).
			Msg("backend compressed history")
	}

	// Convert compressed events to Anthropic messages
	messages := rtctx.EventsToMessages(compressResult.Events)

	// Parse model from agent snapshot
	var agentSnap struct {
		Model domain.ModelConfig `json:"model"`
	}
	_ = json.Unmarshal(session.Agent, &agentSnap)

	return &ForwardPayload{
		Memory:            compressResult.Memory,
		Messages:          messages,
		NewEvents:         toNewEvents(newEvents),
		ModelID:           agentSnap.Model.ID,
		ContextWindowSize: contextWindowForModel(agentSnap.Model.ID),
	}, nil
}

func toNewEvents(events []domain.SendEventParams) []NewEvent {
	result := make([]NewEvent, len(events))
	for i, e := range events {
		result[i] = NewEvent{Type: e.Type, Content: e.Content}
	}
	return result
}

// contextWindowForModel returns the context window size for a given model.
func contextWindowForModel(modelID string) int {
	// Default context windows for known models
	switch modelID {
	case "claude-opus-4-6", "claude-sonnet-4-6":
		return 200_000
	case "claude-haiku-4-5":
		return 200_000
	default:
		return 200_000
	}
}

// Teardown stops and removes a sandbox for the given session.
func (o *Orchestrator) Teardown(ctx context.Context, sessionID string) error {
	o.mu.Lock()
	sbx, ok := o.sandboxes[sessionID]
	if ok {
		delete(o.sandboxes, sessionID)
	}
	o.mu.Unlock()

	if !ok || sbx.ContainerID == "" {
		return nil
	}

	log.Info().Str("session_id", sessionID).Str("container", sbx.ContainerID).Msg("tearing down sandbox")
	_ = o.docker.Stop(ctx, sbx.ContainerID)
	return o.docker.Remove(ctx, sbx.ContainerID)
}

func (o *Orchestrator) ensureSandbox(ctx context.Context, sessionID string) (*SandboxInfo, error) {
	o.mu.Lock()
	if sbx, ok := o.sandboxes[sessionID]; ok {
		o.mu.Unlock()
		return sbx, nil
	}
	o.mu.Unlock()

	return o.provision(ctx, sessionID)
}

func (o *Orchestrator) provision(ctx context.Context, sessionID string) (*SandboxInfo, error) {
	log.Info().Str("session_id", sessionID).Msg("provisioning sandbox")

	// Load session
	session, err := o.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	// Parse agent snapshot
	var agentSnap struct {
		ID         string            `json:"id"`
		Version    int               `json:"version"`
		Name       string            `json:"name"`
		System     string            `json:"system"`
		Model      domain.ModelConfig `json:"model"`
		Tools      json.RawMessage   `json:"tools"`
		MCPServers json.RawMessage   `json:"mcp_servers"`
		Skills     []string          `json:"skills"`
	}
	if err := json.Unmarshal(session.Agent, &agentSnap); err != nil {
		return nil, fmt.Errorf("parse agent snapshot: %w", err)
	}

	// Load environment
	env, err := o.envRepo.GetByID(ctx, session.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("load environment: %w", err)
	}

	// Resolve skills
	var skills []SkillManifest
	for _, skillName := range agentSnap.Skills {
		skill, err := o.skillRepo.GetByName(ctx, session.WorkspaceID, skillName)
		if err != nil {
			log.Warn().Str("skill", skillName).Msg("skill not found, skipping")
			continue
		}
		files := map[string]string{}
		if len(skill.Files) > 0 && string(skill.Files) != "{}" {
			_ = json.Unmarshal(skill.Files, &files)
		}
		skills = append(skills, SkillManifest{
			Name:        skill.Name,
			Description: skill.Description,
			License:     skill.License,
			Content:     skill.Content,
			Files:       files,
		})
	}

	// Generate token
	token := o.tokenGen.Generate(sessionID)

	// Build deploy manifest
	manifest := DeployManifest{
		Agent: AgentConfigFile{
			ID:                agentSnap.ID,
			Version:           agentSnap.Version,
			Name:              agentSnap.Name,
			SessionID:         sessionID,
			ControlPlaneURL:   o.config.ControlPlaneURL,
			ControlPlaneToken: token,
			AnthropicAPIKey:   o.config.AnthropicAPIKey,
		},
		SystemPrompt: agentSnap.System,
		Model:        agentSnap.Model,
		Tools:        agentSnap.Tools,
		MCPServers:   agentSnap.MCPServers,
		Skills:       skills,
		Packages:     env.Config.Packages,
	}

	// 1. Create container (not started yet)
	containerName := fmt.Sprintf("sandbox-%s", sessionID)
	containerID, err := o.docker.Create(ctx, CreateOpts{
		Image:   o.config.RuntimeImage,
		Name:    containerName,
		Network: o.config.NetworkName,
		WorkDir: "/workspace",
		Env: []string{
			"WORKSPACE_PATH=/workspace",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	// 2. Deploy config files BEFORE starting (runtime reads config on boot)
	configFiles := o.buildConfigTar(manifest)
	if err := o.docker.CopyToContainer(ctx, containerID, "/workspace", configFiles); err != nil {
		_ = o.docker.Remove(ctx, containerID)
		return nil, fmt.Errorf("deploy config: %w", err)
	}

	// 3. Start container (runtime can now read config immediately)
	info, err := o.docker.Start(ctx, containerID)
	if err != nil {
		_ = o.docker.Remove(ctx, containerID)
		return nil, fmt.Errorf("start container: %w", err)
	}

	// 4. Wait for runtime health
	runtimeURL := fmt.Sprintf("http://%s:%d", info.IP, runtimePort)
	if err := o.waitForHealth(ctx, runtimeURL); err != nil {
		_ = o.docker.Remove(ctx, containerID)
		return nil, fmt.Errorf("runtime not healthy: %w", err)
	}

	sbx := &SandboxInfo{
		ID:          containerName,
		SessionID:   sessionID,
		ContainerID: containerID,
		ContainerIP: info.IP,
		Status:      SandboxStatusRunning,
		CreatedAt:   time.Now().UTC(),
	}

	o.mu.Lock()
	o.sandboxes[sessionID] = sbx
	o.mu.Unlock()

	log.Info().
		Str("session_id", sessionID).
		Str("container_id", info.ID).
		Str("container_ip", info.IP).
		Msg("sandbox provisioned")

	return sbx, nil
}

func (o *Orchestrator) buildConfigTar(manifest DeployManifest) map[string][]byte {
	files := map[string][]byte{}

	// CLAUDE.md
	files["CLAUDE.md"] = []byte(manifest.SystemPrompt)

	// agent.json
	agentJSON, _ := json.MarshalIndent(manifest.Agent, "", "  ")
	files[".claude/agent.json"] = agentJSON

	// settings.json
	settings := SettingsFile{Model: manifest.Model, MCPServers: manifest.MCPServers}
	settingsJSON, _ := json.MarshalIndent(settings, "", "  ")
	files[".claude/settings.json"] = settingsJSON

	// tools.json
	tools := manifest.Tools
	if tools == nil {
		tools = json.RawMessage("[]")
	}
	files[".claude/tools.json"] = tools

	// packages.json
	pkgs := PackagesFile{
		Apt: manifest.Packages.Apt, Cargo: manifest.Packages.Cargo,
		Gem: manifest.Packages.Gem, Go: manifest.Packages.Go,
		Npm: manifest.Packages.Npm, Pip: manifest.Packages.Pip,
	}
	pkgsJSON, _ := json.MarshalIndent(pkgs, "", "  ")
	files[".claude/packages.json"] = pkgsJSON

	// session.json
	state := SessionStateFile{Memory: manifest.SessionMemory, Status: "idle"}
	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	files[".claude/session.json"] = stateJSON

	// Skills
	for _, skill := range manifest.Skills {
		skillMD := reconstructSkillMD(skill)
		files[fmt.Sprintf(".claude/skills/%s/SKILL.md", skill.Name)] = []byte(skillMD)
		for relPath, content := range skill.Files {
			files[fmt.Sprintf(".claude/skills/%s/%s", skill.Name, relPath)] = []byte(content)
		}
	}

	return files
}

func (o *Orchestrator) waitForHealth(ctx context.Context, runtimeURL string) error {
	healthURL := runtimeURL + "/health"
	client := &http.Client{Timeout: 2 * time.Second}

	deadline := time.Now().Add(healthPollTimeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(healthPollInterval):
		}
	}

	return fmt.Errorf("runtime did not become healthy within %s", healthPollTimeout)
}

func (o *Orchestrator) forwardPayload(ctx context.Context, sbx *SandboxInfo, payload *ForwardPayload) error {
	url := fmt.Sprintf("http://%s:%d/v1/events", sbx.ContainerIP, runtimePort)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("forward payload: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("runtime returned %d", resp.StatusCode)
	}

	return nil
}

func (o *Orchestrator) emitEvent(ctx context.Context, sessionID, eventType string, payload interface{}) {
	var payloadJSON json.RawMessage
	if payload != nil {
		payloadJSON, _ = json.Marshal(payload)
	} else {
		payloadJSON = json.RawMessage("{}")
	}

	evt := &domain.Event{
		ID:          domain.NewEventID(),
		SessionID:   sessionID,
		Type:        eventType,
		ProcessedAt: time.Now().UTC(),
		Payload:     payloadJSON,
	}

	_ = o.eventRepo.Create(ctx, evt)
	_ = o.eventBus.Publish(ctx, sessionID, evt)
}
