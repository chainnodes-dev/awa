// Package metrics defines and registers all Prometheus metrics for the Phaxa platform.
//
// All metrics use the "phaxa_" prefix and are registered on the default registry,
// which is exposed at GET /metrics by the Prometheus HTTP handler.
//
// Metric naming follows the Prometheus conventions:
//   - Counters end in _total
//   - Histograms end in _seconds (for durations) or _bytes (for sizes)
//   - Gauges have no suffix convention
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ── Run lifecycle ─────────────────────────────────────────────────────────────

// RunsStartedTotal counts workflow runs started, labelled by workflow name and version.
var RunsStartedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_runs_started_total",
	Help: "Total number of workflow runs started.",
}, []string{"workflow_name", "workflow_version", "tenant_id"})

// RunsCompletedTotal counts runs that reached a terminal state successfully.
// The final_state label holds the terminal state name (e.g. APPROVED, REJECTED).
var RunsCompletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_runs_completed_total",
	Help: "Total number of workflow runs completed (terminal state reached).",
}, []string{"workflow_name", "workflow_version", "final_state", "tenant_id"})

// RunsFailedTotal counts runs that ended in an unrecoverable error.
var RunsFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_runs_failed_total",
	Help: "Total number of workflow runs that failed with an error.",
}, []string{"workflow_name", "workflow_version", "tenant_id"})

// RunsActive is the current number of in-flight runs (direct mode only).
var RunsActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "phaxa_runs_active",
	Help: "Current number of active (running or waiting) workflow runs.",
}, []string{"workflow_name", "tenant_id"})

// RunDurationSeconds tracks end-to-end run duration from start to completion/failure.
var RunDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "phaxa_run_duration_seconds",
	Help:    "End-to-end duration of workflow runs in seconds.",
	Buckets: prometheus.ExponentialBuckets(0.1, 3, 10), // 0.1s → ~2h
}, []string{"workflow_name", "workflow_version", "status", "tenant_id"})

// ── State transitions ─────────────────────────────────────────────────────────

// StateTransitionsTotal counts every state transition.
var StateTransitionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_state_transitions_total",
	Help: "Total number of state transitions recorded.",
}, []string{"workflow_name", "from_state", "to_state", "trigger", "tenant_id"})

// ── Agent / executor ──────────────────────────────────────────────────────────

// AgentExecutionsTotal counts agent executions with a result label (success/error).
var AgentExecutionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_agent_executions_total",
	Help: "Total number of agent executions dispatched.",
}, []string{"workflow_name", "agent_name", "result", "tenant_id"})

// AgentDurationSeconds records how long each agent execution takes.
var AgentDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "phaxa_agent_duration_seconds",
	Help:    "Duration of individual agent executions in seconds.",
	Buckets: prometheus.ExponentialBuckets(0.05, 2.5, 10), // 50ms → ~1h
}, []string{"workflow_name", "agent_name", "tenant_id"})

// ── Human-in-the-loop ─────────────────────────────────────────────────────────

// HITLRequestsTotal counts HITL requests created.
var HITLRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_hitl_requests_total",
	Help: "Total number of human-in-the-loop requests created.",
}, []string{"workflow_name", "state_name", "tenant_id"})

// HITLResolutionDurationSeconds tracks how long humans take to resolve a HITL request.
// The resolution label holds the value provided by the resolver (e.g. approve/reject).
var HITLResolutionDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "phaxa_hitl_resolution_duration_seconds",
	Help:    "Time from HITL request creation to human resolution in seconds.",
	Buckets: prometheus.ExponentialBuckets(60, 3, 10), // 1min → ~50h
}, []string{"workflow_name", "resolution", "tenant_id"})

// ── HTTP ──────────────────────────────────────────────────────────────────────

// HTTPRequestsTotal counts HTTP requests handled by the API server.
var HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "phaxa_http_requests_total",
	Help: "Total number of HTTP requests handled by the API server.",
}, []string{"method", "path", "status_code", "tenant_id"})

// HTTPRequestDurationSeconds records API handler latency.
var HTTPRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "phaxa_http_request_duration_seconds",
	Help:    "HTTP request duration in seconds.",
	Buckets: prometheus.DefBuckets,
}, []string{"method", "path", "tenant_id"})
