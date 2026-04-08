package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cchu-code/managed-agents/internal/api"
	"github.com/cchu-code/managed-agents/internal/api/handler"
	"github.com/cchu-code/managed-agents/internal/config"
	"github.com/cchu-code/managed-agents/internal/infra/postgres"
	redisinfra "github.com/cchu-code/managed-agents/internal/infra/redis"
	rtctx "github.com/cchu-code/managed-agents/internal/runtime/context"
	"github.com/cchu-code/managed-agents/internal/sandbox"
	"github.com/cchu-code/managed-agents/internal/service"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	migrateUp := flag.Bool("migrate-up", false, "Run database migrations up")
	migrateDown := flag.Bool("migrate-down", false, "Run database migrations down")
	flag.Parse()

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()
	log.Info().Msg("connected to PostgreSQL")

	// Handle migration commands
	if *migrateUp {
		log.Info().Msg("running migrations up...")
		if err := postgres.MigrateUp(ctx, pool); err != nil {
			log.Fatal().Err(err).Msg("migration up failed")
		}
		log.Info().Msg("migrations completed successfully")
		return
	}
	if *migrateDown {
		log.Info().Msg("running migrations down...")
		if err := postgres.MigrateDown(ctx, pool); err != nil {
			log.Fatal().Err(err).Msg("migration down failed")
		}
		log.Info().Msg("migrations rolled back successfully")
		return
	}

	// Connect to Redis
	redisClient, err := redisinfra.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}
	defer redisClient.Close()
	log.Info().Msg("connected to Redis")

	// Initialize repositories
	envRepo := postgres.NewEnvironmentRepo(pool)
	agentRepo := postgres.NewAgentRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)
	eventRepo := postgres.NewEventRepo(pool)
	analyticsRepo := postgres.NewAnalyticsRepo(pool)
	skillRepo := postgres.NewSkillRepo(pool)
	workspaceRepo := postgres.NewWorkspaceRepo(pool)
	apiKeyRepo := postgres.NewAPIKeyRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	memberRepo := postgres.NewWorkspaceMemberRepo(pool)

	// Initialize event bus
	eventBus := redisinfra.NewEventBus(redisClient)

	// Initialize sandbox orchestrator (replaces in-process Runner)
	tokenGen := sandbox.NewTokenGenerator(cfg.SandboxSecret)
	compressor := rtctx.NewCompressor(rtctx.DefaultCompressionConfig())
	orchestrator := sandbox.NewOrchestrator(
		sandbox.OrchestratorConfig{
			RuntimeImage:    cfg.RuntimeImage,
			ControlPlaneURL: cfg.ControlPlaneURL,
			AnthropicAPIKey: cfg.AnthropicKey,
			DockerHost:      cfg.DockerHost,
			NetworkName:     cfg.NetworkName,
		},
		tokenGen,
		compressor,
		sessionRepo, eventRepo, eventBus,
		skillRepo, envRepo, analyticsRepo,
	)

	// Initialize auth service
	authService := service.NewAuthService(userRepo, workspaceRepo, memberRepo, cfg.JWTSecret)

	// Initialize handlers
	envHandler := handler.NewEnvironmentHandler(envRepo)
	agentHandler := handler.NewAgentHandler(agentRepo)
	sessionHandler := handler.NewSessionHandler(sessionRepo, agentRepo, envRepo)
	eventHandler := handler.NewEventHandler(eventRepo, sessionRepo, eventBus, orchestrator)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsRepo)
	skillHandler := handler.NewSkillHandler(skillRepo)
	internalHandler := handler.NewInternalHandler(eventRepo, sessionRepo, eventBus, analyticsRepo, tokenGen)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceRepo)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyRepo, workspaceRepo)
	bootstrapHandler := handler.NewBootstrapHandler(cfg.AdminSecret, workspaceRepo, apiKeyRepo)
	authHandler := handler.NewAuthHandler(authService)

	// Build router
	router := api.NewRouter(api.Deps{
		EnvironmentHandler: envHandler,
		AgentHandler:       agentHandler,
		SessionHandler:     sessionHandler,
		EventHandler:       eventHandler,
		AnalyticsHandler:   analyticsHandler,
		SkillHandler:       skillHandler,
		InternalHandler:    internalHandler,
		WorkspaceHandler:   workspaceHandler,
		APIKeyHandler:      apiKeyHandler,
		BootstrapHandler:   bootstrapHandler,
		AuthHandler:        authHandler,
		APIKeyRepo:         apiKeyRepo,
		JWTValidator:       authService,
		MemberRepo:         memberRepo,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", addr).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-done
	log.Info().Msg("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server shutdown error")
	}

	log.Info().Msg("server stopped")
}
