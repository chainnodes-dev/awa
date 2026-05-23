package llm

// NewGrokProvider creates an LLM provider for xAI's Grok models.
// Uses the OpenAI-compatible endpoint at api.x.ai.
func NewGrokProvider(apiKey, defaultModel string, maxTokens int) *OpenAIProvider {
	if defaultModel == "" {
		defaultModel = "grok-3"
	}
	return NewOpenAICompatibleProvider("https://api.x.ai/v1/chat/completions", apiKey, "grok", defaultModel, maxTokens)
}
