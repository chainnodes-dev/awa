package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/asm-platform/asm/internal/api"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/health"
	"github.com/asm-platform/asm/internal/logger"
	"github.com/asm-platform/asm/internal/migrate"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/scheduler"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/internal/telemetry"
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
)

// Removed redundant seedLLMConfigs. Seeding is now handled via api.SeedLLMConfigs.

func main() {
	// Load .env if present. Existing environment variables take precedence so
	// that explicit exports (e.g. EXECUTOR_MODE=temporal go run ...) are never
	// overwritten. Silently ignore the file when it doesn't exist.
	_ = godotenv.Load()

	cfg := config.Load()
	logger.Init(cfg.LogLevel)
	

	// -- Health checker (populated as deps are wired) --
	checker := health.New()

	ctx := context.Background()

	// -- Event Bus --
	// NewRedisBus only parses the URL — the TCP handshake is lazy.
	// Ping immediately so we fall back to the local bus when Redis is down
	// rather than silently dropping events at runtime.
	var bus events.Bus
	if cfg.RedisURL != "" {
		redisBus, err := events.NewRedisBus(cfg.RedisURL)
		if err == nil {
			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			err = redisBus.Ping(pingCtx)
			pingCancel()
		}
		if err != nil {
			slog.Warn("Redis unavailable, using in-process event bus — multi-process fan-out disabled",
				"error", err)
			bus = events.NewLocalBus()
		} else {
			checker.Add("redis", redisBus)
			bus = redisBus
			slog.Info("Event bus: Redis", "url", cfg.RedisURL)
		}
	} else {
		slog.Info("Event bus: in-process (REDIS_URL not set)")
		bus = events.NewLocalBus()
	}

	// -- Database migrations --
	if cfg.DatabaseURL != "" {
		if err := migrate.Up(cfg.DatabaseURL); err != nil {
			slog.Error("Migration failed", "error", err)
			os.Exit(1)
		}
	}

	// -- Store --
	var s store.Store
	if cfg.DatabaseURL != "" {
		pgStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Warn("Postgres unavailable, using memory store", "error", err)
			s = store.NewMemoryStore()
		} else {
			checker.Add("postgres", pgStore)
			s = pgStore
		}
	} else {
		s = store.NewMemoryStore()
	}
	defer s.Close()

	// -- Telemetry --
	if cfg.EnableTelemetry {
		telemetry.Start(s)
	}

	// -- Ensure Default Tenant exists --
	defaultTenant, err := api.EnsureDefaultTenant(ctx, s)
	if err != nil {
		slog.Error("Failed to ensure default tenant", "error", err)
		os.Exit(1)
	}
	tenantCtx := store.WithTenantID(ctx, defaultTenant.ID)

	// -- Seeding (Sync) --
	api.SeedAdminWithTenant(cfg, s, defaultTenant)
	api.SeedLLMConfigs(cfg, s, defaultTenant.ID)
	// Workflows and MCP seeding will happen below after mcpEntries are loaded.

	// -- LLM Config: Load registry from DB --
	llmRegistry, activeLLMProvider, err := llm.BuildRegistryFromDB(tenantCtx, s)
	if err != nil {
		slog.Warn("Failed to build LLM registry from DB, using empty registry", "error", err)
		llmRegistry = llm.NewRegistry()
	}
	if activeLLMProvider == "" {
		if first := llmRegistry.First(); first != nil {
			activeLLMProvider = first.Name()
			slog.Info("LLM default provider", "provider", activeLLMProvider)
		} else {
			slog.Warn("No LLM providers configured — agent states will fail")
		}
	}

	// -- Temporal client (Mandatory) --
	temporalClient, err := temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
	if err != nil {
		slog.Error("Failed to connect to Temporal", "address", cfg.TemporalAddress, "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
	if err := temporalpkg.CheckHealth(healthCtx, temporalClient); err != nil {
		slog.Error("Temporal health check failed", "address", cfg.TemporalAddress, "error", err)
		os.Exit(1)
	}
	healthCancel()

	temporalEngine := temporalpkg.NewEngineClient(temporalClient, cfg.TemporalTaskQueue)

	// -- Engine --
	engine := orchestrator.NewEngine(s, bus, temporalEngine)
	slog.Info("Temporal connected",
		"address", cfg.TemporalAddress,
		"namespace", cfg.TemporalNamespace,
		"queue", cfg.TemporalTaskQueue)

	// -- AI Workflow Generator --
	// Resolve mcp_registry.yaml relative to the project root.
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	registryPath := filepath.Join(projectRoot, "mcp_registry.yaml")

	mcpEntries, regErr := designer.LoadMCPRegistry(registryPath)
	if regErr != nil {
		slog.Warn("MCP registry not loaded; AI generation will have no server context", "error", regErr)
		mcpEntries = nil
	}

	// -- Seeding (Sync - Continued) --
	api.SeedWorkflows(s, defaultTenant.ID)
	api.SeedMCPServers(s, defaultTenant.ID, mcpEntries)

	var gen *designer.Generator
	// Use the active LLM provider if available.
	if llmProv, _ := llmRegistry.Get(activeLLMProvider); llmProv != nil {
		gen = designer.NewGenerator(llmProv, mcpEntries, s)
		slog.Info("AI workflow generator enabled", "provider", activeLLMProvider)
	} else {
		slog.Info("AI workflow generator disabled (no LLM provider configured)")
	}

	// -- Scheduler --
	// Load all stored workflow definitions and register any that carry a cron
	// schedule expression.  New workflows are registered when SaveDefinition is
	// called (handled in the API layer); removed when DeleteDefinition is called.
	sched := scheduler.New(engine.StartRun)
	if defs, err := s.ListDefinitions(ctx, store.DefinitionFilter{}); err == nil {
		for _, def := range defs {
			if err := sched.AddWorkflow(def); err != nil {
				slog.Warn("Failed to register workflow schedule", "workflow", def.Metadata.Name, "error", err)
			}
		}
	}
	sched.Start()
	slog.Info("Scheduler started")

	// -- API Server --
	srv := api.NewServer(cfg, engine, s, bus, gen, mcpEntries, checker, sched, llmRegistry)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Server listening", "port", cfg.Port)
		if err := srv.Start(); err != nil {
			slog.Info("Server stopped", "reason", err)
		}
	}()

	<-quit
	slog.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	stopCtx := sched.Stop()
	<-stopCtx.Done() // wait for in-flight cron jobs to finish
	slog.Info("Goodbye")
}
