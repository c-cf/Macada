package sandbox

import (
	"strings"
	"testing"
)

func TestTokenGenerator_GenerateAndValidate(t *testing.T) {
	gen := NewTokenGenerator("test-secret")

	token := gen.Generate("sesn_01ABC")
	if !strings.HasPrefix(token, "sandbox_") {
		t.Errorf("token should start with sandbox_, got %q", token)
	}

	if !gen.Validate("sesn_01ABC", token) {
		t.Error("token should be valid")
	}
}

func TestTokenGenerator_InvalidToken(t *testing.T) {
	gen := NewTokenGenerator("test-secret")
	if gen.Validate("sesn_01ABC", "sandbox_invalid") {
		t.Error("invalid token should not validate")
	}
}

func TestTokenGenerator_WrongSession(t *testing.T) {
	gen := NewTokenGenerator("test-secret")
	token := gen.Generate("sesn_01ABC")
	if gen.Validate("sesn_OTHER", token) {
		t.Error("token for different session should not validate")
	}
}

func TestTokenGenerator_RandomSecret(t *testing.T) {
	gen := NewTokenGenerator("")
	token := gen.Generate("sesn_01")
	if !gen.Validate("sesn_01", token) {
		t.Error("token with random secret should still validate")
	}
}
