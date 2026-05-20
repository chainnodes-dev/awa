package temporal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	temporalactivity "go.temporal.io/sdk/activity"

	"github.com/asm-platform/asm/internal/enterprise"
	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor/code"
	"github.com/asm-platform/asm/internal/metrics"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// Activities holds all dependencies needed by Temporal activity functions.
// Register an instance with the Temporal worker via worker.RegisterActivity(acts).
type Activities struct {
	Store     store.Store
	Bus       events.Bus
	Executor  orchestrator.Executor
	Client    client.Client // used by EmitWorkflowEvent to signal waiting workflows
	Verifier  *enterprise.Verifier
	SecretMgr secrets.SecretManager
}

func NewActivities(s store.Store, bus events.Bus, exec orchestrator.Executor, c client.Client, v *enterprise.Verifier, secretMgr secrets.SecretManager) *Activities {
	return &Activities{Store: s, Bus: bus, Executor: exec, Client: c, Verifier: v, SecretMgr: secretMgr}
}

// UpdateRun persists run state changes to the store.
func (a *Activities) UpdateRun(ctx context.Context, p UpdateRunParams) error {
	run, err := a.Store.GetRun(ctx, p.RunID)
	if err != nil {
		return err
	}
	run.CurrentState = p.CurrentState
	run.Status = p.Status
	run.Blackboard = p.Blackboard
	run.FailureReason = p.FailureReason
	run.UpdatedAt = time.Now()
	if p.IsTerminal {
		now := time.Now()
		run.CompletedAt = &now

		metrics.RunsCompletedTotal.WithLabelValues(p.WorkflowName, p.WorkflowVersion, string(p.Status), p.TenantID).Inc()
		metrics.RunsActive.WithLabelValues(p.WorkflowName, p.TenantID).Dec()

		// Recording RunDurationSeconds if StartTime is known
		if !run.StartedAt.IsZero() {
			metrics.RunDurationSeconds.WithLabelValues(p.WorkflowName, p.WorkflowVersion, string(p.Status), p.TenantID).Observe(time.Since(run.StartedAt).Seconds())
		}
	}
	return a.Store.UpdateRun(ctx, run)
}

// RecordTransition appends an immutable transition record to the store.
func (a *Activities) RecordTransition(ctx context.Context, p RecordTransitionParams) error {
	if p.Record.ID == "" {
		p.Record.ID = uuid.NewString()
	}
	if p.Record.Timestamp.IsZero() {
		p.Record.Timestamp = time.Now()
	}

	metrics.StateTransitionsTotal.WithLabelValues(
		p.WorkflowName,
		p.Record.FromState,
		p.Record.ToState,
		p.Record.Trigger,
		p.TenantID,
	).Inc()

	return a.Store.RecordTransition(ctx, p.Record)
}

// PublishEvent publishes a single event to the event bus.
//
// p.Data arrives from Temporal's JSON serialization as map[string]interface{},
// losing the original concrete type. We re-marshal it to json.RawMessage before
// constructing the event so that the payload is embedded as typed JSON bytes —
// consistent with events created directly via events.New in the direct executor.
func (a *Activities) PublishEvent(ctx context.Context, p PublishEventParams) error {
	raw, err := json.Marshal(p.Data)
	if err != nil {
		return err
	}
	return a.Bus.Publish(ctx, events.New(p.EventType, json.RawMessage(raw)))
}

// CreateHITL persists a new HITL request to the store.
func (a *Activities) CreateHITL(ctx context.Context, p CreateHITLParams) error {
	if p.Request.ID == "" {
		p.Request.ID = uuid.NewString()
	}
	if err := a.Store.CreateHITL(ctx, p.Request); err != nil {
		return err
	}

	// Notify the UI that we are waiting for human input.
	_ = a.Bus.Publish(ctx, events.New(events.HITLWaiting, events.HITLWaitingPayload{
		RunID:     p.Request.RunID,
		StateName: p.Request.StateName,
		Assignee:  p.Request.Assignee,
	}))

	return nil
}

// ExecuteCode runs a user-written JavaScript code node via the goja sandbox
// and returns its structured output. The activity heartbeats every 10 s so
// Temporal can detect worker failures during long-running scripts.
func (a *Activities) ExecuteCode(ctx context.Context, p ExecuteCodeParams) (*asmtypes.AgentOutput, error) {
	timeout := 60 * time.Second
	if p.StateDef.Timeout != "" {
		if d, err := time.ParseDuration(p.StateDef.Timeout); err == nil {
			timeout = d
		}
	}
	heartbeat := func() { temporalactivity.RecordHeartbeat(ctx, "code running") }
	return code.Execute(p.StateDef.Code.Code, p.Blackboard, p.ValidTriggers, timeout, heartbeat)
}

// ExecuteAgent runs an LLM agent and returns its structured output.
// This activity has a longer timeout than the others (10 min vs 10 s).
func (a *Activities) ExecuteAgent(ctx context.Context, p ExecuteAgentParams) (*asmtypes.AgentOutput, error) {
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				temporalactivity.RecordHeartbeat(ctx, "agent running")
			case <-done:
				return
			}
		}
	}()

	return a.Executor.Execute(ctx, orchestrator.AgentTask{
		RunID:      p.RunID,
		TenantID:   p.TenantID,
		AgentDef:   p.AgentDef,
		StateDef:   p.StateDef,
		Blackboard: p.Blackboard,
		Def:        p.Def,
	})
}


