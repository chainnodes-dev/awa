package store

import (
	"context"
	"time"
)

// LLMConfig holds provider settings for a single LLM provider within a tenant.
type LLMConfig struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Provider     string    `json:"provider"`
	APIKey       string    `json:"api_key"`
	BaseURL      string    `json:"base_url,omitempty"`
	DefaultModel    string    `json:"default_model"`
	MaxOutputTokens int       `json:"max_output_tokens"`
	Enabled         bool      `json:"enabled"`
	IsDefault       bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LLMConfigStore manages per-tenant LLM provider configuration.
type LLMConfigStore interface {
	UpsertLLMConfig(ctx context.Context, cfg *LLMConfig) error
	GetLLMConfig(ctx context.Context, provider string) (*LLMConfig, error)
	ListLLMConfigs(ctx context.Context) ([]*LLMConfig, error)
	DeleteLLMConfig(ctx context.Context, provider string) error
	SetDefaultProvider(ctx context.Context, provider string) error
	HasAnyLLMConfig(ctx context.Context) (bool, error)
}
