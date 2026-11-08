package llm

// ModelProfile defines a semantic profile mapped to a specific provider and model.
type ModelProfile struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Badge       string `json:"badge,omitempty"`
}

// DefaultModelProfiles returns the standard built-in semantic profiles.
func DefaultModelProfiles() map[string]ModelProfile {
	return map[string]ModelProfile{
		"fast": {
			ID:          "fast",
			Provider:    "openrouter",
			Model:       "~deepseek/deepseek-flash-latest",
			Label:       "Fast & Efficient",
			Description: "High-throughput classification, routing & structured extraction",
			Badge:       "Fast",
		},
		"reasoning": {
			ID:          "reasoning",
			Provider:    "openrouter",
			Model:       "deepseek/deepseek-r1",
			Label:       "Deep Reasoning",
			Description: "Complex planning, multi-step deduction & architectural decisions",
			Badge:       "Reasoning",
		},
		"coding": {
			ID:          "coding",
			Provider:    "openrouter",
			Model:       "qwen/qwen-2.5-coder-32b-instruct",
			Label:       "Code & Transform",
			Description: "Code generation, SQL querying & complex data transforms",
			Badge:       "Coding",
		},
		"general": {
			ID:          "general",
			Provider:    "openrouter",
			Model:       "~deepseek/deepseek-flash-latest",
			Label:       "Default Generalist",
			Description: "High-capability reasoning for general tool calling & orchestration",
			Badge:       "Flagship",
		},
		"local": {
			ID:          "local",
			Provider:    "ollama",
			Model:       "llama3.2",
			Label:       "On-Prem / Local",
			Description: "Zero data-leakage execution on local hardware via Ollama",
			Badge:       "Private",
		},
	}
}

// DefaultDesignerProfiles returns the standard tier profiles for business process generation.
func DefaultDesignerProfiles() map[string]ModelProfile {
	return map[string]ModelProfile{
		"frontier": {
			ID:          "frontier",
			Provider:    "openrouter",
			Model:       "anthropic/claude-sonnet-4.5",
			Label:       "Frontier",
			Description: "Maximum intelligence & reasoning for complex enterprise workflows",
			Badge:       "Frontier",
		},
		"balanced": {
			ID:          "balanced",
			Provider:    "openrouter",
			Model:       "google/gemini-2.0-flash-001",
			Label:       "Balanced",
			Description: "Optimal speed, quality & cost balance for standard workflows",
			Badge:       "Default",
		},
		"fast": {
			ID:          "fast",
			Provider:    "openrouter",
			Model:       "~deepseek/deepseek-flash-latest",
			Label:       "Fast",
			Description: "Rapid prototyping & simple workflows with ultra-low latency",
			Badge:       "Fast",
		},
	}
}

