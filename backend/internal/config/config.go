package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port            int
	DatabaseURL     string
	RedisURL        string
	AnthropicKey    string
	LogLevel        string
	SandboxSecret   string
	RuntimeImage    string
	ControlPlaneURL string
	DockerHost      string
	NetworkName     string
	AdminSecret     string
	JWTSecret       string
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

	return &Config{
		Port:            port,
		DatabaseURL:     dbURL,
		RedisURL:        redisURL,
		AnthropicKey:    anthropicKey,
		LogLevel:        logLevel,
		SandboxSecret:   sandboxSecret,
		RuntimeImage:    runtimeImage,
		ControlPlaneURL: controlPlaneURL,
		DockerHost:      dockerHost,
		NetworkName:     networkName,
		AdminSecret:     adminSecret,
		JWTSecret:       jwtSecret,
	}, nil
}
