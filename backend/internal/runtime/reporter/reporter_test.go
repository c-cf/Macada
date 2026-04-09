package reporter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestReporter_ReportSuccess(t *testing.T) {
	var received int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&received, 1)

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing auth header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type")
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rep := &Reporter{
		controlPlaneURL: srv.URL,
		sessionID:       "sesn_01",
		token:           "test-token",
		client:          &http.Client{Timeout: 5 * time.Second},
		cancel:          func() {},
	}

	err := rep.Report(context.Background(), "agent.message", map[string]string{"text": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("expected 1 request, got %d", received)
	}
}

func TestReporter_RetryOnServerError(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rep := &Reporter{
		controlPlaneURL: srv.URL,
		sessionID:       "sesn_01",
		token:           "tok",
		client:          &http.Client{Timeout: 5 * time.Second},
		cancel:          func() {},
	}

	err := rep.Report(context.Background(), "test.event", nil)
	if err != nil {
		t.Fatalf("should succeed after retry: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestReporter_NoRetryOnClientError(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	rep := &Reporter{
		controlPlaneURL: srv.URL,
		sessionID:       "sesn_01",
		token:           "tok",
		client:          &http.Client{Timeout: 5 * time.Second},
		cancel:          func() {},
	}

	err := rep.Report(context.Background(), "test.event", nil)
	if err == nil {
		t.Fatal("should fail on 400")
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("should not retry on 400, got %d attempts", attempts)
	}
}
