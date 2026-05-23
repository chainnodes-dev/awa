package temporal

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// NewWorker creates a Temporal worker with all Phaxa workflows and activities
// registered. The caller starts it with w.Run(interruptCh).
func NewWorker(c client.Client, taskQueue string, acts *Activities) worker.Worker {
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(ASMWorkflow)
	w.RegisterActivity(acts)
	return w
}
