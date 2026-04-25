package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Role is the set of RBAC roles recognised by the platform.
type Role string

const (
	RoleSuperAdmin Role = "super_admin" // platform operator; can manage tenants
	RoleAdmin      Role = "admin"       // tenant administrator
	RoleEditor     Role = "editor"      // can define workflows and setup processes
	RoleRunner     Role = "runner"      // can start/trigger runs and approve HITL
	RoleOperator   Role = "operator"    // deprecated; can do both (admin/editor/runner union)
	RoleViewer     Role = "viewer"      // read-only
)

// ValidRole returns true when r is one of the recognised roles.
func ValidRole(r string) bool {
	switch Role(r) {
	case RoleSuperAdmin, RoleAdmin, RoleEditor, RoleRunner, RoleOperator, RoleViewer:
		return true
	}
	return false
}

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	issuer          = "asm-platform"
)

// Claims is the JWT payload for access tokens.
type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	Username string `json:"sub"` // reuse "sub" for username
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService issues and validates access tokens.
type JWTService struct {
	secret []byte
}

// NewJWTService creates a JWTService signed with the given secret.
func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

// GenerateAccessToken creates a signed JWT for the given user.
func (j *JWTService) GenerateAccessToken(userID, tenantID, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

// AccessTokenTTL returns the access token lifetime (exposed for test helpers).
func AccessTokenTTL() time.Duration { return accessTokenTTL }

// RefreshTokenTTL returns the refresh token lifetime.
func RefreshTokenTTL() time.Duration { return refreshTokenTTL }

// ValidateToken parses and validates a signed JWT string.
// Returns the embedded Claims on success.
func (j *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
