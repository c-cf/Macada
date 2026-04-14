package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                int
	DatabaseURL         string
	RedisURL            string
	AnthropicKey        string
	LogLevel            string
	SandboxSecret       string
	RuntimeImage        string
	ControlPlaneURL     string
	DockerHost          string
	NetworkName         string
	AdminSecret         string
	JWTSecret           string
	FileStoragePath     string
	SandboxMemoryMB     int64
	SandboxCPUs         float64
	CORSAllowedOrigins  []string
	RateLimitRPS        float64
	RateLimitBurst      int
	VaultEncryptionKey  string
}

// ValidationResult holds warnings and errors from config validation.
type ValidationResult struct {
	Errors   []string
	Warnings []string
}

// Validate checks that required configuration values are present and valid.
// It returns a ValidationResult containing any errors (fatal) and warnings (non-fatal).
func (c *Config) Validate() ValidationResult {
	var result ValidationResult

	if c.AnthropicKey == "" {
		result.Errors = append(result.Errors, "ANTHROPIC_API_KEY is required but not set — LLM calls will fail with opaque 401 errors")
	}

	if c.SandboxSecret == "" {
		result.Errors = append(result.Errors, "SANDBOX_SECRET is required but not set — sandbox token generation will fail")
	}

	if u, err := url.Parse(c.ControlPlaneURL); err != nil || u.Scheme == "" || u.Host == "" {
		result.Errors = append(result.Errors,
			fmt.Sprintf("CONTROL_PLANE_URL must be a valid URL with scheme and host, got %q", c.ControlPlaneURL))
	}

	if c.JWTSecret == "change-me-in-production" {
		result.Warnings = append(result.Warnings, "JWT_SECRET is using the default value \"change-me-in-production\" — this is insecure for production use")
	}

	return result
}

func Load() (*Config, error) {
	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		port = p
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	sandboxSecret := os.Getenv("SANDBOX_SECRET")
	runtimeImage := os.Getenv("RUNTIME_IMAGE")
	if runtimeImage == "" {
		runtimeImage = "macada-runtime:latest"
	}
	controlPlaneURL := os.Getenv("CONTROL_PLANE_URL")
	if controlPlaneURL == "" {
		controlPlaneURL = "http://backend:8080"
	}
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}
	networkName := os.Getenv("SANDBOX_NETWORK")
	if networkName == "" {
		networkName = "macada_internal"
	}

	adminSecret := os.Getenv("ADMIN_SECRET")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production"
	}

	fileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	if fileStoragePath == "" {
		fileStoragePath = "/data/files"
	}

	sandboxMemoryMB := int64(512)
	if v := os.Getenv("SANDBOX_MEMORY_MB"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid SANDBOX_MEMORY_MB: %w", err)
		}
		sandboxMemoryMB = n
	}

	sandboxCPUs := 0.5
	if v := os.Getenv("SANDBOX_CPUS"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid SANDBOX_CPUS: %w", err)
		}
		sandboxCPUs = f
	}

	// CORS: comma-separated list of allowed origins; defaults to "*" (dev-only).
	var corsOrigins []string
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		for _, o := range strings.Split(v, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	}
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"*"}
	}

	// Rate limiting
	rateLimitRPS := 20.0
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_RPS: %w", err)
		}
		rateLimitRPS = f
	}
	rateLimitBurst := 40
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_BURST: %w", err)
		}
		rateLimitBurst = n
	}

	// Vault encryption key (64 hex chars = 32 bytes for AES-256-GCM).
	// Optional: vault features are disabled when not set.
	vaultEncryptionKey := os.Getenv("VAULT_ENCRYPTION_KEY")

	return &Config{
		Port:               port,
		DatabaseURL:        dbURL,
		RedisURL:           redisURL,
		AnthropicKey:       anthropicKey,
		LogLevel:           logLevel,
		SandboxSecret:      sandboxSecret,
		RuntimeImage:       runtimeImage,
		ControlPlaneURL:    controlPlaneURL,
		DockerHost:         dockerHost,
		NetworkName:        networkName,
		AdminSecret:        adminSecret,
		JWTSecret:          jwtSecret,
		FileStoragePath:    fileStoragePath,
		SandboxMemoryMB:    sandboxMemoryMB,
		SandboxCPUs:        sandboxCPUs,
		CORSAllowedOrigins: corsOrigins,
		RateLimitRPS:       rateLimitRPS,
		RateLimitBurst:     rateLimitBurst,
		VaultEncryptionKey: vaultEncryptionKey,
	}, nil
}
