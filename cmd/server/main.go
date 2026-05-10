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
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
)

// seedLLMConfigs inserts LLM provider configs from env vars on first boot.
// It is a no-op if any llm_configs rows already exist for the default tenant.
func seedLLMConfigs(ctx context.Context, s store.Store, cfg *config.Config) {
	hasAny, err := s.HasAnyLLMConfig(ctx)
	if err != nil || hasAny {
		return
	}
	type seed struct{ provider, apiKey, baseURL, model string }
	seeds := []seed{
		{"anthropic", cfg.AnthropicKey, "", cfg.DefaultModel},
		{"openai", cfg.OpenAIKey, "", cfg.DefaultModel},
		{"grok", cfg.GrokKey, "", "grok-3"},
		{"deepseek", cfg.DeepseekKey, "", "deepseek-chat"},
		{"gemini", cfg.GeminiKey, "", "gemini-2.0-flash"},
		{"ollama", "", cfg.OllamaURL, cfg.OllamaModel},
	}
	for _, sd := range seeds {
		if sd.provider == "ollama" {
			if sd.baseURL == "" {
				continue
			}
		} else if sd.apiKey == "" {
			continue
		}
		_ = s.UpsertLLMConfig(ctx, &store.LLMConfig{
			TenantID:     store.DefaultTenantID,
			Provider:     sd.provider,
			APIKey:       sd.apiKey,
			BaseURL:      sd.baseURL,
			DefaultModel: sd.model,
			Enabled:      true,
		})
	}
	if cfg.LLMProvider != "" {
		_ = s.SetDefaultProvider(ctx, cfg.LLMProvider)
	}
	slog.Info("Seeded LLM configs from environment")
}

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

	// -- LLM Config: seed DB from env on first boot, then load registry from DB --
	tenantCtx := store.WithTenantID(ctx, store.DefaultTenantID)
	seedLLMConfigs(tenantCtx, s, cfg)

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

	var gen *designer.Generator
	mcpEntries, regErr := designer.LoadMCPRegistry(registryPath)
	if regErr != nil {
		slog.Warn("MCP registry not loaded; AI generation will have no server context", "error", regErr)
		mcpEntries = nil
	}
	// Use the active LLM provider if available.
	if llmProv, _ := llmRegistry.Get(activeLLMProvider); llmProv != nil {
		gen = designer.NewGenerator(llmProv, mcpEntries, s)
		slog.Info("AI workflow generator enabled", "provider", activeLLMProvider)
	} else {
		slog.Info("AI workflow generator disabled (no LLM provider configured)")
	}

	// -- MCP Server Registry: seed from YAML if DB table is empty --
	// The mcp_registry.yaml is the operator-curated source of truth for teams that
	// haven't yet managed servers via the API. Once the DB has any entries, those
	// take precedence (see ListMCPServers handler). Seeding is idempotent: we only
	// insert when HasAnyMCPServer returns false for the default tenant.
	if len(mcpEntries) > 0 {
		seedCtx := store.WithTenantID(ctx, store.DefaultTenantID)
		if hasAny, err := s.HasAnyMCPServer(seedCtx); err == nil && !hasAny {
			for _, e := range mcpEntries {
				srv := &store.MCPServer{
					TenantID:    store.DefaultTenantID,
					Name:        e.Name,
					EnvVars:     map[string]string{e.EnvVar: ""},
					Description: e.Description,
					CreatedBy:   "seed",
				}
				if err := s.CreateMCPServer(seedCtx, srv); err != nil {
					slog.Warn("MCP seed: failed to insert entry", "name", e.Name, "error", err)
				}
			}
			slog.Info("MCP server registry seeded from YAML", "count", len(mcpEntries))
		}
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
	srv := api.NewServer(cfg, engine, s, bus, gen, checker, sched, llmRegistry)

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
