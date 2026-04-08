package sse

import (
	"fmt"
	"net/http"
)

// Writer writes Server-Sent Events to an http.ResponseWriter.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewWriter creates an SSE writer. Returns nil if the ResponseWriter does not support flushing.
func NewWriter(w http.ResponseWriter) *Writer {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	return &Writer{w: w, flusher: flusher}
}

// WriteEvent writes a named event with data.
func (s *Writer) WriteEvent(eventType string, data []byte) error {
	if _, err := fmt.Fprintf(s.w, "event: %s\n", eventType); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// WriteData writes a data-only event (no event name).
func (s *Writer) WriteData(data []byte) error {
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}
