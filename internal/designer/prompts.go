package designer

import (
	"sync"
)

// PromptManager handles the storage and retrieval of system prompts used by
// the AI workflow generator and skill analyser.
type PromptManager struct {
	mu      sync.RWMutex
	prompts map[string]string
}

// NewPromptManager creates a manager with default prompts.
func NewPromptManager() *PromptManager {
	pm := &PromptManager{
		prompts: make(map[string]string),
	}
	return pm
}

// Get returns the prompt for the given ID, falling back to an empty string
// if no override exists. The caller should provide the hardcoded default.
func (pm *PromptManager) Get(id string, defaultPrompt string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if custom, ok := pm.prompts[id]; ok && custom != "" {
		return custom
	}
	return defaultPrompt
}

// Set overrides the prompt for the given ID.
func (pm *PromptManager) Set(id, content string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.prompts[id] = content
}

// GetAll returns all overrides.
func (pm *PromptManager) GetAll() map[string]string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	out := make(map[string]string, len(pm.prompts))
	for k, v := range pm.prompts {
		out[k] = v
	}
	return out
}

const (
	PromptIDBase     = "workflow_generator_base"
	PromptIDSkill    = "skill_analyser_preamble"
	PromptIDRefine   = "workflow_refinement_addendum"

	// Pipeline step prompts — each maps to one stage of the multi-step generator.
	PromptIDDecompose      = "workflow_decompose"
	PromptIDCategorise     = "workflow_categorise"
	PromptIDWire           = "workflow_wire"
	PromptIDImplFinish     = "workflow_implement_finish"
)
