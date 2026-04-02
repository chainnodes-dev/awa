package store

import (
	"context"
	"time"
)

// User is the platform user type. PasswordHash is never serialised to JSON.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // "admin" | "operator" | "viewer"
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

// UserStore manages platform users.
type UserStore interface {
	CreateUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	UpdateUserRole(ctx context.Context, id, role string) error
	DeleteUser(ctx context.Context, id string) error
	ListUsers(ctx context.Context) ([]*User, error)
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
