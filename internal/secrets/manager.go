package secrets

import (
	"context"
)

// SecretManager defines the interface for storing and retrieving tenant secrets securely.
type SecretManager interface {
	GetSecret(ctx context.Context, tenantID, key string) (string, error)
	SetSecret(ctx context.Context, tenantID, key, value string) error
	DeleteSecret(ctx context.Context, tenantID, key string) error
	ListSecrets(ctx context.Context, tenantID string) ([]string, error)
}
