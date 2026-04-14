package loop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	rtcfg "github.com/c-cf/macada/internal/runtime/config"
	rtctx "github.com/c-cf/macada/internal/runtime/context"
	"github.com/c-cf/macada/internal/runtime/prompt"
	"github.com/c-cf/macada/internal/runtime/reporter"
	"github.com/c-cf/macada/internal/runtime/toolset"
	"github.com/rs/zerolog/log"
)

const (
	maxToolRounds  = 50
	defaultMaxToks = 8192
	// Mid-turn compression triggers when messages exceed this fraction of context window
	midTurnCompressRatio = 0.75
)

// RunInput is the pre-compressed input from the control plane.
type RunInput struct {
	// Memory is the compressed summary of older conversation (from backend compression).
	Memory *rtctx.SessionMemory
	// History is recent conversation turns already in message format (from backend).
	History []rtctx.Message
	// NewMessages are the new user messages that triggered this turn.
	NewMessages []rtctx.Message
	// ContextWindowSize is the model's context window in tokens.
	ContextWindowSize int
}

// Loop implements the agent loop: prompt → API → tools → repeat.
type Loop struct {
	config     rtcfg.RuntimeConfig
	api        *AnthropicClient
	tools      *ToolExecutor
	toolset    *toolset.Toolset // nil if agent type doesn't specify a toolset
	reporter   *reporter.Reporter
	compressor *rtctx.Compressor
}

// NewLoop creates a new agent loop.
func NewLoop(
	config rtcfg.RuntimeConfig,
	api *AnthropicClient,
	tools *ToolExecutor,
	ts *toolset.Toolset,
	reporter *reporter.Reporter,
	compressor *rtctx.Compressor,
) *Loop {
	return &Loop{
		config:     config,
		api:        api,
		tools:      tools,
		toolset:    ts,
		reporter:   reporter,
		compressor: compressor,
	}
}

// Run executes one turn with just user messages (no pre-compressed context).
// Used when the runtime is bootstrapping without backend forwarding.
func (l *Loop) Run(ctx context.Context, userMessages []rtctx.Message) error {
	return l.RunWithInput(ctx, RunInput{
		NewMessages:       userMessages,
		ContextWindowSize: 200_000,
	})
}

