package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	rtcfg "github.com/cchu-code/managed-agents/internal/runtime/config"
	rtctx "github.com/cchu-code/managed-agents/internal/runtime/context"
	"github.com/cchu-code/managed-agents/internal/runtime/loop"
	"github.com/cchu-code/managed-agents/internal/runtime/reporter"
	rt "github.com/cchu-code/managed-agents/internal/runtime"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const defaultBasePath = "/workspace"

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	basePath := os.Getenv("WORKSPACE_PATH")
	if basePath == "" {
		basePath = defaultBasePath
	}

	// 1. Load config from filesystem
	log.Info().Str("base", basePath).Msg("loading config")
	cfg, err := rtcfg.Load(basePath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	log.Info().
		Str("agent_id", cfg.Agent.ID).
		Str("session_id", cfg.Agent.SessionID).
		Str("model", cfg.Settings.Model.ID).
		Int("skills", len(cfg.Skills)).
		Msg("config loaded")

	// 2. Create reporter
	rep := reporter.NewReporter(
		cfg.Agent.ControlPlaneURL,
		cfg.Agent.SessionID,
		cfg.Agent.ControlPlaneToken,
	)
	defer rep.Close()

	// Report startup
	_ = rep.Report(context.Background(), "runtime.started", map[string]string{
		"agent_id":   cfg.Agent.ID,
		"session_id": cfg.Agent.SessionID,
	})

	// 3. Create components
	apiClient := loop.NewAnthropicClient(cfg.Agent.AnthropicAPIKey)
	toolExec := loop.NewToolExecutor(basePath)
	compressor := rtctx.NewCompressor(rtctx.DefaultCompressionConfig())

	agentLoop := loop.NewLoop(*cfg, apiClient, toolExec, rep, compressor)

	// 4. Start HTTP server
	server := rt.NewServer(agentLoop)
	addr := ":9090"
	srv := &http.Server{
		Addr:    addr,
		Handler: server.Handler(),
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", addr).Msg("runtime server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("runtime server failed")
		}
	}()

	// 5. Install packages if configured
	if hasPackages(cfg.Packages) {
		log.Info().Msg("installing packages...")
		installPackages(cfg.Packages)
		_ = rep.Report(context.Background(), "runtime.packages_installed", nil)
	}

	log.Info().Msg("runtime ready")

	<-done
	log.Info().Msg("shutting down...")

	_ = rep.Report(context.Background(), "runtime.stopped", nil)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}

	log.Info().Msg("runtime stopped")
}

func hasPackages(p rtcfg.PackagesConfig) bool {
	return len(p.Apt) > 0 || len(p.Pip) > 0 || len(p.Npm) > 0 ||
		len(p.Go) > 0 || len(p.Cargo) > 0 || len(p.Gem) > 0
}

func installPackages(p rtcfg.PackagesConfig) {
	exec := loop.NewToolExecutor("/workspace")
	ctx := context.Background()

	if len(p.Apt) > 0 {
		cmd := fmt.Sprintf("apt-get update -qq && apt-get install -y -qq %s", joinArgs(p.Apt))
		result := exec.Execute(ctx, "bash", toJSON(map[string]string{"command": cmd}))
		if result.IsError {
			log.Warn().Str("output", result.Content).Msg("apt install failed")
		}
	}
	if len(p.Pip) > 0 {
		cmd := fmt.Sprintf("pip install -q %s", joinArgs(p.Pip))
		result := exec.Execute(ctx, "bash", toJSON(map[string]string{"command": cmd}))
		if result.IsError {
			log.Warn().Str("output", result.Content).Msg("pip install failed")
		}
	}
	if len(p.Npm) > 0 {
		cmd := fmt.Sprintf("npm install -g %s", joinArgs(p.Npm))
		result := exec.Execute(ctx, "bash", toJSON(map[string]string{"command": cmd}))
		if result.IsError {
			log.Warn().Str("output", result.Content).Msg("npm install failed")
		}
	}
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}

func toJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
