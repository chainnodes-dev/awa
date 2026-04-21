package llm

import "fmt"

// NewOllamaProvider creates an LLM provider that talks to a local Ollama server.
//
// baseURL is the root of the Ollama HTTP API, e.g. "http://mintbox:11434".
// The OpenAI-compatible endpoint (/v1/chat/completions) is appended automatically.
//
// defaultModel is the Ollama model tag to use when the workflow does not specify
// one, e.g. "llama3.2" or "llama3.2:3b".
//
// Ollama does not require an API key; the Authorization header is omitted.
// Tool calling is supported on models that implement it (llama3.1+, llama3.2 11B+).
// Smaller models (llama3.2 3B) may return tool calls inconsistently — use them
// for simple text-output states and rely on the structured-output parsing fallback.
func NewOllamaProvider(baseURL, defaultModel string, maxTokens int) *OpenAIProvider {
	if defaultModel == "" {
		defaultModel = "llama3.2"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	endpoint := fmt.Sprintf("%s/v1/chat/completions", baseURL)
	return NewOpenAICompatibleProvider(endpoint, "", "ollama", defaultModel, maxTokens)
}
