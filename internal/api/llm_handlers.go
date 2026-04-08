package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/store"
)

// ListLLMConfigs returns all provider configs for the current tenant. API keys are masked.
func (h *Handlers) ListLLMConfigs(c *gin.Context) {
	ctx := store.WithTenantID(c.Request.Context(), store.DefaultTenantID)
	configs, err := h.store.ListLLMConfigs(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, cfg := range configs {
		if cfg.APIKey != "" {
			cfg.APIKey = "***"
		}
	}
	c.JSON(http.StatusOK, configs)
}

type upsertLLMConfigRequest struct {
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url"`
	DefaultModel string `json:"default_model"`
	Enabled      bool   `json:"enabled"`
}

// UpsertLLMConfig creates or updates a provider's configuration.
func (h *Handlers) UpsertLLMConfig(c *gin.Context) {
	provider := c.Param("provider")
	var req upsertLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := store.WithTenantID(c.Request.Context(), store.DefaultTenantID)

	// Preserve existing key if the client sends the masked placeholder or empty string.
	apiKey := req.APIKey
	if apiKey == "***" || apiKey == "" {
		if existing, err := h.store.GetLLMConfig(ctx, provider); err == nil {
			apiKey = existing.APIKey
		} else {
			apiKey = ""
		}
	}

	cfg := &store.LLMConfig{
		TenantID:     store.DefaultTenantID,
		Provider:     provider,
		APIKey:       apiKey,
		BaseURL:      req.BaseURL,
		DefaultModel: req.DefaultModel,
		Enabled:      req.Enabled,
	}
	if err := h.store.UpsertLLMConfig(ctx, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.reloadLLMRegistry(ctx)
	c.Status(http.StatusNoContent)
}

// DeleteLLMConfig removes a provider's configuration.
func (h *Handlers) DeleteLLMConfig(c *gin.Context) {
	provider := c.Param("provider")
	ctx := store.WithTenantID(c.Request.Context(), store.DefaultTenantID)
	if err := h.store.DeleteLLMConfig(ctx, provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadLLMRegistry(ctx)
	c.Status(http.StatusNoContent)
}

type setDefaultRequest struct {
	Provider string `json:"provider" binding:"required"`
}

// SetDefaultProvider sets the global default LLM provider for the tenant.
func (h *Handlers) SetDefaultProvider(c *gin.Context) {
	var req setDefaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := store.WithTenantID(c.Request.Context(), store.DefaultTenantID)
	if err := h.store.SetDefaultProvider(ctx, req.Provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.reloadLLMRegistry(ctx)
	c.Status(http.StatusNoContent)
}

// TestLLMConnection sends a minimal completion request to verify the provider works.
func (h *Handlers) TestLLMConnection(c *gin.Context) {
	provider := c.Param("provider")
	ctx := store.WithTenantID(c.Request.Context(), store.DefaultTenantID)

	cfg, err := h.store.GetLLMConfig(ctx, provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "provider not configured"})
		return
	}

	model := cfg.DefaultModel
	if model == "" {
		if def, ok := llm.ProviderDefaultModels[provider]; ok {
			model = def
		}
	}

	var prov llm.Provider
	switch provider {
	case "anthropic":
		prov = llm.NewAnthropicProvider(cfg.APIKey, model)
	case "ollama":
		prov = llm.NewOllamaProvider(cfg.BaseURL, model)
	default:
		endpoint := cfg.BaseURL
		if endpoint == "" {
			endpoint = llm.ProviderEndpoint(provider)
		}
		prov = llm.NewOpenAICompatibleProvider(endpoint, cfg.APIKey, provider, model)
	}

	_, testErr := prov.Complete(c.Request.Context(), llm.CompletionRequest{
		Model:     model,
		MaxTokens: 5,
		Messages:  []llm.Message{{Role: "user", Content: "Say 'ok'"}},
	})
	if testErr != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": testErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// reloadLLMRegistry rebuilds the in-process LLM registry from DB after a config change.
func (h *Handlers) reloadLLMRegistry(ctx context.Context) {
	if h.llmRegistry == nil {
		return
	}
	reg, defaultName, err := llm.BuildRegistryFromDB(ctx, h.store)
	if err != nil {
		slog.Warn("Failed to reload LLM registry", "error", err)
		return
	}
	h.llmRegistry.Reload(reg.Snapshot(), defaultName)
	slog.Info("LLM registry reloaded", "default", defaultName)
}
