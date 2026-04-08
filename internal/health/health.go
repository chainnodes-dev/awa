// Package health provides HTTP handlers for liveness and readiness probes.
//
// GET /health/live  — always 200 while the process is running (liveness)
// GET /health/ready — 200 only when all configured dependencies are reachable (readiness)
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger is implemented by any dependency that can report its own health.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Checker holds named dependency pingers and exposes HTTP handlers.
type Checker struct {
	deps map[string]Pinger
}

// New creates a Checker. Pass dependency name → Pinger pairs via Add.
func New() *Checker {
	return &Checker{deps: make(map[string]Pinger)}
}

// Add registers a named dependency for readiness checks.
func (c *Checker) Add(name string, p Pinger) {
	c.deps[name] = p
}

// Live is a lightweight liveness handler — always returns 200 while the
// process is up. Kubernetes uses this to decide whether to restart a pod.
func (c *Checker) Live(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready checks all registered dependencies. Returns 200 with per-dep status
// when everything is healthy, or 503 with failure details if any dep is down.
// Kubernetes uses this to decide whether to send traffic to the pod.
func (c *Checker) Ready(ctx *gin.Context) {
	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), 3*time.Second)
	defer cancel()

	results := make(map[string]string, len(c.deps))
	allOK := true

	for name, p := range c.deps {
		if err := p.Ping(reqCtx); err != nil {
			results[name] = err.Error()
			allOK = false
		} else {
			results[name] = "ok"
		}
	}

	status := http.StatusOK
	overall := "ok"
	if !allOK {
		status = http.StatusServiceUnavailable
		overall = "degraded"
	}

	ctx.JSON(status, gin.H{
		"status": overall,
		"checks": results,
	})
}
