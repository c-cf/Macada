package pagination

import (
	"encoding/base64"
	"fmt"
	"time"
)

// EncodeCursor encodes a (created_at, id) pair into an opaque pagination cursor.
func EncodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%s|%s", createdAt.Format(time.RFC3339Nano), id)
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor back into (created_at, id).
func DecodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor: %w", err)
	}

	parts := splitFirst(string(raw), '|')
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor time: %w", err)
	}

	return t, parts[1], nil
}

func splitFirst(s string, sep byte) []string {
	for i := range len(s) {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
