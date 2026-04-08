// cmd/specialist-worker is a template for building specialist worker binaries.
//
// Copy this directory to your own repo, rename it, and swap out the example
// handlers below for your own deterministic Go logic. The binary only needs
// access to Temporal — no Postgres, no Redis, no LLM keys required.
//
// Environment variables:
//
//	TEMPORAL_ADDRESS    Temporal frontend address (default: localhost:7233)
//	TEMPORAL_NAMESPACE  Temporal namespace        (default: default)
//	SPECIALIST_TASK_QUEUE  Task queue this worker polls (default: specialist-workers)
//	                    Must match task_queue in workflow YAML agent definitions.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/logger"
	"github.com/asm-platform/asm/internal/specialist"
	temporalpkg "github.com/asm-platform/asm/internal/temporal"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)

	// The task queue this worker polls. Override via env var.
	taskQueue := os.Getenv("SPECIALIST_TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = "specialist-workers"
	}

	// ── Register handlers ─────────────────────────────────────────────────
	// Each handler name must match the "name" field of the agent in the
	// workflow YAML, and that agent must declare:
	//
	//   task_queue: <taskQueue value above>
	//
	// Add or remove handlers as needed for your domain.

	sw := specialist.New(taskQueue)
	sw.Register("invoice-validator", validateInvoice)
	// sw.Register("my-other-agent", myOtherHandler)
	// ─────────────────────────────────────────────────────────────────────

	temporalClient, err := temporalpkg.NewClient(cfg.TemporalAddress, cfg.TemporalNamespace)
	if err != nil {
		slog.Error("Failed to connect to Temporal", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	w := sw.Build(temporalClient)

	// Graceful shutdown on SIGINT / SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	interruptCh := make(chan interface{}, 1)
	go func() {
		<-sigCh
		interruptCh <- struct{}{}
	}()

	slog.Info("Specialist worker starting", "queue", taskQueue,
		"temporal", cfg.TemporalAddress, "handlers", sw.Handlers())
	if err := w.Run(interruptCh); err != nil {
		slog.Error("Worker run error", "error", err)
		os.Exit(1)
	}
	slog.Info("Specialist worker stopped")
}

// ── Example handler: invoice validation ──────────────────────────────────────
//
// validateInvoice is a deterministic alternative to an LLM agent for invoice
// validation states. It checks required fields and basic constraints without
// any AI involvement.
//
// Corresponding workflow YAML:
//
//	agents:
//	  - name: invoice-validator
//	    task_queue: specialist-workers  # routes here
//	    # no model: needed — this handler runs in Go, not in an LLM
//
// Returned triggers: "validation_passed" | "validation_failed"
func validateInvoice(_ context.Context, bb map[string]interface{}) (*asmtypes.AgentOutput, error) {
	// 1. Required fields check.
	required := []string{"invoice_id", "vendor_id", "amount", "currency"}
	for _, field := range required {
		v := bb[field]
		if v == nil || v == "" {
			return &asmtypes.AgentOutput{
				Trigger:   "validation_failed",
				Reasoning: fmt.Sprintf("required field missing: %s", field),
				BlackboardUpdates: map[string]interface{}{
					"validation_error": fmt.Sprintf("missing required field: %s", field),
				},
			}, nil
		}
	}

	// 2. Amount must be a positive number.
	amount, ok := toFloat64(bb["amount"])
	if !ok || amount <= 0 {
		return &asmtypes.AgentOutput{
			Trigger:   "validation_failed",
			Reasoning: "amount must be a positive number",
			BlackboardUpdates: map[string]interface{}{
				"validation_error": "amount must be a positive number",
			},
		}, nil
	}

	// 3. Currency must be a known ISO 4217 code.
	if !isKnownCurrency(fmt.Sprintf("%v", bb["currency"])) {
		return &asmtypes.AgentOutput{
			Trigger:   "validation_failed",
			Reasoning: fmt.Sprintf("unknown currency code: %v", bb["currency"]),
			BlackboardUpdates: map[string]interface{}{
				"validation_error": fmt.Sprintf("unknown currency: %v", bb["currency"]),
			},
		}, nil
	}

	// All checks passed.
	return &asmtypes.AgentOutput{
		Trigger:   "validation_passed",
		Reasoning: "all required fields present and valid",
		BlackboardUpdates: map[string]interface{}{
			"validation_passed": true,
		},
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// knownCurrencies is a minimal set of ISO 4217 codes for the example handler.
// Production usage should use a proper currency library or a complete table.
var knownCurrencies = map[string]bool{
	"USD": true, "EUR": true, "GBP": true, "JPY": true, "CHF": true,
	"CAD": true, "AUD": true, "CNY": true, "SEK": true, "NOK": true,
	"DKK": true, "NZD": true, "SGD": true, "HKD": true, "MXN": true,
	"BRL": true, "INR": true, "ZAR": true, "PLN": true, "CZK": true,
}

func isKnownCurrency(code string) bool {
	return knownCurrencies[code]
}
