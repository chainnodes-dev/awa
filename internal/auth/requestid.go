package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/asm-platform/asm/internal/logger"
)

const requestIDHeader = "X-Request-ID"

// RequestID is a Gin middleware that ensures every request has a unique ID.
// It reads X-Request-ID from the incoming request (useful for client-generated
// IDs or upstream proxies), or generates a new UUID when absent. The ID is:
//   - Written back in the X-Request-ID response header
//   - Stored in the Go request context via logger.WithRequestID so downstream
//     slog calls can attach it automatically with logger.FromContext(ctx)
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(requestIDHeader)
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Header(requestIDHeader, rid)

		ctx := logger.WithRequestID(c.Request.Context(), rid)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
