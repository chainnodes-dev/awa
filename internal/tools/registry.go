package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool is the interface implemented by every capability available to LLM agents.
// Workers are "stupid": they call a tool and return its result — the tool itself
// decides nothing about workflow routing.
type Tool interface {
	// Name returns the unique name the LLM uses to invoke this tool.
	Name() string
	// Description is shown to the LLM to help it decide when to use the tool.
	Description() string
	// InputSchema returns a JSON Schema object describing the expected input.
	InputSchema() json.RawMessage
	// Execute runs the tool with the given JSON input and returns a JSON result.
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

// Registry holds all tools available to agents in the current process.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry. A second registration with the same
// name silently overwrites the first.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns the named tool or an error if it is not registered.
func (r *Registry) Get(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not registered", name)
	}
	return t, nil
}

// Has reports whether a tool is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.tools[name]
	return ok
}

// All returns every registered tool in unspecified order.
func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Merge returns a new registry containing all tools from r plus all tools from
// other. Tools in other win on name collision.
func (r *Registry) Merge(other *Registry) *Registry {
	merged := NewRegistry()
	for _, t := range r.tools {
		merged.Register(t)
	}
	for _, t := range other.tools {
		merged.Register(t)
	}
	return merged
}
