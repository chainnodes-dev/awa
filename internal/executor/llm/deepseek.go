package llm

// NewDeepseekProvider creates an LLM provider for Deepseek models.
func NewDeepseekProvider(apiKey, defaultModel string, maxTokens int) *OpenAIProvider {
	if defaultModel == "" {
		defaultModel = "deepseek-chat"
	}
	return NewOpenAICompatibleProvider("https://api.deepseek.com/v1/chat/completions", apiKey, "deepseek", defaultModel, maxTokens)
}
