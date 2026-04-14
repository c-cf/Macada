package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	osexec "os/exec"
	"os/signal"
	"syscall"
	"time"

	rtcfg "github.com/c-cf/macada/internal/runtime/config"
	rtctx "github.com/c-cf/macada/internal/runtime/context"
	"github.com/c-cf/macada/internal/runtime/loop"
	"github.com/c-cf/macada/internal/runtime/reporter"
	"github.com/c-cf/macada/internal/runtime/toolset"
	rt "github.com/c-cf/macada/internal/runtime"
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
	// LLM calls go through the control plane proxy — API key never enters the container
	llmProxyURL := fmt.Sprintf("%s/internal/v1/sandbox/%s/llm",
		cfg.Agent.ControlPlaneURL, cfg.Agent.SessionID)
	apiClient := loop.NewAnthropicClient(llmProxyURL, cfg.Agent.ControlPlaneToken)
	toolExec := loop.NewToolExecutor(basePath)
	compressor := rtctx.NewCompressor(rtctx.DefaultCompressionConfig())

	// Resolve toolset from agent type (nil if no toolset specified).
	// The reporter doubles as file uploader — it POSTs to the control plane's internal file API.
	ts := toolset.Resolve(cfg.Agent.Type, basePath, rep)
	if ts != nil {
		log.Info().Str("agent_type", cfg.Agent.Type).Msg("toolset activated")
	}

	agentLoop := loop.NewLoop(*cfg, apiClient, toolExec, ts, rep, compressor)

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
	ctx := context.Background()

	if len(p.Apt) > 0 {
		// Run apt-get update first
		runCmd(ctx, "apt-get", "update", "-qq")
		// Use exec.Command directly to avoid shell injection
		args := append([]string{"install", "-y", "-qq", "--"}, p.Apt...)
		if output, err := osexec.CommandContext(ctx, "apt-get", args...).CombinedOutput(); err != nil {
			log.Warn().Str("output", string(output)).Msg("apt install failed")
		}
	}
	if len(p.Pip) > 0 {
		args := append([]string{"install", "-q", "--"}, p.Pip...)
		if output, err := osexec.CommandContext(ctx, "pip", args...).CombinedOutput(); err != nil {
			log.Warn().Str("output", string(output)).Msg("pip install failed")
		}
	}
	if len(p.Npm) > 0 {
		args := append([]string{"install", "-g", "--"}, p.Npm...)
		if output, err := osexec.CommandContext(ctx, "npm", args...).CombinedOutput(); err != nil {
			log.Warn().Str("output", string(output)).Msg("npm install failed")
		}
	}
}

func runCmd(ctx context.Context, name string, args ...string) {
	if output, err := osexec.CommandContext(ctx, name, args...).CombinedOutput(); err != nil {
		log.Warn().Str("output", string(output)).Str("cmd", name).Msg("command failed")
	}
}