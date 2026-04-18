-- Event subscriptions
-- Tracks which runs are currently blocked at a wait state listening for a
-- named platform event. Populated by RegisterEventSubscription activity,
-- consumed (deleted) by EmitWorkflowEvent when the event fires.

CREATE TABLE IF NOT EXISTS event_subscriptions (
    id               TEXT        PRIMARY KEY,   -- deterministic: temporalID__stateName
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    run_id           TEXT        NOT NULL,
    temporal_id      TEXT        NOT NULL,       -- Temporal workflow execution ID (asm-run-{uuid})
    event_name       TEXT        NOT NULL,
    on_match_trigger TEXT        NOT NULL DEFAULT 'event_received',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup: given a tenant + event name, find all waiting runs.
CREATE INDEX IF NOT EXISTS idx_event_subs_lookup ON event_subscriptions(tenant_id, event_name);
