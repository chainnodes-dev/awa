package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/store"
)

const claimsKey = "auth_claims"

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
		c.Set(claimsKey, claims)
		// Propagate tenant_id into the standard request context so store
		// methods called with c.Request.Context() are automatically scoped.
		ctx := store.WithTenantID(c.Request.Context(), claims.TenantID)
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
	v, exists := c.Get(claimsKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}
