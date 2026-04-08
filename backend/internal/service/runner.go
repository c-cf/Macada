package service

import (
	"context"
	"encoding/json"
	"time"

	svcctx "github.com/cchu-code/managed-agents/internal/runtime/context"
	"github.com/cchu-code/managed-agents/internal/runtime/prompt"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/internal/infra/postgres"
	"github.com/rs/zerolog/log"
)

// Runner implements the agent loop that processes user messages.
// It composes the system prompt and compresses context before calling the model.
type Runner struct {
	eventRepo     domain.EventRepository
	sessionRepo   domain.SessionRepository
	eventBus      domain.EventBus
	analyticsRepo *postgres.AnalyticsRepo
	promptBuilder *prompt.Builder
	compressor    *svcctx.Compressor
}

func NewRunner(
	eventRepo domain.EventRepository,
	sessionRepo domain.SessionRepository,
	eventBus domain.EventBus,
	analyticsRepo *postgres.AnalyticsRepo,
	promptBuilder *prompt.Builder,
	compressor *svcctx.Compressor,
) *Runner {
	return &Runner{
		eventRepo:     eventRepo,
		sessionRepo:   sessionRepo,
		eventBus:      eventBus,
		analyticsRepo: analyticsRepo,
		promptBuilder: promptBuilder,
		compressor:    compressor,
	}
}

func (r *Runner) Run(ctx context.Context, sessionID string, events []domain.SendEventParams) error {
	bgCtx := context.Background()

	// 1. Set session to running
	if err := r.sessionRepo.UpdateStatus(bgCtx, sessionID, domain.SessionStatusRunning); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to set session running")
		return err
	}
	r.emitEvent(bgCtx, sessionID, domain.EventTypeSessionRunning, nil)

	// 2. Build context: fetch session, events, compress, build prompt
	systemBlocks, compressResult := r.prepareContext(bgCtx, sessionID)

	// Persist updated memory if compression produced one
	if compressResult != nil && compressResult.Memory != nil {
		memJSON, err := json.Marshal(compressResult.Memory)
		if err == nil {
			r.sessionRepo.UpdateMemory(bgCtx, sessionID, memJSON)
		}
	}

	// Log prompt preparation results
	if compressResult != nil && compressResult.Tier > 0 {
		log.Info().
			Str("session_id", sessionID).
			Int("tier", compressResult.Tier).
			Int("tokens_before", compressResult.TokensBefore).
			Int("tokens_after", compressResult.TokensAfter).
			Msg("context compressed")
	}
	if len(systemBlocks) > 0 {
		log.Debug().
			Str("session_id", sessionID).
			Int("system_blocks", len(systemBlocks)).
			Msg("system prompt built")
	}

	// 3. Emit span.model_request_start
	startEvt := r.emitEvent(bgCtx, sessionID, domain.EventTypeModelRequestStart, nil)

	// 4. Demo response (production: call Anthropic Messages API with systemBlocks + messages)
	responseText := "Hello! I'm your managed agent. I received your message and I'm ready to help. This is a demo response — in production, I would be powered by Claude and have access to tools in your environment."

	r.emitEvent(bgCtx, sessionID, domain.EventTypeAgentMessage, map[string]interface{}{
		"content": []map[string]string{
			{"type": "text", "text": responseText},
		},
	})

	// 5. Emit span.model_request_end with usage
	modelInputTokens := int64(100)
	modelOutputTokens := int64(50)
	modelCacheRead := int64(0)
	modelCacheCreation := int64(0)

	r.emitEvent(bgCtx, sessionID, domain.EventTypeModelRequestEnd, map[string]interface{}{
		"model_request_start_id": startEvt.ID,
		"is_error":               false,
		"model_usage": domain.ModelUsage{
			InputTokens:              modelInputTokens,
			OutputTokens:             modelOutputTokens,
			CacheCreationInputTokens: modelCacheCreation,
			CacheReadInputTokens:     modelCacheRead,
		},
	})

	// 5b. Record analytics
	now := time.Now().UTC()
	agentID := r.extractAgentID(bgCtx, sessionID)

	wsID := r.extractWorkspaceID(bgCtx, sessionID)
	logEntry := postgres.LogRow{
		ID:                  domain.NewLLMLogID(),
		WorkspaceID:         wsID,
		SessionID:           sessionID,
		AgentID:             agentID,
		Model:               "claude-sonnet-4-6",
		InputTokens:         modelInputTokens,
		OutputTokens:        modelOutputTokens,
		CacheReadTokens:     modelCacheRead,
		CacheCreationTokens: modelCacheCreation,
		LatencyMs:           0,
		IsError:             false,
		CreatedAt:           now,
	}

	if err := r.analyticsRepo.InsertRequestLog(bgCtx, logEntry); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to insert LLM request log")
	}
	if err := r.analyticsRepo.IncrementDailyUsage(bgCtx, wsID, now, logEntry.Model, modelInputTokens, modelOutputTokens, modelCacheRead, modelCacheCreation); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to increment daily usage")
	}

	// 6. Emit session.status_idle
	r.emitEvent(bgCtx, sessionID, domain.EventTypeSessionIdle, map[string]interface{}{
		"stop_reason": map[string]string{"type": "end_turn"},
	})

	// 7. Update session status
	if err := r.sessionRepo.UpdateStatus(bgCtx, sessionID, domain.SessionStatusIdle); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to set session idle")
		return err
	}

	// 8. Update session usage
	inputTokens := int64(100)
	outputTokens := int64(50)
	r.sessionRepo.UpdateUsage(bgCtx, sessionID, domain.SessionUsage{
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
	})

	return nil
}

