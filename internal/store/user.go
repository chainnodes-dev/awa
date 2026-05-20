package store

import (
	"context"
	"time"
)

// Tenant is a platform tenant (one per customer / deployment unit).
type Tenant struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Slug         string            `json:"slug"` // URL-safe identifier, e.g. "acme-corp"
	LogoURL      string    `json:"logo_url,omitempty"`
	LicenseToken string    `json:"license_token,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// IDPConfig represents external auth settings for a tenant.
type IDPConfig struct {
	TenantID     string            `json:"tenant_id"`
	IssuerURL    string            `json:"issuer_url"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"-"`
	RoleMapping  map[string]string `json:"role_mapping"`
	Active       bool              `json:"active"`
}

// AuditLog represents a single recorded action for auditing.
type AuditLog struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ip_address"`
	CreatedAt time.Time              `json:"created_at"`
}

// User is the platform user type. PasswordHash is never serialised to JSON.
type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "super_admin" | "admin" | "operator" | "viewer"
	CreatedAt    time.Time `json:"created_at"`
}

// RefreshToken represents a stored, hashed refresh token.
// The raw token is never persisted; only its SHA-256 hex hash is stored.
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"` // sha256 hex; never serialised
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantStore manages platform tenants.
type TenantStore interface {
	CreateTenant(ctx context.Context, t *Tenant) error
	GetTenant(ctx context.Context, id string) (*Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	UpdateTenantLicense(ctx context.Context, id, token string) error
	GetTenantIDPConfig(ctx context.Context, tenantID string) (*IDPConfig, error)
	UpsertTenantIDPConfig(ctx context.Context, cfg *IDPConfig) error
	ListTenants(ctx context.Context) ([]*Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
	HasAnyTenant(ctx context.Context) (bool, error)
	UpdateTenantBranding(ctx context.Context, id, name, logoURL string) error

	RecordAuditLog(ctx context.Context, entry *AuditLog) error
	ListAuditLogs(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error)
}

// UserFilter controls which users are returned by ListUsers.
type UserFilter struct {
	Limit  int // 0 = no limit
	Offset int
}

// UserStore manages platform users.
type UserStore interface {
	CreateUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateUserRole(ctx context.Context, id, role string) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context, filter UserFilter) ([]*User, error)
	// HasAnyUser returns true if at least one user row exists.
	// Used at startup to determine whether to seed the default admin.
	HasAnyUser(ctx context.Context) (bool, error)
}

// RefreshTokenStore manages opaque refresh tokens.
type RefreshTokenStore interface {
	CreateRefreshToken(ctx context.Context, t *RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
}