// RunWithInput executes one turn with pre-compressed context from the backend.
//
// The two-layer compression model:
//   - Backend (between turns): compresses full event history → Memory + recent Messages
//   - Runtime (within turn): if tool chains push messages near context limit → microcompact
func (l *Loop) RunWithInput(ctx context.Context, input RunInput) error {
	// Build system prompt with memory from backend
	systemBlocks, err := l.buildSystemPromptWithMemory(ctx, input.Memory)
	if err != nil {
		l.reportError(ctx, "unknown_error", err.Error(), "exhausted")
		return fmt.Errorf("build system prompt: %w", err)
	}
	apiSystem := toAPISystemBlocks(systemBlocks)

	// Assemble conversation: backend history + new user messages
	messages := toAPIMessages(input.History)
	messages = append(messages, toAPIMessages(input.NewMessages)...)

	contextLimit := input.ContextWindowSize
	if contextLimit == 0 {
		contextLimit = 200_000
	}
	midTurnThreshold := int(float64(contextLimit) * midTurnCompressRatio)

	_ = l.reporter.Report(ctx, "session.status_running", nil)

	// Merge toolset definitions with user-provided tools
	apiTools := l.config.Tools
	if l.toolset != nil {
		apiTools = l.toolset.MergeDefinitions(l.config.Tools)
	}

	for round := 0; round < maxToolRounds; round++ {
		// Mid-turn compression: if messages are getting too large, thin old tool results
		messages = l.midTurnCompress(messages, midTurnThreshold)

		startTime := time.Now()
		_ = l.reporter.Report(ctx, "span.model_request_start", nil)

		resp, err := l.api.CreateMessage(ctx, MessageRequest{
			Model:     l.config.Settings.Model.ID,
			System:    apiSystem,
			Messages:  messages,
			Tools:     apiTools,
			MaxTokens: defaultMaxToks,
		})
		if err != nil {
			_ = l.reporter.Report(ctx, "span.model_request_end", map[string]interface{}{
				"is_error": true,
				"error":    err.Error(),
			})
			l.reportAPIError(ctx, err)
			return fmt.Errorf("api call: %w", err)
		}

		_ = l.reporter.Report(ctx, "span.model_request_end", map[string]interface{}{
			"is_error":    false,
			"latency_ms":  time.Since(startTime).Milliseconds(),
			"model":       l.config.Settings.Model.ID,
			"model_usage": resp.Usage,
		})

		// Process response
		var toolUses []ContentBlock
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				_ = l.reporter.Report(ctx, "agent.message", map[string]interface{}{
					"content": []map[string]string{{"type": "text", "text": block.Text}},
				})
			case "tool_use":
				toolUses = append(toolUses, block)
				_ = l.reporter.Report(ctx, "agent.tool_use", map[string]interface{}{
					"id":    block.ID,
					"name":  block.Name,
					"input": block.Input,
				})
			}
		}

		// Append assistant response
		assistantContent, _ := json.Marshal(resp.Content)
		messages = append(messages, Message{Role: "assistant", Content: assistantContent})

		if resp.StopReason == "end_turn" || len(toolUses) == 0 {
			_ = l.reporter.Report(ctx, "session.status_idle", map[string]interface{}{
				"stop_reason": map[string]string{"type": resp.StopReason},
			})
			return nil
		}

		// Execute tools (toolset first, then legacy executor)
		var toolResults []map[string]interface{}
		for _, tu := range toolUses {
			result := l.executeTool(ctx, tu.Name, tu.Input)
			_ = l.reporter.Report(ctx, "agent.tool_result", map[string]interface{}{
				"tool_use_id": tu.ID,
				"content":     result.Content,
				"is_error":    result.IsError,
			})
			toolResults = append(toolResults, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": tu.ID,
				"content":     result.Content,
				"is_error":    result.IsError,
			})
		}

		resultsJSON, _ := json.Marshal(toolResults)
		messages = append(messages, Message{Role: "user", Content: resultsJSON})
	}

	_ = l.reporter.Report(ctx, "session.status_idle", map[string]interface{}{
		"stop_reason": map[string]string{"type": "max_tool_rounds"},
	})
	return fmt.Errorf("reached max tool rounds (%d)", maxToolRounds)
}

// midTurnCompress applies microcompact if messages exceed the token threshold.
// It replaces old tool_result content with "[cleared]" to free up context space.
func (l *Loop) midTurnCompress(messages []Message, threshold int) []Message {
	estimated := estimateAPIMessagesTokens(messages)
	if estimated <= threshold {
		return messages
	}

	log.Info().
		Int("estimated_tokens", estimated).
		Int("threshold", threshold).
		Msg("mid-turn compression triggered")

	// Find tool_result messages and thin old ones (keep last 3)
	keepRecent := 3
	var toolResultIndices []int
	for i, msg := range messages {
		if msg.Role == "user" {
			var blocks []struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(msg.Content, &blocks) == nil {
				for _, b := range blocks {
					if b.Type == "tool_result" {
						toolResultIndices = append(toolResultIndices, i)
						break
					}
				}
			}
		}
	}

	if len(toolResultIndices) <= keepRecent {
		return messages
	}

	thinSet := map[int]bool{}
	for _, idx := range toolResultIndices[:len(toolResultIndices)-keepRecent] {
		thinSet[idx] = true
	}

	result := make([]Message, len(messages))
	for i, msg := range messages {
		if thinSet[i] {
			result[i] = thinToolResultMessage(msg)
		} else {
			result[i] = msg
		}
	}

	return result
}

