package llm

import (
	"context"
	"fmt"
	"sync"
)

// Provider is the LLM-agnostic interface for all language model calls.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	// Stream sends tokens to the tokenCh as they arrive. Caller must drain the channel.
	Stream(ctx context.Context, req CompletionRequest, tokenCh chan<- string) (*CompletionResponse, error)
	Name() string
	MaxOutputTokens() int
}

type CompletionRequest struct {
	Model        string
	SystemPrompt string
	Messages     []Message
	Tools        []Tool
	MaxTokens    int
	Temperature  float64
}

type Message struct {
	Role       string      `json:"role"`                   // "user" | "assistant" | "tool"
	Content    interface{} `json:"content"`                // string or []ContentBlock
	ToolCallID string      `json:"tool_call_id,omitempty"` // OpenAI: identifies the tool call being responded to
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`   // OpenAI: assistant-side tool invocations
}

type ContentBlock struct {
	Type      string      `json:"type"`                  // "text" | "tool_use" | "tool_result"
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`          // tool_use: unique call ID
	ToolUseID string      `json:"tool_use_id,omitempty"` // tool_result: references the tool_use ID (Anthropic)
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	Output    interface{} `json:"content,omitempty"` // tool_result content
	Reasoning string      `json:"reasoning,omitempty"` // DeepSeek: thought content
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type CompletionResponse struct {
	Content    string      `json:"content"`
	Reasoning  string      `json:"reasoning,omitempty"` // DeepSeek reasoning_content
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	StopReason string      `json:"stop_reason"`
	Usage      *TokenUsage `json:"usage,omitempty"`
}

type ToolCall struct {
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Registry holds registered LLM providers. All methods are thread-safe so the
// registry can be hot-reloaded at runtime when the UI saves new LLM config.
type Registry struct {
	mu              sync.RWMutex
	providers       map[string]Provider
	defaultProvider string
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("LLM provider '%s' not registered", name)
	}
	return p, nil
}

// First returns an arbitrary registered provider, or nil if none are registered.
func (r *Registry) First() Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.providers {
		return p
	}
	return nil
}

// Default returns the name of the global default provider.
func (r *Registry) Default() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultProvider
}

// SetDefault sets the global default provider name.
func (r *Registry) SetDefault(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultProvider = name
}

// Snapshot returns a shallow copy of the current providers map.
func (r *Registry) Snapshot() map[string]Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]Provider, len(r.providers))
	for k, v := range r.providers {
		out[k] = v
	}
	return out
}

// Reload atomically replaces the entire provider map and default name.
// Safe to call concurrently with Get/Default.
func (r *Registry) Reload(providers map[string]Provider, defaultName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = providers
	r.defaultProvider = defaultName
}
