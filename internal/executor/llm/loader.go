package llm

import (
	"context"
	"log/slog"

	"github.com/asm-platform/asm/internal/store"
)

// ProviderDefaultModels maps each provider name to its recommended default model.
var ProviderDefaultModels = map[string]string{
	"anthropic": "claude-sonnet-4-5",
	"openai":    "gpt-4o",
	"grok":      "grok-3",
	"deepseek":  "deepseek-v4-flash",
	"mistral":   "mistral-large-latest",
	"groq":      "llama-3.1-70b-versatile",
	"together":  "meta-llama/Llama-3-70b-chat-hf",
	"fireworks": "accounts/fireworks/models/llama-v3p1-70b-instruct",
	"cohere":    "command-r-plus",
	"qwen":      "qwen-plus",
	"glm":       "glm-4-plus",
	"gemini":    "gemini-2.0-flash",
	"ollama":    "llama3.2",
}

// ProviderEndpoint returns the default API endpoint for OpenAI-compatible providers.
func ProviderEndpoint(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	case "grok":
		return "https://api.x.ai/v1/chat/completions"
	case "deepseek":
		return "https://api.deepseek.com/v1/chat/completions"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
	case "mistral":
		return "https://api.mistral.ai/v1/chat/completions"
	case "groq":
		return "https://api.groq.com/openai/v1/chat/completions"
	case "together":
		return "https://api.together.xyz/v1/chat/completions"
	case "fireworks":
		return "https://api.fireworks.ai/inference/v1/chat/completions"
	case "cohere":
		return "https://api.cohere.com/compatibility/v1/chat/completions"
	case "qwen":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	case "glm":
		return "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	default:
		return ""
	}
}

// BuildRegistryFromDB loads all enabled LLM configs for the tenant in ctx,
// constructs the appropriate Provider for each, and returns a populated Registry
// and the name of the default provider.
func BuildRegistryFromDB(ctx context.Context, s store.LLMConfigStore) (*Registry, string, error) {
	configs, err := s.ListLLMConfigs(ctx)
	if err != nil {
		return nil, "", err
	}

	reg := NewRegistry()
	var defaultName string

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		model := cfg.DefaultModel
		if model == "" {
			model = ProviderDefaultModels[cfg.Provider]
		}

		var prov Provider
		switch cfg.Provider {
		case "anthropic":
			if cfg.APIKey == "" {
				continue
			}
			prov = NewAnthropicProvider(cfg.APIKey, model, cfg.MaxOutputTokens)
		case "ollama":
			if cfg.BaseURL == "" {
				continue
			}
			prov = NewOllamaProvider(cfg.BaseURL, model, cfg.MaxOutputTokens)
		default:
			// openai, grok, deepseek, gemini — all OpenAI-compatible
			if cfg.APIKey == "" {
				continue
			}
			endpoint := cfg.BaseURL
			if endpoint == "" {
				endpoint = ProviderEndpoint(cfg.Provider)
			}
			prov = NewOpenAICompatibleProvider(endpoint, cfg.APIKey, cfg.Provider, model, cfg.MaxOutputTokens)
		}

		reg.Register(prov)
		slog.Info("LLM provider loaded from DB", "provider", cfg.Provider, "model", model, "max_tokens", cfg.MaxOutputTokens)

		if cfg.IsDefault {
			defaultName = cfg.Provider
		}
	}

	// If no explicit default, prefer Ollama then first registered.
	if defaultName == "" {
		if _, err := reg.Get("ollama"); err == nil {
			defaultName = "ollama"
		} else if first := reg.First(); first != nil {
			defaultName = first.Name()
		}
	}

	reg.SetDefault(defaultName)
	return reg, defaultName, nil
}
