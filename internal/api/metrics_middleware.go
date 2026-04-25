package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/metrics"
	"github.com/asm-platform/asm/internal/store"
)

// MetricsMiddleware records Prometheus HTTP metrics for every request.
// It uses c.FullPath() (the route template, e.g. "/api/v1/runs/:id") rather
// than the raw URL path to keep cardinality bounded.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		path := c.FullPath()
		if path == "" {
			// 404 — no matched route; use a fixed label to avoid high cardinality.
			path = "unmatched"
		}

		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())
		elapsed := time.Since(start).Seconds()

		tenantID := store.TenantIDFromContext(c.Request.Context())
		if tenantID == "" {
			tenantID = "unknown"
		}

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status, tenantID).Inc()
		metrics.HTTPRequestDurationSeconds.WithLabelValues(method, path, tenantID).Observe(elapsed)
	}
}
