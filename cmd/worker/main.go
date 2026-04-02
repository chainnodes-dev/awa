package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/store"
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
)

func main() {
	cfg := config.Load()

	// -- Event Bus --
	var bus events.Bus
	if cfg.RedisURL != "" {
		redisBus, err := events.NewRedisBus(cfg.RedisURL)
		if err != nil {
			log.Printf("[worker] Redis unavailable (%v), using local bus", err)
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
			log.Fatalf("[worker] Postgres unavailable: %v", err)
		}
		s = pgStore
	} else {
		log.Fatal("[worker] DATABASE_URL is required in temporal mode")
	}
	defer s.Close()

	// -- LLM Registry --
	llmRegistry := llm.NewRegistry()
	if cfg.AnthropicKey != "" {
		llmRegistry.Register(llm.NewAnthropicProvider(cfg.AnthropicKey, cfg.DefaultModel))
	}
	if cfg.OpenAIKey != "" {
		llmRegistry.Register(llm.NewOpenAIProvider(cfg.OpenAIKey, cfg.DefaultModel))
	}

	// -- Executor (runs LLM calls inside Temporal activities) --
	exec := executor.NewDirectExecutor(llmRegistry, bus, cfg.LLMProvider)

	// -- Temporal client --
	temporalClient, err := temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
	if err != nil {
		log.Fatalf("[worker] failed to connect to Temporal: %v", err)
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

	log.Printf("[worker] starting on task queue '%s' (temporal: %s)", cfg.TemporalTaskQueue, cfg.TemporalAddress)
	if err := w.Run(interruptCh); err != nil {
		log.Fatalf("[worker] run error: %v", err)
	}
	log.Println("[worker] stopped")
}
