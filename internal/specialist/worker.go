// Package specialist provides the SDK for building specialist Temporal workers.
//
// A specialist worker replaces LLM agents for specific states with deterministic
// Go code. It registers on its own task queue and implements the same
// ExecuteAgent activity contract as the LLM worker — the workflow and the
// blackboard are unaffected.
//
// Quick-start:
//
//	sw := specialist.New("validate-workers")
//	sw.Register("invoice-validator", func(ctx context.Context, bb map[string]interface{}) (*asmtypes.AgentOutput, error) {
//	    // ... your logic here
//	    return &asmtypes.AgentOutput{Trigger: "validation_passed", ...}, nil
//	})
//	w := sw.Build(temporalClient)
//	w.Run(interruptCh)
//
// The corresponding workflow YAML declares:
//
//	agents:
//	  - name: invoice-validator
//	    task_queue: validate-workers  # routes here instead of the LLM worker
package specialist

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	temporalpkg "github.com/asm-platform/asm/internal/temporal"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// HandlerFunc is the signature every specialist handler must satisfy.
// It receives the current blackboard and returns an AgentOutput containing
// the trigger to fire and any blackboard updates — identical contract to LLM agents.
type HandlerFunc func(ctx context.Context, bb map[string]interface{}) (*asmtypes.AgentOutput, error)

// Worker is a builder for a specialist Temporal worker.
// Create one per specialist domain (e.g. "validate-workers"), register a
// HandlerFunc for each agent name the worker handles, then call Build.
type Worker struct {
	taskQueue string
	handlers  map[string]HandlerFunc
}

// New creates a new specialist Worker that listens on taskQueue.
// taskQueue must match the task_queue field in the workflow YAML agent definition.
func New(taskQueue string) *Worker {
	return &Worker{
		taskQueue: taskQueue,
		handlers:  make(map[string]HandlerFunc),
	}
}

// Register associates agentName with fn.
// agentName must exactly match the "name" field in the workflow YAML agent definition.
// Registering the same name twice overwrites the first registration.
func (w *Worker) Register(agentName string, fn HandlerFunc) {
	w.handlers[agentName] = fn
}

// Build creates and returns a Temporal worker pre-configured with the
// ExecuteAgent activity wired to the registered handlers.
// The caller is responsible for starting (w.Run) and stopping the returned worker.
func (w *Worker) Build(c client.Client) worker.Worker {
	tw := worker.New(c, w.taskQueue, worker.Options{})
	// RegisterActivityWithOptions lets us pin the activity name to "ExecuteAgent"
	// — the same name the LLM worker registers — so Temporal routes correctly
	// based solely on the task queue declared in the workflow.
	tw.RegisterActivityWithOptions(w.executeAgent, activity.RegisterOptions{
		Name: "ExecuteAgent",
	})
	return tw
}

// TaskQueue returns the task queue this worker listens on.
func (w *Worker) TaskQueue() string {
	return w.taskQueue
}

// Handlers returns the names of all registered agent handlers.
func (w *Worker) Handlers() []string {
	names := make([]string, 0, len(w.handlers))
	for name := range w.handlers {
		names = append(names, name)
	}
	return names
}

// executeAgent is the Temporal activity implementation for specialist workers.
// It dispatches to the registered HandlerFunc keyed by AgentDef.Name.
// This method is NOT exported — it is registered by Build() under the name
// "ExecuteAgent" to match the LLM worker's activity name.
func (w *Worker) executeAgent(ctx context.Context, p temporalpkg.ExecuteAgentParams) (*asmtypes.AgentOutput, error) {
	fn, ok := w.handlers[p.AgentDef.Name]
	if !ok {
		return nil, fmt.Errorf(
			"specialist worker on queue %q: no handler registered for agent %q (registered: %v)",
			w.taskQueue, p.AgentDef.Name, w.Handlers(),
		)
	}
	return fn(ctx, p.Blackboard)
}
