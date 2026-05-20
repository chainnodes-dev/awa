package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/enterprise"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/logger"
	"github.com/asm-platform/asm/internal/mcp"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	ctx := context.Background()

	// -- Event Bus --
	// Mirror the server's setup: ping Redis first so we fall back to a local bus
	// rather than silently dropping events when Redis is unreachable.
	// Without a shared bus (Redis), events published by the worker (agent.thinking,
	// agent.prompt, agent.response, etc.) never reach the server's WebSocket Hub.
	var bus events.Bus
	if cfg.RedisURL != "" {
		redisBus, err := events.NewRedisBus(cfg.RedisURL)
		if err == nil {
			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			err = redisBus.Ping(pingCtx)
			pingCancel()
		}
		if err != nil {
			slog.Warn("Redis unavailable, falling back to local bus — WebSocket events will NOT reach the API server",
				"url", cfg.RedisURL, "error", err)
			bus = events.NewLocalBus()
		} else {
			bus = redisBus
			slog.Info("Event bus: Redis (events will reach the API server WebSocket hub)", "url", cfg.RedisURL)
		}
	} else {
		slog.Warn("REDIS_URL not set — using local bus, WebSocket events will NOT reach the API server")
		bus = events.NewLocalBus()
	}

	// -- Store --
	var s store.Store
	if cfg.DatabaseURL != "" {
		pgStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("Postgres unavailable", "error", err)
			os.Exit(1)
		}
		s = pgStore
	} else {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	defer s.Close()

	// -- LLM Registry (loaded from DB) --
	workerTenantCtx := store.WithTenantID(ctx, store.DefaultTenantID)
	llmRegistry, activeLLMProvider, llmErr := llm.BuildRegistryFromDB(workerTenantCtx, s)
	if llmErr != nil {
		slog.Warn("Failed to build LLM registry from DB", "error", llmErr)
		llmRegistry = llm.NewRegistry()
	}
	if activeLLMProvider == "" {
		if first := llmRegistry.First(); first != nil {
			activeLLMProvider = first.Name()
			slog.Info("LLM default provider", "provider", activeLLMProvider)
		} else {
			slog.Warn("No LLM providers configured — agent activities will fail")
		}
	}

	// -- MCP Manager --
	mcpManager := mcp.NewManager(s)
	defer mcpManager.Shutdown()

	// Reload LLM registry from DB periodically to pick up UI config changes.
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			reg, defaultName, err := llm.BuildRegistryFromDB(workerTenantCtx, s)
			if err != nil {
				slog.Warn("LLM registry reload failed", "error", err)
				continue
			}
			llmRegistry.Reload(reg.Snapshot(), defaultName)
			slog.Debug("LLM registry reloaded from DB", "default", defaultName)
		}
	}()

	// -- Temporal client --
	var temporalClient client.Client
	var err error
	for i := 0; i < 24; i++ { // retry for up to 2 minutes
		temporalClient, err = temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
		if err == nil {
			break
		}
		slog.Warn("Temporal not ready or namespace default not found yet, retrying...", "attempt", i+1, "error", err)
		time.Sleep(5 * time.Second)
	}
	if err != nil {
		slog.Error("Failed to connect to Temporal after multiple attempts", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	// -- Activities + Worker --
	var verifier *enterprise.Verifier
	if cfg.LicensePublicKey != "" {
		pub, err := enterprise.ParseRSAPublicKey(cfg.LicensePublicKey)
		if err != nil {
			slog.Error("Failed to parse license public key", "error", err)
		} else {
			verifier = enterprise.NewVerifier(pub)
		}
	}

	var secretMgr secrets.SecretManager
	if pgs, ok := s.(*store.PostgresStore); ok {
		masterKey, err := secrets.GenerateOrLoadMasterKey()
		if err != nil {
			slog.Error("Failed to load master key in worker", "error", err)
			os.Exit(1)
		}
		secretMgr, err = secrets.NewAESSecretManager(pgs.Pool(), masterKey)
		if err != nil {
			slog.Error("Failed to initialize secret manager in worker", "error", err)
			os.Exit(1)
		}
	} else {
		secretMgr = secrets.NewMemorySecretManager()
	}

	temporalEngine := temporalpkg.NewEngineClient(temporalClient, cfg.TemporalTaskQueue)
	engine := orchestrator.NewEngine(s, bus, temporalEngine)
	exec := executor.NewExecutor(llmRegistry, bus, s, activeLLMProvider, mcpManager, engine, secretMgr)

	acts := temporalpkg.NewActivities(s, bus, exec, temporalClient, verifier, secretMgr)
	w := temporalpkg.NewWorker(temporalClient, cfg.TemporalTaskQueue, acts)

	// Graceful shutdown on SIGINT / SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Temporal's worker.Run expects <-chan interface{}.
	interruptCh := make(chan interface{}, 1)
	go func() {
		<-sigCh
		interruptCh <- struct{}{}
	}()

	slog.Info("Worker starting", "queue", cfg.TemporalTaskQueue, "temporal", cfg.TemporalAddress)
	if err := w.Run(interruptCh); err != nil {
		slog.Error("Worker run error", "error", err)
		os.Exit(1)
	}
	slog.Info("Worker stopped")
}
