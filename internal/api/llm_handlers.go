package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/store"
)

// ListLLMConfigs returns all provider configs for the current tenant. API keys are masked.
func (h *Handlers) ListLLMConfigs(c *gin.Context) {
	ctx := c.Request.Context()
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
	APIKey          string `json:"api_key"`
	BaseURL         string `json:"base_url"`
	DefaultModel    string `json:"default_model"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	Enabled         bool   `json:"enabled"`
}

// UpsertLLMConfig creates or updates a provider's configuration.
func (h *Handlers) UpsertLLMConfig(c *gin.Context) {
	provider := c.Param("provider")
	var req upsertLLMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	tenantID := store.TenantIDFromContext(ctx)
	if tenantID == "" {
		tenantID = store.DefaultTenantID
	}

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
		TenantID:        tenantID,
		Provider:        provider,
		APIKey:          apiKey,
		BaseURL:         req.BaseURL,
		DefaultModel:    req.DefaultModel,
		MaxOutputTokens: req.MaxOutputTokens,
		Enabled:         req.Enabled,
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
	ctx := c.Request.Context()
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
	ctx := c.Request.Context()
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
	ctx := c.Request.Context()

	cfg, err := h.store.GetLLMConfig(ctx, provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "provider not configured"})
		return
	}

	model := cfg.DefaultModel
	if model == "" || (provider == "openrouter" && model == "anthropic/claude-3.5-sonnet") {
		if def, ok := llm.ProviderDefaultModels[provider]; ok {
			model = def
		}
	}
	if provider == "openrouter" && model == "deepseek/deepseek-flash-latest" {
		model = "~deepseek/deepseek-flash-latest"
	}

	var prov llm.Provider
	switch provider {
	case "anthropic":
		prov = llm.NewAnthropicProvider(cfg.APIKey, model, cfg.MaxOutputTokens)
	case "ollama":
		prov = llm.NewOllamaProvider(cfg.BaseURL, model, cfg.MaxOutputTokens)
	default:
		endpoint := cfg.BaseURL
		if endpoint == "" {
			endpoint = llm.ProviderEndpoint(provider)
		}
		prov = llm.NewOpenAICompatibleProvider(endpoint, cfg.APIKey, provider, model, cfg.MaxOutputTokens)
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

	// Also rebuild the AI generator if we have an appropriate cloud default provider.
	generatorProvider := defaultName
	if generatorProvider == "ollama" {
		generatorProvider = ""
		// Look for any other enabled non-ollama provider in the registry.
		for name := range reg.Snapshot() {
			if name != "ollama" {
				generatorProvider = name
				break
			}
		}
	}

	if generatorProvider != "" {
		slog.Info("Attempting to re-initialize AI generator", "provider", generatorProvider)
		if prov, err := h.llmRegistry.Get(generatorProvider); err == nil {
			h.generator = designer.NewGenerator(prov, h.mcpEntries, h.store)
			slog.Info("AI workflow generator successfully re-initialized", "provider", generatorProvider)
		} else {
			slog.Warn("Failed to get provider from registry during reload", "provider", generatorProvider, "error", err)
			h.generator = nil
		}
	} else {
		h.generator = nil
		slog.Info("AI workflow generator disabled (no cloud default provider configured)")
	}
}

// ListLocalModels returns the list of models downloaded in the local Ollama instance.
func (h *Handlers) ListLocalModels(c *gin.Context) {
	ctx := c.Request.Context()
	cfg, err := h.store.GetLLMConfig(ctx, "ollama")
	baseURL := "http://localhost:11434"
	if err == nil && cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	tagsURL := baseURL + "/api/tags"
	httpReq, err := http.NewRequestWithContext(ctx, "GET", tagsURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to contact Ollama: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("Ollama returned error status: %d", resp.StatusCode)})
		return
	}

	var tags struct {
		Models []interface{} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse Ollama response: %v", err)})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// PullLocalModel triggers downloading model weights from Ollama and streams progress updates.
func (h *Handlers) PullLocalModel(c *gin.Context) {
	var req struct {
		Model string `json:"model" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	cfg, err := h.store.GetLLMConfig(ctx, "ollama")
	baseURL := "http://localhost:11434"
	if err == nil && cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	pullURL := baseURL + "/api/pull"
	payload, err := json.Marshal(map[string]string{"name": req.Model})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", pullURL, bytes.NewReader(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to contact Ollama: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("Ollama returned error status: %d", resp.StatusCode)})
		return
	}

	// Set headers for streaming response
	c.Header("Content-Type", "application/json-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Status(http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				slog.Error("error reading ollama pull stream", "err", err)
			}
			break
		}
		_, writeErr := c.Writer.Write(line)
		if writeErr != nil {
			slog.Warn("client disconnected from ollama pull stream", "err", writeErr)
			break
		}
		c.Writer.Flush()
	}
}

