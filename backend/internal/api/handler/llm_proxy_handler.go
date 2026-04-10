package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/c-cf/macada/internal/sandbox"
	"github.com/go-chi/chi/v5"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

// LLMProxyHandler proxies LLM requests from sandbox runtimes to the Anthropic API.
// This keeps the API key on the control plane — containers never see it.
type LLMProxyHandler struct {
	anthropicKey string
	tokenGen     *sandbox.TokenGenerator
	client       *http.Client
}

// NewLLMProxyHandler creates a new LLM proxy handler.
func NewLLMProxyHandler(
	anthropicKey string,
	tokenGen *sandbox.TokenGenerator,
) *LLMProxyHandler {
	return &LLMProxyHandler{
		anthropicKey: anthropicKey,
		tokenGen:     tokenGen,
		client:       &http.Client{},
	}
}

// ProxyLLM receives an Anthropic Messages API request from a sandbox runtime,
// injects the real API key, forwards to Anthropic, and pipes the response back.
//
// The request body is forwarded as-is (no parsing). The response is streamed back
// transparently, supporting both streaming and non-streaming modes.
func (h *LLMProxyHandler) ProxyLLM(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session_id")

	// Validate sandbox token
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || !h.tokenGen.Validate(sessionID, token) {
		writeError(w, http.StatusUnauthorized, "invalid sandbox token")
		return
	}

	// Build upstream request to Anthropic
	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, anthropicAPIURL, r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create upstream request")
		return
	}

	// Copy relevant headers from runtime request
	upstream.Header.Set("Content-Type", "application/json")
	upstream.Header.Set("anthropic-version", r.Header.Get("anthropic-version"))

	// Inject the real API key (never sent to container)
	upstream.Header.Set("x-api-key", h.anthropicKey)

	// Forward any beta headers
	if beta := r.Header.Get("anthropic-beta"); beta != "" {
		upstream.Header.Set("anthropic-beta", beta)
	}

	// Execute upstream request
	resp, err := h.client.Do(upstream)
	if err != nil {
		writeError(w, http.StatusBadGateway, "upstream request failed")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	// Copy response headers back
	for _, key := range []string{"Content-Type", "X-Request-Id", "Request-Id"} {
		if v := resp.Header.Get(key); v != "" {
			w.Header().Set(key, v)
		}
	}

	// Stream the response back transparently
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
