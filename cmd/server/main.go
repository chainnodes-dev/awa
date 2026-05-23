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
	"go.temporal.io/sdk/client"

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
	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/internal/telemetry"
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	logger.Init(cfg.LogLevel)
	
	checker := health.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Database & Migrations
	var s store.Store
	if cfg.DatabaseURL != "" {
		slog.Info("Connecting to database...", "url", cfg.DatabaseURL)
		var pgStore *store.PostgresStore
		var err error
		for i := 0; i < 10; i++ {
			if err = migrate.Up(cfg.DatabaseURL); err == nil {
				pgStore, err = store.NewPostgresStore(ctx, cfg.DatabaseURL)
				if err == nil {
					break
				}
			}
			slog.Warn("Waiting for database...", "attempt", i+1, "error", err)
			time.Sleep(3 * time.Second)
		}
		if err != nil {
			slog.Error("Database failed to initialize", "error", err)
			os.Exit(1)
		}
		checker.Add("postgres", pgStore)
		s = pgStore
	} else {
		s = store.NewMemoryStore()
	}
	defer s.Close()

	// 2. Tenants & Seeding
	defaultTenant, err := api.EnsureDefaultTenant(ctx, s)
	if err != nil {
		slog.Error("Failed to ensure default tenant", "error", err)
		os.Exit(1)
	}
	tenantCtx := store.WithTenantID(ctx, defaultTenant.ID)
	
	if cfg.AdminPassword != "" {
		api.SeedAdminWithTenant(cfg, s, defaultTenant)
	}
	api.SeedLLMConfigs(cfg, s, defaultTenant.ID)

	// 3. Temporal (Lazy connection)
	var temporalEngine orchestrator.TemporalEngineClient
	go func() {
		for {
			tClient, err := temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
			if err == nil {
				hCtx, hCancel := context.WithTimeout(ctx, 5*time.Second)
				err = temporalpkg.CheckHealth(hCtx, tClient)
				hCancel()
				if err == nil {
					slog.Info("Temporal connected successfully")
					temporalEngine = temporalpkg.NewEngineClient(tClient, cfg.TemporalTaskQueue)
					checker.Add("temporal", &temporalPinger{client: tClient})
					return
				}
			}
			slog.Warn("Temporal not ready yet, retrying...", "error", err)
			time.Sleep(5 * time.Second)
		}
	}()

	// 4. Infrastructure
	var bus events.Bus = events.NewLocalBus()
	if cfg.RedisURL != "" {
		if rb, err := events.NewRedisBus(cfg.RedisURL); err == nil {
			bus = rb
			checker.Add("redis", rb)
		}
	}

	// 5. Engine
	engine := orchestrator.NewEngine(s, bus, &lazyTemporalExecutor{get: func() orchestrator.TemporalEngineClient { return temporalEngine }})
	
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	registryPath := filepath.Join(projectRoot, "mcp_registry.yaml")
	mcpEntries, _ := designer.LoadMCPRegistry(registryPath)

	api.SeedWorkflows(s, defaultTenant.ID)

	llmRegistry, activeLLMProvider, _ := llm.BuildRegistryFromDB(tenantCtx, s)
	var gen *designer.Generator
	if llmProv, _ := llmRegistry.Get(activeLLMProvider); llmProv != nil {
		gen = designer.NewGenerator(llmProv, mcpEntries, s)
	}

	var secretMgr secrets.SecretManager
	if pgs, ok := s.(*store.PostgresStore); ok {
		masterKey, err := secrets.GenerateOrLoadMasterKey()
		if err != nil {
			slog.Error("Failed to load master key", "error", err)
			os.Exit(1)
		}
		secretMgr, err = secrets.NewAESSecretManager(pgs.Pool(), masterKey)
		if err != nil {
			slog.Error("Failed to initialize secret manager", "error", err)
			os.Exit(1)
		}
	} else {
		secretMgr = secrets.NewMemorySecretManager()
	}

	// 6. Server
	sched := scheduler.New(engine.StartRun)
	srv := api.NewServer(cfg, engine, s, bus, gen, mcpEntries, checker, sched, llmRegistry, secretMgr)
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Server binding port", "port", cfg.Port)
		if err := srv.Start(); err != nil {
			slog.Info("Server stopped", "reason", err)
		}
	}()

	sched.Start()
	if cfg.EnableTelemetry {
		telemetry.Start(s)
	}

	<-quit
	slog.Info("Shutting down...")
	sdCtx, sdCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer sdCancel()
	_ = srv.Shutdown(sdCtx)
	slog.Info("Goodbye")
}

// wrappers
type temporalPinger struct{ client client.Client }
func (tp *temporalPinger) Ping(ctx context.Context) error { return temporalpkg.CheckHealth(ctx, tp.client) }

type lazyTemporalExecutor struct{ get func() orchestrator.TemporalEngineClient }
func (l *lazyTemporalExecutor) StartWorkflow(ctx context.Context, run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (string, error) {
	exec := l.get()
	if exec == nil { return "", context.DeadlineExceeded }
	return exec.StartWorkflow(ctx, run, def)
}
func (l *lazyTemporalExecutor) SendTriggerSignal(ctx context.Context, tid, trigger string, payload map[string]interface{}) error {
	exec := l.get()
	if exec == nil { return context.DeadlineExceeded }
	return exec.SendTriggerSignal(ctx, tid, trigger, payload)
}
func (l *lazyTemporalExecutor) SendHITLSignal(ctx context.Context, tid string, sig asmtypes.HITLSignal) error {
	exec := l.get()
	if exec == nil { return context.DeadlineExceeded }
	return exec.SendHITLSignal(ctx, tid, sig)
}
func (l *lazyTemporalExecutor) SendChatSignal(ctx context.Context, tid, msg, snd string) error {
	exec := l.get()
	if exec == nil { return context.DeadlineExceeded }
	return exec.SendChatSignal(ctx, tid, msg, snd)
}
func (l *lazyTemporalExecutor) TerminateWorkflow(ctx context.Context, tid string) error {
	exec := l.get()
	if exec == nil { return context.DeadlineExceeded }
	return exec.TerminateWorkflow(ctx, tid)
}
func (l *lazyTemporalExecutor) AwaitWorkflowCompletion(ctx context.Context, tid string) error {
	exec := l.get()
	if exec == nil { return context.DeadlineExceeded }
	return exec.AwaitWorkflowCompletion(ctx, tid)
}
func (l *lazyTemporalExecutor) Close() {
	if exec := l.get(); exec != nil { exec.Close() }
}
