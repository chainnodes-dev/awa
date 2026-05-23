package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asm-platform/asm/internal/store"
	"github.com/google/uuid"
)

// Report represents the anonymized data sent to Phaxa HQ.
type Report struct {
	InstallationID string `json:"installation_id"`
	Country        string `json:"country"`
	WorkflowCount  int    `json:"workflow_count"`
	TotalRuns      int    `json:"total_runs"`
	RunsLast7d     int    `json:"runs_last_7d"`
	RunsLast30d    int    `json:"runs_last_30d"`
	ReportedAt     time.Time `json:"reported_at"`
}

// Global service instance for the API handlers to use.
var globalService *Service

// Start initializes and starts the global telemetry service.
// This matches the call site in cmd/server/main.go.
func Start(store store.Store) {
	globalService = NewService(store, 24*time.Hour)
	go globalService.Start(context.Background())
}

// GetGlobalService returns the initialized telemetry service.
func GetGlobalService() *Service {
	return globalService
}

// Service handles the periodic collection and reporting of telemetry data.
type Service struct {
	store    store.Store
	interval time.Duration
	endpoint string
}

func NewService(s store.Store, interval time.Duration) *Service {
	endpoint := os.Getenv("PHAXA_TELEMETRY_URL")
	if endpoint == "" {
		endpoint = "https://telemetry.phaxa.com/v1/report"
	}
	return &Service{
		store:    s,
		interval: interval,
		endpoint: endpoint,
	}
}

// Start runs the telemetry reporter in a background loop.
func (s *Service) Start(ctx context.Context) {
	log.Printf("Telemetry: starting background service (interval: %v)", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run once at startup
	s.Report(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Report(ctx)
		}
	}
}

// Report collects and sends the telemetry report if enabled.
func (s *Service) Report(ctx context.Context) {
	enabled, _ := s.store.GetSystemSetting(ctx, "telemetry_enabled")
	if enabled == "false" {
		return
	}

	report, err := s.Collect(ctx)
	if err != nil {
		log.Printf("Telemetry: failed to collect data: %v", err)
		return
	}

	payload, _ := json.Marshal(report)
	req, err := http.NewRequestWithContext(ctx, "POST", s.endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Telemetry: failed to send report: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		log.Printf("Telemetry: HQ returned error: %d", resp.StatusCode)
	} else {
		log.Printf("Telemetry: report sent successfully (%s)", report.InstallationID)
	}
}

// Collect aggregates the current metrics into a Report.
func (s *Service) Collect(ctx context.Context) (*Report, error) {
	// 1. Get or generate Installation ID
	instID, err := s.store.GetSystemSetting(ctx, "installation_id")
	if err != nil || instID == "" {
		instID = uuid.NewString()
		_ = s.store.SetSystemSetting(ctx, "installation_id", instID)
	}

	// 2. Get Country (from ENV or default)
	country := os.Getenv("PHAXA_COUNTRY")
	if country == "" {
		country = "unknown"
	}

	// 3. Count Workflows (across all tenants)
	// Passing an empty context to store methods that extract tenantID
	// might fail depending on implementation. We should use a privileged context.
	privCtx := store.WithTenantID(ctx, "") 
	wfCount, err := s.store.CountDefinitions(privCtx)
	if err != nil {
		return nil, fmt.Errorf("count workflows: %w", err)
	}

	// 4. Count Runs
	totalRuns, err := s.store.CountRuns(privCtx, store.RunFilter{})
	if err != nil {
		return nil, fmt.Errorf("count total runs: %w", err)
	}

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)
	runs7d, err := s.store.CountRuns(privCtx, store.RunFilter{StartedFrom: &sevenDaysAgo})
	if err != nil {
		return nil, fmt.Errorf("count runs 7d: %w", err)
	}

	thirtyDaysAgo := now.AddDate(0, 0, -30)
	runs30d, err := s.store.CountRuns(privCtx, store.RunFilter{StartedFrom: &thirtyDaysAgo})
	if err != nil {
		return nil, fmt.Errorf("count runs 30d: %w", err)
	}

	return &Report{
		InstallationID: instID,
		Country:        country,
		WorkflowCount:  wfCount,
		TotalRuns:      totalRuns,
		RunsLast7d:     runs7d,
		RunsLast30d:    runs30d,
		ReportedAt:     now,
	}, nil
}
