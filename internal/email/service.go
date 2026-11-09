package email

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// WorkflowStarter starts a workflow run when an inbound email matches a trigger.
type WorkflowStarter func(ctx context.Context, tenantID, workflowName, version string, inputs map[string]interface{}) (*asmtypes.WorkflowRun, error)

// Service coordinates inbound email ingestion, event publishing, and workflow execution.
type Service struct {
	store    store.Store
	bus      events.Bus
	starter  WorkflowStarter
	pollers  map[string]*IMAPPoller // tenantID -> poller
}

// NewService creates a new email service.
func NewService(s store.Store, bus events.Bus, starter WorkflowStarter) *Service {
	return &Service{
		store:   s,
		bus:     bus,
		starter: starter,
		pollers: make(map[string]*IMAPPoller),
	}
}

// IngestEmail processes a parsed email message, publishes platform events, and initiates matching workflow runs.
func (s *Service) IngestEmail(ctx context.Context, tenantID string, email *ParsedEmail, source string, targetWorkflow string) (*asmtypes.WorkflowRun, error) {
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}
	if source == "" {
		source = "webhook"
	}

	payload := events.InboundEmailPayload{
		TenantID:    tenantID,
		MessageID:   email.MessageID,
		From:        email.From,
		To:          email.To,
		Cc:          email.Cc,
		Subject:     email.Subject,
		Date:        email.Date,
		TextBody:    email.TextBody,
		HTMLBody:    email.HTMLBody,
		Attachments: email.Attachments,
		Headers:     email.Headers,
		Source:      source,
	}

	// 1. Publish platform event to bus so WebSocket subscribers and waiting wait-states wake up
	if s.bus != nil {
		_ = s.bus.Publish(ctx, events.New(events.EmailInbound, payload))
		// Also publish as generic workflow event so wait states with on_event: "email.inbound" wake up
		_ = s.bus.Publish(ctx, events.New(events.WorkflowEvent, map[string]interface{}{
			"tenant_id":  tenantID,
			"event_name": "email.inbound",
			"payload":   email.ToMap(),
		}))
		_ = s.bus.Publish(ctx, events.New(events.WorkflowEvent, map[string]interface{}{
			"tenant_id":  tenantID,
			"event_name": "email.received",
			"payload":   email.ToMap(),
		}))
	}

	// 2. If a target workflow is specified (e.g. from the webhook URL query or trigger), start it
	if targetWorkflow != "" && s.starter != nil {
		tCtx := store.WithTenantID(ctx, tenantID)
		latestDef, _, err := s.store.GetLatestDefinition(tCtx, targetWorkflow)
		if err != nil || latestDef == nil {
			return nil, fmt.Errorf("target workflow %q not found: %w", targetWorkflow, err)
		}

		run, err := s.starter(ctx, tenantID, latestDef.Metadata.Name, latestDef.Metadata.Version, email.ToMap())
		if err != nil {
			return nil, fmt.Errorf("start workflow %q on email: %w", targetWorkflow, err)
		}
		slog.Info("Started workflow run from inbound email", "workflow", targetWorkflow, "run_id", run.ID, "subject", email.Subject)
		return run, nil
	}

	// 3. Otherwise, search for any workflow in the tenant that has an email trigger
	if s.starter != nil {
		tCtx := store.WithTenantID(ctx, tenantID)
		defs, err := s.store.ListDefinitions(tCtx, store.DefinitionFilter{})
		if err == nil {
			for _, def := range defs {
				for _, tr := range def.Triggers {
					if tr.Type == "email" || tr.Type == "email_inbound" || tr.Type == "email_imap" {
						// Check filter if specified (e.g. subject_contains or from_domain)
						if matchTriggerFilter(tr.Config, email) {
							run, err := s.starter(ctx, tenantID, def.Metadata.Name, def.Metadata.Version, email.ToMap())
							if err == nil {
								slog.Info("Triggered workflow from email filter match", "workflow", def.Metadata.Name, "run_id", run.ID)
								return run, nil
							}
						}
					}
				}
			}
		}
	}

	return nil, nil
}

func matchTriggerFilter(config map[string]interface{}, email *ParsedEmail) bool {
	if config == nil {
		return true
	}

	if subContains, ok := config["subject_contains"].(string); ok && subContains != "" {
		if !strings.Contains(strings.ToLower(email.Subject), strings.ToLower(subContains)) {
			return false
		}
	}

	if fromContains, ok := config["from_contains"].(string); ok && fromContains != "" {
		if !strings.Contains(strings.ToLower(email.From), strings.ToLower(fromContains)) {
			return false
		}
	}

	return true
}

// RegisterIMAPPoller sets up an active poller for a tenant.
func (s *Service) RegisterIMAPPoller(ctx context.Context, tenantID string, cfg IMAPConfig, interval string) error {
	var dur time.Duration
	if interval != "" {
		dur, _ = time.ParseDuration(interval)
	}
	if dur <= 0 {
		dur = 60 * time.Second
	}

	if existing, ok := s.pollers[tenantID]; ok {
		existing.Stop()
		delete(s.pollers, tenantID)
	}

	poller := NewIMAPPoller(cfg, dur, func(pollCtx context.Context, email *ParsedEmail) error {
		_, err := s.IngestEmail(pollCtx, tenantID, email, "imap", "")
		return err
	})

	if err := poller.Start(ctx); err != nil {
		return err
	}

	s.pollers[tenantID] = poller
	return nil
}

// StopAll terminates all running background IMAP pollers.
func (s *Service) StopAll() {
	for id, p := range s.pollers {
		p.Stop()
		delete(s.pollers, id)
	}
}
