package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// StartFn is called by the scheduler to start a workflow run.
type StartFn func(ctx context.Context, workflowName, version string, input map[string]interface{}) (*asmtypes.WorkflowRun, error)

// Scheduler manages cron-based workflow triggers.
type Scheduler struct {
	cron    *cron.Cron
	startFn StartFn

	mu      sync.Mutex
	entries map[string]cron.EntryID // key: "tenantID/name@version"
}

// CronTrigger describes a single cron-scheduled workflow start.
type CronTrigger struct {
	Schedule        string                 // 6-field cron expression (seconds first)
	TenantID        string
	WorkflowName    string
	WorkflowVersion string
	Input           map[string]interface{}
}

func New(startFn StartFn) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		startFn: startFn,
		entries: make(map[string]cron.EntryID),
	}
}

// AddCronTrigger registers a cron expression and returns the entry ID.
func (s *Scheduler) AddCronTrigger(t CronTrigger) (cron.EntryID, error) {
	id, err := s.cron.AddFunc(t.Schedule, func() {
		// Scoping background workflow runs to the correct tenant context
		ctx := store.WithTenantID(context.Background(), t.TenantID)
		run, err := s.startFn(ctx, t.WorkflowName, t.WorkflowVersion, t.Input)
		if err != nil {
			slog.Error("Scheduler failed to start workflow", "workflow", t.WorkflowName, "tenant", t.TenantID, "error", err)
			return
		}
		slog.Info("Scheduler started run", "run_id", run.ID, "workflow", t.WorkflowName, "tenant", t.TenantID)
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

// AddWorkflow registers a cron trigger for def if it has a non-empty Schedule.
// It is idempotent: calling it again for the same workflow replaces the existing trigger.
func (s *Scheduler) AddWorkflow(tenantID string, def *asmtypes.WorkflowDef) error {
	if def.Metadata.Schedule == "" {
		return nil
	}
	key := workflowKey(tenantID, def.Metadata.Name, def.Metadata.Version)

	// Remove any existing trigger for this workflow before adding the new one.
	s.removeByKey(key)

	id, err := s.AddCronTrigger(CronTrigger{
		Schedule:        def.Metadata.Schedule,
		TenantID:        tenantID,
		WorkflowName:    def.Metadata.Name,
		WorkflowVersion: def.Metadata.Version,
	})
	if err != nil {
		return fmt.Errorf("register schedule for %s: %w", key, err)
	}

	s.mu.Lock()
	s.entries[key] = id
	s.mu.Unlock()

	slog.Info("Workflow schedule registered", "workflow", def.Metadata.Name,
		"version", def.Metadata.Version, "tenant", tenantID, "schedule", def.Metadata.Schedule)
	return nil
}

// RemoveWorkflow deregisters the cron trigger for the given workflow, if any.
// It matches by tenantID, workflow name, and version.
func (s *Scheduler) RemoveWorkflow(tenantID, name, version string) {
	s.removeByKey(workflowKey(tenantID, name, version))
}

func (s *Scheduler) removeByKey(key string) {
	s.mu.Lock()
	id, ok := s.entries[key]
	if ok {
		delete(s.entries, key)
	}
	s.mu.Unlock()
	if ok {
		s.cron.Remove(id)
	}
}

// RemoveTrigger removes a trigger by its cron entry ID.
func (s *Scheduler) RemoveTrigger(id cron.EntryID) {
	s.cron.Remove(id)
}

// Start begins the cron scheduler in a background goroutine.
func (s *Scheduler) Start() { s.cron.Start() }

// Stop signals the scheduler to stop accepting new jobs and returns a context
// that is done when all currently-running jobs have finished.
func (s *Scheduler) Stop() context.Context { return s.cron.Stop() }

func workflowKey(tenantID, name, version string) string { return tenantID + "/" + name + "@" + version }
