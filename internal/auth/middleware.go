package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/store"
)

type authKey string

const claimsKey authKey = "auth_claims"

// WithClaims returns a context.Context with the provided Claims attached.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext retrieves the JWT Claims from a context.Context.
func ClaimsFromContext(ctx context.Context) *Claims {
	if ctx == nil {
		return nil
	}
	claims, _ := ctx.Value(claimsKey).(*Claims)
	return claims
}

// RequireAuth validates the Bearer token in the Authorization header.
// On success it sets the parsed Claims in the gin context AND propagates
// the tenant ID into the request context so store methods can scope queries.
// On failure it aborts the request with HTTP 401.
func RequireAuth(svc *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed Authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := svc.ValidateToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(string(claimsKey), claims)
		// Propagate tenant_id and claims into the standard request context so
		// store methods and audit actions can automatically access them.
		ctx := store.WithTenantID(c.Request.Context(), claims.TenantID)
		ctx = WithClaims(ctx, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequireRole aborts with HTTP 403 if the caller's role is not in the allowed set.
// Must be placed after RequireAuth in the middleware chain.
// super_admin bypasses all role checks — it is implicitly allowed everywhere.
func RequireRole(roles ...Role) gin.HandlerFunc {
	allowed := make(map[Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		claims := ClaimsFrom(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		// super_admin has platform-wide access.
		if Role(claims.Role) == RoleSuperAdmin {
			c.Next()
			return
		}
		if !allowed[Role(claims.Role)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// ClaimsFrom retrieves the JWT Claims injected by RequireAuth.
// Returns nil if RequireAuth has not run or the value is missing.
func ClaimsFrom(c *gin.Context) *Claims {
	v, exists := c.Get(string(claimsKey))
	if !exists {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}
