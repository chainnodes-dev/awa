package api

import (
	"github.com/gin-gonic/gin"
)

// SecurityMiddleware adds security headers, including Content-Security-Policy.
// It is designed to be permissive in development (allowing 'unsafe-eval' for Vite)
// while being strict in production.
func SecurityMiddleware(env string, origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic security headers
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// CSP is temporarily disabled to debug the 'eval' issue.
		// We suspect an external policy is being enforced.

		c.Next()
	}
}
