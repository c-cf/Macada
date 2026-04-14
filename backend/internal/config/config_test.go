package config

import (
	"strings"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	validBase := func() Config {
		return Config{
			AnthropicKey:    "sk-ant-test-key",
			SandboxSecret:   "test-sandbox-secret",
			ControlPlaneURL: "http://backend:8080",
			JWTSecret:       "a-real-secret-value",
		}
	}

	tests := []struct {
		name         string
		modify       func(*Config)
		wantErrors   []string
		wantWarnings []string
	}{
		{
			name:   "valid config has no errors or warnings",
			modify: func(c *Config) {},
		},
		{
			name:       "empty AnthropicKey is an error",
			modify:     func(c *Config) { c.AnthropicKey = "" },
			wantErrors: []string{"ANTHROPIC_API_KEY"},
		},
		{
			name:       "empty SandboxSecret is an error",
			modify:     func(c *Config) { c.SandboxSecret = "" },
			wantErrors: []string{"SANDBOX_SECRET"},
		},
		{
			name:       "invalid ControlPlaneURL is an error",
			modify:     func(c *Config) { c.ControlPlaneURL = "not-a-url" },
			wantErrors: []string{"CONTROL_PLANE_URL"},
		},
		{
			name:       "empty ControlPlaneURL is an error",
			modify:     func(c *Config) { c.ControlPlaneURL = "" },
			wantErrors: []string{"CONTROL_PLANE_URL"},
		},
		{
			name:         "default JWTSecret produces a warning",
			modify:       func(c *Config) { c.JWTSecret = "change-me-in-production" },
			wantWarnings: []string{"JWT_SECRET"},
		},
		{
			name: "multiple missing values produce multiple errors",
			modify: func(c *Config) {
				c.AnthropicKey = ""
				c.SandboxSecret = ""
			},
			wantErrors: []string{"ANTHROPIC_API_KEY", "SANDBOX_SECRET"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validBase()
			tt.modify(&cfg)
			result := cfg.Validate()

			if len(tt.wantErrors) == 0 && len(result.Errors) > 0 {
				t.Errorf("expected no errors, got %v", result.Errors)
			}
			for _, want := range tt.wantErrors {
				found := false
				for _, e := range result.Errors {
					if strings.Contains(e, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", want, result.Errors)
				}
			}

			if len(tt.wantWarnings) == 0 && len(result.Warnings) > 0 {
				t.Errorf("expected no warnings, got %v", result.Warnings)
			}
			for _, want := range tt.wantWarnings {
				found := false
				for _, w := range result.Warnings {
					if strings.Contains(w, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning containing %q, got %v", want, result.Warnings)
				}
			}
		})
	}
}
