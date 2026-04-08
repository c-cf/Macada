package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	maxRetries       = 3
	initialBackoff   = 500 * time.Millisecond
	requestTimeout   = 10 * time.Second
	heartbeatInterval = 30 * time.Second
)

// Reporter sends events from the agent runtime back to the control plane.
type Reporter struct {
	controlPlaneURL string
	sessionID       string
	token           string
	client          *http.Client

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewReporter creates a new Reporter and starts the heartbeat goroutine.
func NewReporter(controlPlaneURL, sessionID, token string) *Reporter {
	ctx, cancel := context.WithCancel(context.Background())
	r := &Reporter{
		controlPlaneURL: controlPlaneURL,
		sessionID:       sessionID,
		token:           token,
		client:          &http.Client{Timeout: requestTimeout},
		cancel:          cancel,
	}
	r.wg.Add(1)
	go r.heartbeatLoop(ctx)
	return r
}

// Report sends an event to the control plane with retries.
func (r *Reporter) Report(ctx context.Context, eventType string, payload interface{}) error {
	body := map[string]interface{}{
		"events": []map[string]interface{}{
			{
				"type":    eventType,
				"payload": payload,
			},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	url := fmt.Sprintf("%s/internal/v1/sandbox/%s/events", r.controlPlaneURL, r.sessionID)

	var lastErr error
	backoff := initialBackoff
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+r.token)

		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt+1).Str("type", eventType).Msg("report event failed")
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Client error — don't retry
			return lastErr
		}
		log.Warn().Int("status", resp.StatusCode).Int("attempt", attempt+1).Str("type", eventType).Msg("report event server error")
	}

	return fmt.Errorf("report event after %d retries: %w", maxRetries, lastErr)
}

// Close stops the heartbeat and waits for it to finish.
func (r *Reporter) Close() {
	r.cancel()
	r.wg.Wait()
}

func (r *Reporter) heartbeatLoop(ctx context.Context) {
	defer r.wg.Done()
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Report(ctx, "runtime.heartbeat", map[string]string{
				"session_id": r.sessionID,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
}
