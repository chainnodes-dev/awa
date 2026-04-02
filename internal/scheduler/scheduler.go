package scheduler

import (
	"context"
	"log"

	"github.com/robfig/cron/v3"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// StartFn is called by the scheduler to start a workflow run.
type StartFn func(ctx context.Context, workflowName, version string, input map[string]interface{}) (*asmtypes.WorkflowRun, error)

// Scheduler manages cron-based and event-based workflow triggers.
type Scheduler struct {
	cron    *cron.Cron
	startFn StartFn
}

type CronTrigger struct {
	Schedule        string                 // cron expression e.g. "0 9 * * *"
	WorkflowName    string
	WorkflowVersion string
	Input           map[string]interface{}
}

func New(startFn StartFn) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		startFn: startFn,
	}
}

func (s *Scheduler) AddCronTrigger(t CronTrigger) (cron.EntryID, error) {
	id, err := s.cron.AddFunc(t.Schedule, func() {
		ctx := context.Background()
		run, err := s.startFn(ctx, t.WorkflowName, t.WorkflowVersion, t.Input)
		if err != nil {
			log.Printf("[scheduler] failed to start '%s': %v", t.WorkflowName, err)
			return
		}
		log.Printf("[scheduler] started run %s for '%s'", run.ID, t.WorkflowName)
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Scheduler) RemoveTrigger(id cron.EntryID) {
	s.cron.Remove(id)
}

func (s *Scheduler) Start() { s.cron.Start() }

func (s *Scheduler) Stop() context.Context { return s.cron.Stop() }
