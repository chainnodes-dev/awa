package temporal

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/api/enums/v1"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// NewClient dials Temporal and returns a ready client.
// Note: client.Dial establishes a lazy gRPC connection — it succeeds even when
// the Temporal server is unreachable. Call CheckHealth to verify liveness.
func NewClient(address, namespace string) (client.Client, error) {
	c, err := client.Dial(client.Options{
		HostPort:  address,
		Namespace: namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("dial temporal at %s: %w", address, err)
	}
	return c, nil
}

// CheckHealth pings the Temporal server to verify the connection is live.
// Use a context with a short deadline (e.g. 5 s) to avoid blocking at startup.
func CheckHealth(ctx context.Context, c client.Client) error {
	_, err := c.CheckHealth(ctx, nil)
	if err != nil {
		return fmt.Errorf("temporal health check: %w", err)
	}
	return nil
}

// EngineClient implements orchestrator.TemporalEngineClient using the Temporal SDK.
type EngineClient struct {
	client    client.Client
	taskQueue string
}

// NewEngineClient wraps an existing Temporal client.
func NewEngineClient(c client.Client, taskQueue string) *EngineClient {
	return &EngineClient{client: c, taskQueue: taskQueue}
}

// StartWorkflow starts a new ASMWorkflow execution for the given run.
// The workflow ID is deterministically derived from the run ID so duplicate
// starts are rejected by Temporal.
func (e *EngineClient) StartWorkflow(ctx context.Context, run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (string, error) {
	wfID := "asm-run-" + run.ID
	opts := client.StartWorkflowOptions{
		ID:                    wfID,
		TaskQueue:             e.taskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}
	_, err := e.client.ExecuteWorkflow(ctx, opts, ASMWorkflow, WorkflowParams{
		RunID:      run.ID,
		TenantID:   run.TenantID,
		Def:        def,
		Blackboard: run.Blackboard,
	})
	if err != nil {
		return "", fmt.Errorf("execute workflow: %w", err)
	}
	return wfID, nil
}

// SendTriggerSignal sends an external trigger signal to an active Phaxa workflow.
func (e *EngineClient) SendTriggerSignal(ctx context.Context, temporalID, trigger string, payload map[string]interface{}) error {
	return e.client.SignalWorkflow(ctx, temporalID, "", SignalTrigger, TriggerSignalPayload{
		Trigger: trigger,
		Payload: payload,
	})
}

// SendHITLSignal sends a HITL resolution signal to an active Phaxa workflow.
func (e *EngineClient) SendHITLSignal(ctx context.Context, temporalID string, sig asmtypes.HITLSignal) error {
	return e.client.SignalWorkflow(ctx, temporalID, "", SignalHITLResolution, HITLResolutionPayload{
		Resolution: sig.Resolution,
		Resolver:   sig.Resolver,
		Comment:    sig.Comment,
		Payload:    sig.Payload,
	})
}

// SendChatSignal sends a chat message to a waiting agent in Temporal.
func (e *EngineClient) SendChatSignal(ctx context.Context, temporalID, message, sender string) error {
	return e.client.SignalWorkflow(ctx, temporalID, "", SignalChat, ChatSignalPayload{
		Message: message,
		Sender:  sender,
	})
}

// TerminateWorkflow immediately stops a running Temporal workflow.
func (e *EngineClient) TerminateWorkflow(ctx context.Context, temporalID string) error {
	return e.client.TerminateWorkflow(ctx, temporalID, "", "User terminated workflow")
}

// AwaitWorkflowCompletion blocks until the specified temporal workflow finishes execution.
func (e *EngineClient) AwaitWorkflowCompletion(ctx context.Context, temporalID string) error {
	run := e.client.GetWorkflow(ctx, temporalID, "")
	return run.Get(ctx, nil)
}

// Close releases the underlying Temporal client connection.
func (e *EngineClient) Close() {
	e.client.Close()
}
