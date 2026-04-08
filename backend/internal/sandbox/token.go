package sandbox

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// TokenGenerator generates and validates sandbox auth tokens.
type TokenGenerator struct {
	secret []byte
}

// NewTokenGenerator creates a generator with the given secret.
// If secret is empty, a random 32-byte secret is generated.
func NewTokenGenerator(secret string) *TokenGenerator {
	var key []byte
	if secret != "" {
		key = []byte(secret)
	} else {
		key = make([]byte, 32)
		rand.Read(key)
	}
	return &TokenGenerator{secret: key}
}

// Generate creates an HMAC-SHA256 token for the given session ID.
func (g *TokenGenerator) Generate(sessionID string) string {
	mac := hmac.New(sha256.New, g.secret)
	mac.Write([]byte(sessionID))
	return fmt.Sprintf("sandbox_%s", hex.EncodeToString(mac.Sum(nil)))
}

// Validate checks if a token is valid for the given session ID.
func (g *TokenGenerator) Validate(sessionID, token string) bool {
	expected := g.Generate(sessionID)
	return hmac.Equal([]byte(expected), []byte(token))
}
