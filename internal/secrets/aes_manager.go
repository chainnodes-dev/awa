package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AESSecretManager implements SecretManager using AES-256-GCM and a PostgreSQL database.
type AESSecretManager struct {
	pool *pgxpool.Pool
	key  []byte
}

var ErrSecretNotFound = errors.New("secret not found")

// NewAESSecretManager initializes a new AESSecretManager.
// If the key is less than 32 bytes, it returns an error.
func NewAESSecretManager(pool *pgxpool.Pool, key []byte) (*AESSecretManager, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("AES-256 requires a 32-byte key")
	}
	return &AESSecretManager{
		pool: pool,
		key:  key,
	}, nil
}

// GenerateOrLoadMasterKey checks for an existing .asm_master_key file or ASM_MASTER_KEY env var.
// If neither exists, it generates a new 32-byte key, saves it to .asm_master_key, and returns it.
func GenerateOrLoadMasterKey() ([]byte, error) {
	if envKey := os.Getenv("ASM_MASTER_KEY"); envKey != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(envKey)
		if err == nil && len(keyBytes) == 32 {
			return keyBytes, nil
		}
		// If it's a raw 32-char string
		if len(envKey) == 32 {
			return []byte(envKey), nil
		}
	}

	keyPath := ".asm_master_key"
	if b, err := os.ReadFile(keyPath); err == nil {
		keyBytes, err := base64.StdEncoding.DecodeString(string(b))
		if err == nil && len(keyBytes) == 32 {
			return keyBytes, nil
		}
		if len(b) == 32 {
			return b, nil
		}
	}

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(newKey)
	if err := os.WriteFile(keyPath, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("failed to write master key to file: %w", err)
	}

	return newKey, nil
}

func (m *AESSecretManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (m *AESSecretManager) decrypt(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesgcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (m *AESSecretManager) GetSecret(ctx context.Context, tenantID, key string) (string, error) {
	var encrypted string
	err := m.pool.QueryRow(ctx, `SELECT encrypted_value FROM tenant_secrets WHERE tenant_id=$1 AND key=$2`, tenantID, key).Scan(&encrypted)
	if err != nil {
		return "", ErrSecretNotFound
	}
	return m.decrypt(encrypted)
}

func (m *AESSecretManager) SetSecret(ctx context.Context, tenantID, key, value string) error {
	encrypted, err := m.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	_, err = m.pool.Exec(ctx, `
		INSERT INTO tenant_secrets (tenant_id, key, encrypted_value)
		VALUES ($1, $2, $3)
		ON CONFLICT (tenant_id, key) DO UPDATE SET encrypted_value = EXCLUDED.encrypted_value, updated_at = CURRENT_TIMESTAMP`,
		tenantID, key, encrypted,
	)
	return err
}

func (m *AESSecretManager) DeleteSecret(ctx context.Context, tenantID, key string) error {
	_, err := m.pool.Exec(ctx, `DELETE FROM tenant_secrets WHERE tenant_id=$1 AND key=$2`, tenantID, key)
	return err
}

func (m *AESSecretManager) ListSecrets(ctx context.Context, tenantID string) ([]string, error) {
	rows, err := m.pool.Query(ctx, `SELECT key FROM tenant_secrets WHERE tenant_id=$1 ORDER BY key ASC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}
