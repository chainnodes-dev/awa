package llm

// NewGeminiProvider creates an LLM provider for Google AI Studio (Gemini) models.
// Uses the OpenAI-compatible endpoint.
func NewGeminiProvider(apiKey, defaultModel string, maxTokens int) *OpenAIProvider {
	if defaultModel == "" {
		defaultModel = "gemini-2.0-flash"
	}
	return NewOpenAICompatibleProvider(
		"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		apiKey, "gemini", defaultModel, maxTokens,
	)
}
