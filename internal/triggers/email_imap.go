package triggers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/asm-platform/asm/internal/email"
	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

type emailIMAPTrigger struct {
	tenantID     string
	workflowName string
	version      string
	def          asmtypes.TriggerDef
	store        store.Store
	secretMgr    secrets.SecretManager
	poller       *email.IMAPPoller
}

func newEmailIMAPTrigger(tenantID, workflowName, version string, def asmtypes.TriggerDef, s store.Store, sm secrets.SecretManager) (TriggerInstance, error) {
	return &emailIMAPTrigger{
		tenantID:     tenantID,
		workflowName: workflowName,
		version:      version,
		def:          def,
		store:        s,
		secretMgr:    sm,
	}, nil
}

func (t *emailIMAPTrigger) Start(ctx context.Context, startFn StartWorkflowFn) error {
	cfg := email.IMAPConfig{
		Host:    "imap.gmail.com",
		Port:    993,
		Mailbox: "INBOX",
		UseTLS:  true,
	}

	if h, ok := t.def.Config["host"].(string); ok && h != "" {
		cfg.Host = h
	}
	if p, ok := t.def.Config["port"].(float64); ok && p > 0 {
		cfg.Port = int(p)
	}
	if u, ok := t.def.Config["username"].(string); ok && u != "" {
		cfg.Username = u
	}
	if mb, ok := t.def.Config["mailbox"].(string); ok && mb != "" {
		cfg.Mailbox = mb
	}

	// Password can be direct or from secret
	passwordKey, ok := t.def.Config["password_secret"].(string)
	if !ok {
		passwordKey = "IMAP_PASSWORD"
	}
	if t.secretMgr != nil {
		if pw, err := t.secretMgr.GetSecret(ctx, t.tenantID, passwordKey); err == nil && pw != "" {
			cfg.Password = pw
		}
	}
	if cfg.Password == "" {
		if pw, ok := t.def.Config["password"].(string); ok {
			cfg.Password = pw
		}
	}

	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" {
		return fmt.Errorf("email_imap trigger requires host, username, and password (or password_secret)")
	}

	interval := 60 * time.Second
	if intStr, ok := t.def.Config["interval"].(string); ok && intStr != "" {
		if d, err := time.ParseDuration(intStr); err == nil && d > 0 {
			interval = d
		}
	}

	t.poller = email.NewIMAPPoller(cfg, interval, func(pollCtx context.Context, parsed *email.ParsedEmail) error {
		inputs := parsed.ToMap()
		_, err := startFn(pollCtx, t.tenantID, t.workflowName, t.version, inputs)
		if err != nil {
			slog.Error("Failed to start workflow from email_imap trigger", "workflow", t.workflowName, "error", err)
			return err
		}
		slog.Info("Successfully triggered workflow from email_imap", "workflow", t.workflowName, "subject", parsed.Subject)
		return nil
	})

	return t.poller.Start(ctx)
}

func (t *emailIMAPTrigger) Stop(ctx context.Context) error {
	if t.poller != nil {
		t.poller.Stop()
	}
	return nil
}