// DeleteLocalModel deletes model weights from local Ollama cache.
func (h *Handlers) DeleteLocalModel(c *gin.Context) {
	model := c.Query("model")
	if model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model query parameter is required"})
		return
	}

	ctx := c.Request.Context()
	cfg, err := h.store.GetLLMConfig(ctx, "ollama")
	baseURL := "http://localhost:11434"
	if err == nil && cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	deleteURL := baseURL + "/api/delete"
	payload, err := json.Marshal(map[string]string{"name": model})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, bytes.NewReader(payload))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to contact Ollama: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("Ollama returned error status: %d", resp.StatusCode)})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetModelProfiles returns the semantic model profiles configured for the platform.
func (h *Handlers) GetModelProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	profiles := llm.DefaultModelProfiles()

	if raw, err := h.store.GetSystemSetting(ctx, "llm_model_profiles"); err == nil && raw != "" {
		var custom map[string]llm.ModelProfile
		if err := json.Unmarshal([]byte(raw), &custom); err == nil {
			for k, v := range custom {
				if v.Provider == "openrouter" {
					switch v.Model {
					case "anthropic/claude-3.5-sonnet", "", "deepseek/deepseek-flash-latest":
						v.Model = "~deepseek/deepseek-flash-latest"
					}
				}
				profiles[k] = v
			}
		}
	}

	c.JSON(http.StatusOK, profiles)
}

// SetModelProfiles updates the semantic model profiles for the platform.
func (h *Handlers) SetModelProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	var req map[string]llm.ModelProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate and merge with defaults
	defaults := llm.DefaultModelProfiles()
	merged := make(map[string]llm.ModelProfile)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range req {
		if v.ID == "" {
			v.ID = k
		}
		merged[k] = v
	}

	data, err := json.Marshal(merged)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SetSystemSetting(ctx, "llm_model_profiles", string(data)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "profiles": merged})
}

// GetDesignerProfiles returns the workflow generation profiles configured for the platform.
func (h *Handlers) GetDesignerProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	profiles := llm.DefaultDesignerProfiles()

	if raw, err := h.store.GetSystemSetting(ctx, "llm_designer_profiles"); err == nil && raw != "" {
		var custom map[string]llm.ModelProfile
		if err := json.Unmarshal([]byte(raw), &custom); err == nil {
			for k, v := range custom {
				profiles[k] = v
			}
		}
	}

	c.JSON(http.StatusOK, profiles)
}

// SetDesignerProfiles updates the workflow generation profiles.
func (h *Handlers) SetDesignerProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	var req map[string]llm.ModelProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	defaults := llm.DefaultDesignerProfiles()
	merged := make(map[string]llm.ModelProfile)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range req {
		if v.ID == "" {
			v.ID = k
		}
		merged[k] = v
	}

	data, err := json.Marshal(merged)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.SetSystemSetting(ctx, "llm_designer_profiles", string(data)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "profiles": merged})
}

// OpenRouterModelInfo contains curated metadata about an OpenRouter model.
type OpenRouterModelInfo struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	ContextLength   int     `json:"context_length"`
	PromptPrice     float64 `json:"prompt_price"`     // per 1M tokens
	CompletionPrice float64 `json:"completion_price"` // per 1M tokens
	Description     string  `json:"description"`
}

var (
	openRouterCacheMu     sync.RWMutex
	openRouterCacheModels []OpenRouterModelInfo
	openRouterCacheExpiry time.Time
)

