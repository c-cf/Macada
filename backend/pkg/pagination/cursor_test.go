package pagination_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/c-cf/macada/pkg/pagination"
)

func TestCursorRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		createdAt time.Time
		id        string
	}{
		{
			name:      "basic timestamp and ID",
			createdAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			id:        "agent_01HQXYZ123456789ABCDEF",
		},
		{
			name:      "nanosecond precision preserved",
			createdAt: time.Date(2025, 6, 1, 12, 0, 0, 123456789, time.UTC),
			id:        "sesn_01HQXYZ987654321FEDCBA",
		},
		{
			name:      "empty ID",
			createdAt: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			id:        "",
		},
		{
			name:      "ID with pipe character",
			createdAt: time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC),
			id:        "env_some|id|with|pipes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cursor := pagination.EncodeCursor(tc.createdAt, tc.id)

			gotTime, gotID, err := pagination.DecodeCursor(cursor)
			if err != nil {
				t.Fatalf("DecodeCursor returned error: %v", err)
			}

			if !gotTime.Equal(tc.createdAt) {
				t.Errorf("time mismatch: want %v, got %v", tc.createdAt, gotTime)
			}
			if gotID != tc.id {
				t.Errorf("id mismatch: want %q, got %q", tc.id, gotID)
			}
		})
	}
}

func TestDecodeCursorErrors(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{
			name:   "not base64",
			cursor: "!!!invalid-base64!!!",
		},
		{
			name:   "no pipe separator",
			cursor: base64.URLEncoding.EncodeToString([]byte("nopipehere")),
		},
		{
			name:   "invalid timestamp",
			cursor: base64.URLEncoding.EncodeToString([]byte("not-a-time|some_id")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := pagination.DecodeCursor(tc.cursor)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestEncodeCursorDeterministic(t *testing.T) {
	ts := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	id := "agent_ABC123"

	c1 := pagination.EncodeCursor(ts, id)
	c2 := pagination.EncodeCursor(ts, id)

	if c1 != c2 {
		t.Errorf("same inputs produced different cursors: %q vs %q", c1, c2)
	}
}
