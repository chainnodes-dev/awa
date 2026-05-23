package triggers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// StartWorkflowFn is called by a trigger to start a new workflow run.
type StartWorkflowFn func(ctx context.Context, tenantID, workflowName, version string, inputs map[string]interface{}) (*asmtypes.WorkflowRun, error)

// TriggerInstance represents an active listening thread or webhook endpoint.
type TriggerInstance interface {
	Start(ctx context.Context, startFn StartWorkflowFn) error
	Stop(ctx context.Context) error
}

type Manager struct {
	mu        sync.RWMutex
	instances map[string]TriggerInstance
	startFn   StartWorkflowFn
	store     store.Store
	secretMgr secrets.SecretManager
}

func NewManager(s store.Store, startFn StartWorkflowFn, secretMgr secrets.SecretManager) *Manager {
	return &Manager{
		instances: make(map[string]TriggerInstance),
		startFn:   startFn,
		store:     s,
		secretMgr: secretMgr,
	}
}

// AddWorkflow registers all triggers for a given workflow definition.
func (m *Manager) AddWorkflow(ctx context.Context, tenantID string, def *asmtypes.WorkflowDef) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, tDef := range def.Triggers {
		key := fmt.Sprintf("%s/%s/%s/%s", tenantID, def.Metadata.Name, def.Metadata.Version, tDef.Name)
		
		// If already running, skip or recreate. For now, we skip if it exists.
		if _, exists := m.instances[key]; exists {
			continue
		}

		var instance TriggerInstance
		var err error

		switch tDef.Type {
		case "webhook":
			// Webhooks don't need a background instance, they are purely reactive
			// via the HandleWebhook method. We just store a dummy instance to track it.
			instance = &webhookTrigger{}
		case "telegram":
			instance, err = newTelegramTrigger(tenantID, def.Metadata.Name, def.Metadata.Version, tDef, m.store, m.secretMgr)
		case "discord":
			instance, err = newDiscordTrigger(tenantID, def.Metadata.Name, def.Metadata.Version, tDef, m.store, m.secretMgr)
		default:
			slog.Warn("Unknown trigger type", "type", tDef.Type, "workflow", def.Metadata.Name)
			continue
		}

		if err != nil {
			slog.Error("Failed to create trigger", "type", tDef.Type, "error", err)
			continue
		}

		if err := instance.Start(ctx, m.startFn); err != nil {
			slog.Error("Failed to start trigger", "type", tDef.Type, "error", err)
			continue
		}

		m.instances[key] = instance
		slog.Info("Registered trigger", "key", key, "type", tDef.Type)
	}
	return nil
}

// RemoveWorkflow stops and unregisters all triggers for a given workflow.
func (m *Manager) RemoveWorkflow(ctx context.Context, tenantID, workflowName, version string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefix := fmt.Sprintf("%s/%s/%s/", tenantID, workflowName, version)
	for key, instance := range m.instances {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			_ = instance.Stop(ctx)
			delete(m.instances, key)
			slog.Info("Unregistered trigger", "key", key)
		}
	}
}

// HandleWebhook is called by the API router when a webhook payload is received.
func (m *Manager) HandleWebhook(ctx context.Context, tenantID, workflowName, version, triggerName string, payload map[string]interface{}) (*asmtypes.WorkflowRun, error) {
	m.mu.RLock()
	key := fmt.Sprintf("%s/%s/%s/%s", tenantID, workflowName, version, triggerName)
	_, exists := m.instances[key]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("webhook trigger '%s' not found or not active for workflow '%s'", triggerName, workflowName)
	}

	return m.startFn(ctx, tenantID, workflowName, version, payload)
}

type webhookTrigger struct{}

func (w *webhookTrigger) Start(ctx context.Context, startFn StartWorkflowFn) error { return nil }
func (w *webhookTrigger) Stop(ctx context.Context) error                           { return nil }