// ListOpenRouterModels returns available models from OpenRouter (cached with fallback).
func (h *Handlers) ListOpenRouterModels(c *gin.Context) {
	openRouterCacheMu.RLock()
	if len(openRouterCacheModels) > 0 && time.Now().Before(openRouterCacheExpiry) {
		cached := openRouterCacheModels
		openRouterCacheMu.RUnlock()
		c.JSON(http.StatusOK, cached)
		return
	}
	openRouterCacheMu.RUnlock()

	// Fetch from OpenRouter
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", "https://openrouter.ai/api/v1/models", nil)
	if err == nil {
		req.Header.Set("User-Agent", "asm-platform/1.0")
		if cfg, _ := h.store.GetLLMConfig(c.Request.Context(), "openrouter"); cfg != nil && cfg.APIKey != "" && cfg.APIKey != "***" {
			req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		}

		resp, reqErr := client.Do(req)
		if reqErr == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var orResp struct {
				Data []struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					Description   string `json:"description"`
					ContextLength int    `json:"context_length"`
					Pricing       struct {
						Prompt     string `json:"prompt"`
						Completion string `json:"completion"`
					} `json:"pricing"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&orResp); err == nil && len(orResp.Data) > 0 {
				var result []OpenRouterModelInfo
				for _, m := range orResp.Data {
					pPrice, _ := strconv.ParseFloat(m.Pricing.Prompt, 64)
					cPrice, _ := strconv.ParseFloat(m.Pricing.Completion, 64)
					result = append(result, OpenRouterModelInfo{
						ID:              m.ID,
						Name:            m.Name,
						ContextLength:   m.ContextLength,
						PromptPrice:     pPrice * 1000000.0,
						CompletionPrice: cPrice * 1000000.0,
						Description:     m.Description,
					})
				}
				openRouterCacheMu.Lock()
				openRouterCacheModels = result
				openRouterCacheExpiry = time.Now().Add(15 * time.Minute)
				openRouterCacheMu.Unlock()

				c.JSON(http.StatusOK, result)
				return
			}
		}
	}

	// Fallback to cached or curated static list if offline / rate limited
	openRouterCacheMu.RLock()
	if len(openRouterCacheModels) > 0 {
		cached := openRouterCacheModels
		openRouterCacheMu.RUnlock()
		c.JSON(http.StatusOK, cached)
		return
	}
	openRouterCacheMu.RUnlock()

	fallback := []OpenRouterModelInfo{
		{ID: "~deepseek/deepseek-flash-latest", Name: "DeepSeek: DeepSeek Flash Latest", ContextLength: 1048576, PromptPrice: 0.05, CompletionPrice: 0.50, Description: "Redirects to the latest DeepSeek Flash family with 1M context"},
		{ID: "deepseek/deepseek-r1", Name: "DeepSeek: R1 (Reasoning)", ContextLength: 164000, PromptPrice: 0.55, CompletionPrice: 2.19, Description: "Open-weights reasoning model rivaling OpenAI o1"},
		{ID: "anthropic/claude-sonnet-4.5", Name: "Anthropic: Claude Sonnet 4.5", ContextLength: 200000, PromptPrice: 3.0, CompletionPrice: 15.0, Description: "Flagship intelligence for code, reasoning and orchestrations"},
		{ID: "google/gemini-2.0-flash-001", Name: "Google: Gemini 2.0 Flash", ContextLength: 1048576, PromptPrice: 0.10, CompletionPrice: 0.40, Description: "Next-gen multimodal, ultra-fast latency and 1M context"},
		{ID: "openai/gpt-4o", Name: "OpenAI: GPT-4o", ContextLength: 128000, PromptPrice: 2.50, CompletionPrice: 10.0, Description: "Omni model for complex text and vision tasks"},
		{ID: "openai/gpt-4o-mini", Name: "OpenAI: GPT-4o Mini", ContextLength: 128000, PromptPrice: 0.15, CompletionPrice: 0.60, Description: "Fast, cost-effective model for high volume tasks"},
		{ID: "qwen/qwen-2.5-coder-32b-instruct", Name: "Qwen: Qwen 2.5 Coder 32B", ContextLength: 131072, PromptPrice: 0.12, CompletionPrice: 0.12, Description: "Code intelligence model competitive with GPT-4"},
		{ID: "meta-llama/llama-3.3-70b-instruct", Name: "Meta: Llama 3.3 70B Instruct", ContextLength: 131072, PromptPrice: 0.20, CompletionPrice: 0.40, Description: "Open weights industry workhorse with 70B parameters"},
	}
	c.JSON(http.StatusOK, fallback)
}
