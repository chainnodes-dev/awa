package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/asm-platform/asm/internal/api"
	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/orchestrator"
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
			log.Printf("[warn] Redis unavailable (%v), using local bus", err)
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
			log.Printf("[warn] Postgres unavailable (%v), using memory store", err)
			s = store.NewMemoryStore()
		} else {
			s = pgStore
		}
	} else {
		s = store.NewMemoryStore()
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
	if cfg.OllamaURL != "" {
		llmRegistry.Register(llm.NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel))
		log.Printf("[server] Ollama provider registered (url=%s model=%s)", cfg.OllamaURL, cfg.OllamaModel)
	}

	// Resolve active LLM provider: use configured value, fall back to first registered.
	activeLLMProvider := cfg.LLMProvider
	if _, err := llmRegistry.Get(activeLLMProvider); err != nil {
		if first := llmRegistry.First(); first != nil {
			log.Printf("[server] LLM provider %q not registered; falling back to %q", activeLLMProvider, first.Name())
			activeLLMProvider = first.Name()
		} else {
			log.Printf("[server] warning: no LLM providers registered — agent states will fail")
		}
	}

	// -- Executor --
	exec := executor.NewDirectExecutor(llmRegistry, bus, activeLLMProvider)

	// -- Engine --
	engine := orchestrator.NewEngine(s, bus, exec)

	// -- Temporal client (optional) --
	if cfg.ExecutorMode == "temporal" {
		temporalClient, err := temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
		if err != nil {
			log.Fatalf("[server] failed to connect to Temporal at %s: %v", cfg.TemporalAddress, err)
		}
		engine.UseTemporalClient(temporalpkg.NewEngineClient(temporalClient, cfg.TemporalTaskQueue))
		log.Printf("[server] executor mode: temporal (%s / ns=%s / queue=%s)",
			cfg.TemporalAddress, cfg.TemporalNamespace, cfg.TemporalTaskQueue)
	} else {
		log.Printf("[server] executor mode: direct")
	}

	// -- AI Workflow Generator --
	// Resolve mcp_registry.yaml relative to the project root.
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	registryPath := filepath.Join(projectRoot, "mcp_registry.yaml")

	var gen *designer.Generator
	mcpEntries, regErr := designer.LoadMCPRegistry(registryPath)
	if regErr != nil {
		log.Printf("[server] MCP registry not loaded (%v); AI generation will have no server context", regErr)
		mcpEntries = nil
	}
	// Use the active LLM provider if available.
	if llmProv, _ := llmRegistry.Get(activeLLMProvider); llmProv != nil {
		gen = designer.NewGenerator(llmProv, mcpEntries)
		log.Printf("[server] AI workflow generator enabled (provider: %s)", activeLLMProvider)
	} else {
		log.Printf("[server] AI workflow generator disabled (no LLM provider configured)")
	}

	// -- API Server --
	srv := api.NewServer(cfg, engine, s, bus, gen)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] listening on :%s", cfg.Port)
		if err := srv.Start(); err != nil {
			log.Printf("[server] stopped: %v", err)
		}
	}()

	<-quit
	log.Println("[server] shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("[server] goodbye")
}
