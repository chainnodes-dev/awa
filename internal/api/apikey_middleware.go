package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/auth"
	"github.com/asm-platform/asm/internal/store"
)

// hashAPIKey returns the hex-encoded SHA-256 hash of a raw API key string.
func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// RequireAuthOrAPIKey accepts either a standard JWT Bearer token or an API key.
// API keys may be supplied via the X-API-Key header or as a Bearer token with
// the "phaxa_sk_" prefix (e.g. "Authorization: Bearer phaxa_sk_...").
//
// On success the tenant ID is injected into the request context identically to
// RequireAuth so that all downstream store calls are correctly scoped.
func RequireAuthOrAPIKey(jwtSvc *auth.JWTService, s store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract a candidate API key from the request.
		rawKey := c.GetHeader("X-API-Key")
		if rawKey == "" {
			if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer phaxa_sk_") {
				rawKey = strings.TrimPrefix(h, "Bearer ")
			}
		}

		if rawKey != "" {
			hash := hashAPIKey(rawKey)
			key, err := s.GetAPIKeyByHash(c.Request.Context(), hash)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or revoked API key"})
				return
			}
			// Update last_used_at asynchronously — non-blocking, non-fatal.
			go func() { _ = s.TouchAPIKey(context.Background(), key.ID) }()

			ctx := store.WithTenantID(c.Request.Context(), key.TenantID)
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			return
		}

		// 2. Fall back to standard JWT auth.
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtSvc.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("auth_claims", claims)
		ctx := store.WithTenantID(c.Request.Context(), claims.TenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
