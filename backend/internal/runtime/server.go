package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	rtctx "github.com/c-cf/macada/internal/runtime/context"
	"github.com/c-cf/macada/internal/runtime/loop"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Server is the HTTP server running inside the sandbox.
// It receives ForwardPayload from the control plane and feeds it to the agent loop.
type Server struct {
	agentLoop *loop.Loop
	mu        sync.Mutex
	running   bool
}

// NewServer creates a new runtime HTTP server.
func NewServer(agentLoop *loop.Loop) *Server {
	return &Server{agentLoop: agentLoop}
}

// Handler returns the HTTP handler for the runtime server.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", s.Health)
	r.Post("/v1/events", s.HandleEvents)
	r.Post("/v1/stop", s.Stop)

	return r
}

// Health returns 200 OK with the runtime status.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()

	status := "idle"
	if running {
		status = "running"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
}

// forwardPayload mirrors sandbox.ForwardPayload (avoid import cycle).
type forwardPayload struct {
	Memory            *rtctx.SessionMemory `json:"memory,omitempty"`
	Messages          []rtctx.Message      `json:"messages"`
	NewEvents         []newEvent           `json:"new_events"`
	ModelID           string               `json:"model_id"`
	ContextWindowSize int                  `json:"context_window_size"`
}

type newEvent struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

// HandleEvents receives a ForwardPayload from the control plane.
// The payload contains pre-compressed history (memory + messages) and new user events.
func (s *Server) HandleEvents(w http.ResponseWriter, r *http.Request) {
	var payload forwardPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Convert new events to user messages
	newMessages := extractUserMessages(payload.NewEvents)
	if len(newMessages) == 0 && len(payload.Messages) == 0 {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "no_messages"})
		return
	}

	// Build the RunInput for the loop
	input := loop.RunInput{
		Memory:            payload.Memory,
		History:           payload.Messages,
		NewMessages:       newMessages,
		ContextWindowSize: payload.ContextWindowSize,
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		http.Error(w, "agent is already running", http.StatusConflict)
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}()

		if err := s.agentLoop.RunWithInput(context.Background(), input); err != nil {
			log.Error().Err(err).Msg("agent loop error")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// Stop triggers a graceful shutdown.
func (s *Server) Stop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopping"})
}

func extractUserMessages(events []newEvent) []rtctx.Message {
	var messages []rtctx.Message
	for _, evt := range events {
		if evt.Type != "user.message" {
			continue
		}
		text := extractText(evt.Content)
		if text != "" {
			messages = append(messages, rtctx.Message{
				Role:    "user",
				Content: []rtctx.ContentBlock{{Type: "text", Text: text}},
			})
		}
	}
	return messages
}

func extractText(content json.RawMessage) string {
	if content == nil {
		return ""
	}
	// Try as array of blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) == nil {
		for _, b := range blocks {
			if b.Type == "text" {
				return b.Text
			}
		}
	}
	// Try as plain string
	var s string
	if json.Unmarshal(content, &s) == nil {
		return s
	}
	return ""
}