// thinToolResultMessage replaces tool_result content with a cleared marker.
func thinToolResultMessage(msg Message) Message {
	var blocks []map[string]interface{}
	if json.Unmarshal(msg.Content, &blocks) != nil {
		return msg
	}

	for i, block := range blocks {
		if t, _ := block["type"].(string); t == "tool_result" {
			blocks[i]["content"] = "[content cleared]"
		}
	}

	content, _ := json.Marshal(blocks)
	return Message{Role: msg.Role, Content: content}
}

// executeTool routes to the toolset if available, otherwise falls back to the legacy executor.
func (l *Loop) executeTool(ctx context.Context, name string, input json.RawMessage) toolset.ToolResult {
	if l.toolset != nil && l.toolset.CanExecute(name) {
		return l.toolset.Execute(ctx, name, input)
	}
	return l.tools.Execute(ctx, name, input)
}

func (l *Loop) buildSystemPromptWithMemory(ctx context.Context, memory *rtctx.SessionMemory) ([]prompt.SystemBlock, error) {
	var skills []prompt.ResolvedSkill
	for _, s := range l.config.Skills {
		skills = append(skills, prompt.ResolvedSkill{Name: s.Name, Content: s.Content})
	}

	// Use memory from backend (pre-compressed), falling back to config file
	if memory == nil && l.config.Session.Memory != nil {
		memory = l.config.Session.Memory
	}

	input := prompt.PromptInput{
		Agent: prompt.AgentSnapshot{
			ID:      l.config.Agent.ID,
			Version: l.config.Agent.Version,
			Name:    l.config.Agent.Name,
			System:  l.config.SystemPrompt,
			Tools:   l.config.Tools,
		},
		ResolvedSkills: skills,
		SessionMemory:  memory,
	}

	builder := prompt.NewBuilder(nil)
	return builder.Build(ctx, input)
}

// estimateAPIMessagesTokens is a rough estimate of API message token count.
func estimateAPIMessagesTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += 4 + len(m.Content)/4 // ~4 chars per token + overhead
	}
	return total
}

func toAPISystemBlocks(blocks []prompt.SystemBlock) []SystemBlock {
	result := make([]SystemBlock, len(blocks))
	for i, b := range blocks {
		sb := SystemBlock{Type: b.Type, Text: b.Text}
		if b.CacheControl != nil {
			sb.CacheControl = &CacheControl{Type: b.CacheControl.Type}
		}
		result[i] = sb
	}
	return result
}

func toAPIMessages(msgs []rtctx.Message) []Message {
	result := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		content, _ := json.Marshal(m.Content)
		result = append(result, Message{Role: m.Role, Content: content})
	}
	return result
}

// reportAPIError classifies an API error and reports session.error + session.status_idle.
func (l *Loop) reportAPIError(ctx context.Context, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		errorType := classifyAPIError(apiErr.StatusCode)
		l.reportError(ctx, errorType, apiErr.Message, "exhausted")
		return
	}
	l.reportError(ctx, "unknown_error", err.Error(), "exhausted")
}

// reportError sends a session.error event followed by session.status_idle with retries_exhausted.
func (l *Loop) reportError(ctx context.Context, errorType, message, retryStatus string) {
	_ = l.reporter.Report(ctx, "session.error", map[string]interface{}{
		"error": map[string]interface{}{
			"type":    errorType,
			"message": message,
			"retry_status": map[string]string{
				"type": retryStatus,
			},
		},
	})

	stopReason := "retries_exhausted"
	if retryStatus == "terminal" {
		stopReason = "terminal"
	}
	_ = l.reporter.Report(ctx, "session.status_idle", map[string]interface{}{
		"stop_reason": map[string]string{"type": stopReason},
	})
}

// classifyAPIError maps HTTP status codes to API spec error types.
func classifyAPIError(statusCode int) string {
	switch statusCode {
	case 429:
		return "model_rate_limited_error"
	case 529:
		return "model_overloaded_error"
	default:
		return "model_request_failed_error"
	}
}
