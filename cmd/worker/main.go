package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/logger"
	"github.com/asm-platform/asm/internal/store"
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	// -- Event Bus --
	var bus events.Bus
	if cfg.RedisURL != "" {
		redisBus, err := events.NewRedisBus(cfg.RedisURL)
		if err != nil {
			slog.Warn("Redis unavailable, using local bus", "error", err)
			bus = events.NewLocalBus()
		} else {
			bus = redisBus
		}
	} else {
		bus = events.NewLocalBus()
	}

	// -- Store --
	var s store.Store
	ctx := context.Background()
	if cfg.DatabaseURL != "" {
		pgStore, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("Postgres unavailable", "error", err)
			os.Exit(1)
		}
		s = pgStore
	} else {
		slog.Error("DATABASE_URL is required in temporal mode")
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

	// -- Executor (runs LLM calls inside Temporal activities) --
	exec := executor.NewDirectExecutor(llmRegistry, bus, activeLLMProvider)

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
	temporalClient, err := temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
	if err != nil {
		slog.Error("Failed to connect to Temporal", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	// -- Activities + Worker --
	acts := temporalpkg.NewActivities(s, bus, exec)
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
