package store

import (
	"context"
	"time"
)

// EventSubscription tracks a run that is blocked at a wait state listening for
// a named platform event. Created by RegisterEventSubscription activity, consumed
// (deleted) when EmitWorkflowEvent delivers the event to the waiting run.
type EventSubscription struct {
	// ID is deterministic: temporalID + "__" + stateName — prevents duplicate registrations.
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	RunID          string    `json:"run_id"`
	TemporalID     string    `json:"temporal_id"`  // Temporal workflow execution ID
	EventName      string    `json:"event_name"`
	OnMatchTrigger string    `json:"on_match_trigger"` // trigger fired when event arrives
	CreatedAt      time.Time `json:"created_at"`
}

// EventSubscriptionStore manages per-tenant event subscriptions for wait-on-event states.
type EventSubscriptionStore interface {
	// RegisterEventSubscription upserts a subscription (idempotent via deterministic ID).
	RegisterEventSubscription(ctx context.Context, sub *EventSubscription) error
	// UnregisterEventSubscription removes a subscription by ID (no-op if already gone).
	UnregisterEventSubscription(ctx context.Context, id string) error
	// ListEventSubscriptions returns all subscriptions waiting for a given event name
	// in the given tenant. Results are ordered by created_at ascending.
	ListEventSubscriptions(ctx context.Context, tenantID, eventName string) ([]*EventSubscription, error)
}
