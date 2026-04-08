package loop

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	apiURL         = "https://api.anthropic.com/v1/messages"
	apiVersion     = "2023-06-01"
	clientTimeout  = 5 * time.Minute
	maxRetries     = 2
	retryBaseDelay = 1 * time.Second
)

// AnthropicClient calls the Anthropic Messages API.
type AnthropicClient struct {
	apiKey string
	client *http.Client
}

// NewAnthropicClient creates a new client.
func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		apiKey: apiKey,
		client: &http.Client{Timeout: clientTimeout},
	}
}

// SystemBlock is one block in the Anthropic system prompt array.
type SystemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CacheControl instructs the API to cache up to this block.
type CacheControl struct {
	Type string `json:"type"`
}

// MessageRequest is the request body for POST /v1/messages.
type MessageRequest struct {
	Model     string          `json:"model"`
	System    []SystemBlock   `json:"system,omitempty"`
	Messages  []Message       `json:"messages"`
	Tools     json.RawMessage `json:"tools,omitempty"`
	MaxTokens int             `json:"max_tokens"`
}

// Message is a single message in the messages array.
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// MessageResponse is the response from the Messages API.
type MessageResponse struct {
	ID         string         `json:"id"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

// ContentBlock is a block in the response content array.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// Usage contains token usage from the API response.
type Usage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Type       string `json:"type"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("anthropic api %d: %s — %s", e.StatusCode, e.Type, e.Message)
}

// CreateMessage sends a non-streaming request to the Messages API.
func (c *AnthropicClient) CreateMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = 8192
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := c.doRequest(ctx, body)
		if err != nil {
			lastErr = err
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

// CreateMessageStream sends a streaming request and calls onEvent for each SSE event.
func (c *AnthropicClient) CreateMessageStream(ctx context.Context, req MessageRequest, onEvent func(eventType string, data json.RawMessage)) (*MessageResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = 8192
	}

	streamReq := struct {
		MessageRequest
		Stream bool `json:"stream"`
	}{
		MessageRequest: req,
		Stream:         true,
	}

	body, err := json.Marshal(streamReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	return c.parseSSEStream(resp.Body, onEvent)
}

func (c *AnthropicClient) doRequest(ctx context.Context, body []byte) (*MessageResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		apiErr := parseAPIError(resp)
		// Retry on 429 (rate limit) and 529 (overloaded)
		if resp.StatusCode == 429 || resp.StatusCode == 529 {
			return nil, apiErr
		}
		return nil, apiErr
	}

	var result MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *AnthropicClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
}

func (c *AnthropicClient) parseSSEStream(body io.Reader, onEvent func(string, json.RawMessage)) (*MessageResponse, error) {
	scanner := bufio.NewScanner(body)
	var final MessageResponse
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := json.RawMessage(strings.TrimPrefix(line, "data: "))

			if onEvent != nil {
				onEvent(currentEvent, data)
			}

			if currentEvent == "message_stop" {
				// Try to extract final message from accumulated data
				break
			}

			if currentEvent == "message_start" {
				var msg struct {
					Message MessageResponse `json:"message"`
				}
				if json.Unmarshal(data, &msg) == nil {
					final = msg.Message
				}
			}

			if currentEvent == "message_delta" {
				var delta struct {
					Delta struct {
						StopReason string `json:"stop_reason"`
					} `json:"delta"`
					Usage Usage `json:"usage"`
				}
				if json.Unmarshal(data, &delta) == nil {
					final.StopReason = delta.Delta.StopReason
					final.Usage.OutputTokens = delta.Usage.OutputTokens
				}
			}

			if currentEvent == "content_block_start" {
				var block struct {
					ContentBlock ContentBlock `json:"content_block"`
				}
				if json.Unmarshal(data, &block) == nil {
					final.Content = append(final.Content, block.ContentBlock)
				}
			}

			if currentEvent == "content_block_delta" {
				var delta struct {
					Index int `json:"index"`
					Delta struct {
						Type string `json:"type"`
						Text string `json:"text,omitempty"`
					} `json:"delta"`
				}
				if json.Unmarshal(data, &delta) == nil && delta.Index < len(final.Content) {
					if delta.Delta.Type == "text_delta" {
						final.Content[delta.Index].Text += delta.Delta.Text
					}
				}
			}

			currentEvent = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	return &final, nil
}

func parseAPIError(resp *http.Response) *APIError {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &errResp)

	return &APIError{
		StatusCode: resp.StatusCode,
		Type:       errResp.Error.Type,
		Message:    errResp.Error.Message,
	}
}