// prepareContext fetches session data, compresses events, and builds the system prompt.
func (r *Runner) prepareContext(ctx context.Context, sessionID string) ([]prompt.SystemBlock, *svcctx.CompressResult) {
	session, err := r.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to fetch session for context")
		return nil, nil
	}

	// Parse agent snapshot
	var agent prompt.AgentSnapshot
	if err := json.Unmarshal(session.Agent, &agent); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to parse agent snapshot")
		return nil, nil
	}

	// Load existing session memory
	var existingMemory *svcctx.SessionMemory
	if len(session.Memory) > 0 && string(session.Memory) != "{}" {
		existingMemory = &svcctx.SessionMemory{}
		json.Unmarshal(session.Memory, existingMemory)
	}

	// Fetch all session events
	allEvents, _, err := r.eventRepo.ListBySession(ctx, sessionID, domain.EventListParams{})
	if err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to fetch events for context")
		return nil, nil
	}

	// Compress context
	var compressResult *svcctx.CompressResult
	if r.compressor != nil {
		compressResult, err = r.compressor.Compress(allEvents, existingMemory)
		if err != nil {
			log.Error().Err(err).Str("session_id", sessionID).Msg("compression failed")
		}
	}

	// Determine session memory for prompt
	var sessionMemory *svcctx.SessionMemory
	if compressResult != nil && compressResult.Memory != nil {
		sessionMemory = compressResult.Memory
	} else {
		sessionMemory = existingMemory
	}

	// Build system prompt
	var systemBlocks []prompt.SystemBlock
	if r.promptBuilder != nil {
		input := prompt.PromptInput{
			Agent:         agent,
			WorkspaceID:   session.WorkspaceID,
			SessionMemory: sessionMemory,
		}
		systemBlocks, err = r.promptBuilder.Build(ctx, input)
		if err != nil {
			log.Error().Err(err).Str("session_id", sessionID).Msg("failed to build system prompt")
		}
	}

	return systemBlocks, compressResult
}

func (r *Runner) extractWorkspaceID(ctx context.Context, sessionID string) string {
	session, err := r.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return ""
	}
	return session.WorkspaceID
}

func (r *Runner) extractAgentID(ctx context.Context, sessionID string) string {
	session, err := r.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return "unknown"
	}
	var agentSnapshot struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(session.Agent, &agentSnapshot); err != nil {
		return "unknown"
	}
	return agentSnapshot.ID
}

func (r *Runner) emitEvent(ctx context.Context, sessionID, eventType string, payload interface{}) *domain.Event {
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

	if err := r.eventRepo.Create(ctx, evt); err != nil {
		log.Error().Err(err).Str("type", eventType).Msg("failed to persist event")
	}
	if err := r.eventBus.Publish(ctx, sessionID, evt); err != nil {
		log.Error().Err(err).Str("type", eventType).Msg("failed to publish event")
	}

	return evt
}