// LoadWorkflowDef loads a WorkflowDef by name and optional version.
// Used by subprocess states to resolve the child workflow before launching it.
func (a *Activities) LoadWorkflowDef(ctx context.Context, p LoadWorkflowDefParams) (*asmtypes.WorkflowDef, error) {
	ctx = store.WithTenantID(ctx, p.TenantID)
	var def *asmtypes.WorkflowDef
	var err error

	if p.Version != "" {
		def, _, err = a.Store.GetDefinition(ctx, p.WorkflowName, p.Version)
	} else {
		def, _, err = a.Store.GetLatestDefinition(ctx, p.WorkflowName)
	}

	if err != nil {
		return nil, fmt.Errorf("load workflow '%s' (version: %s): %w", p.WorkflowName, p.Version, err)
	}
	return def, nil
}

// GetRunBlackboard reads the current blackboard snapshot for a completed run.
// Used after a child skill workflow completes to extract its output fields.
func (a *Activities) GetRunBlackboard(ctx context.Context, p GetRunBlackboardParams) (map[string]interface{}, error) {
	ctx = store.WithTenantID(ctx, p.TenantID)
	run, err := a.Store.GetRun(ctx, p.RunID)
	if err != nil {
		return nil, fmt.Errorf("get run '%s': %w", p.RunID, err)
	}
	return run.Blackboard, nil
}

// RegisterEventSubscription persists a wait-on-event subscription so that
// EmitWorkflowEvent can locate this workflow when the named event fires.
func (a *Activities) RegisterEventSubscription(ctx context.Context, p RegisterEventSubscriptionParams) error {
	if err := a.Store.RegisterEventSubscription(ctx, p.Subscription); err != nil {
		return fmt.Errorf("register event subscription: %w", err)
	}
	return nil
}

// UnregisterEventSubscription removes the wait-on-event subscription after
// the event has been received (or the wait state timed out).
func (a *Activities) UnregisterEventSubscription(ctx context.Context, p UnregisterEventSubscriptionParams) error {
	return a.Store.UnregisterEventSubscription(ctx, p.SubscriptionID)
}

// EmitWorkflowEvent publishes a named event to the bus and signals every Temporal
// workflow in the same tenant that is currently waiting for that event name.
func (a *Activities) EmitWorkflowEvent(ctx context.Context, p EmitWorkflowEventParams) error {
	// 1. Publish to the event bus so WebSocket clients and other subscribers see it.
	raw, _ := json.Marshal(map[string]interface{}{
		"run_id":     p.RunID,
		"event_name": p.EventName,
		"payload":    p.Payload,
	})
	_ = a.Bus.Publish(ctx, events.New(events.WorkflowEvent, json.RawMessage(raw)))

	// 2. Find all runs in this tenant waiting for this event name.
	subs, err := a.Store.ListEventSubscriptions(ctx, p.TenantID, p.EventName)
	if err != nil {
		return fmt.Errorf("list event subscriptions for '%s': %w", p.EventName, err)
	}

	// 3. Signal each waiting Temporal workflow and remove the subscription.
	for _, sub := range subs {
		sigErr := a.Client.SignalWorkflow(ctx, sub.TemporalID, "", SignalTrigger, TriggerSignalPayload{
			Trigger: sub.OnMatchTrigger,
			Payload: p.Payload,
		})
		if sigErr != nil {
			// Log but continue — the workflow may have already completed.
			fmt.Printf("warn: signal workflow %s for event %s: %v\n", sub.TemporalID, p.EventName, sigErr)
			continue
		}
		_ = a.Store.UnregisterEventSubscription(ctx, sub.ID)
	}
	return nil
}

// GetLicenseClaims retrieves and verifies the license claims for a tenant.
func (a *Activities) GetLicenseClaims(ctx context.Context, tenantID string) (*enterprise.LicenseClaims, error) {
	tenant, err := a.Store.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if tenant.LicenseToken == "" {
		return enterprise.DefaultFreeLicense(tenantID), nil
	}

	if a.Verifier == nil {
		// If no verifier, we fallback to free tier for safety or error out?
		// Given this is for enforcement, if verifier is missing on worker,
		// we should probably warn and use free tier.
		return enterprise.DefaultFreeLicense(tenantID), nil
	}

	return a.Verifier.Verify(tenant.LicenseToken)
}

// SendTelegramMessage resolves Telegram Bot Token, Chat ID and sends message text.
func (a *Activities) SendTelegramMessage(ctx context.Context, p SendTelegramMessageParams) (string, error) {
	token, err := a.SecretMgr.GetSecret(ctx, p.TenantID, "TELEGRAM_BOT_TOKEN")
	if err != nil || token == "" {
		return "", fmt.Errorf("telegram token not configured for tenant %s: %w", p.TenantID, err)
	}

	payload := map[string]string{
		"chat_id": p.ChatID,
		"text":    p.MessageText,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send telegram request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram api returned status code %d", resp.StatusCode)
	}

	return "sent", nil
}

// SendDiscordMessage resolves Discord Bot Token, Channel ID and sends message text.
func (a *Activities) SendDiscordMessage(ctx context.Context, p SendDiscordMessageParams) (string, error) {
	token, err := a.SecretMgr.GetSecret(ctx, p.TenantID, "DISCORD_BOT_TOKEN")
	if err != nil || token == "" {
		return "", fmt.Errorf("discord token not configured for tenant %s: %w", p.TenantID, err)
	}

	payload := map[string]string{
		"content": p.MessageText,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", p.ChannelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send discord request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("discord api returned status code %d", resp.StatusCode)
	}

	return "sent", nil
}

