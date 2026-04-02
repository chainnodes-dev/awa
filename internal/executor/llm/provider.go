package llm

import (
	"context"
	"fmt"
)

// Provider is the LLM-agnostic interface for all language model calls.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	// Stream sends tokens to the tokenCh as they arrive. Caller must drain the channel.
	Stream(ctx context.Context, req CompletionRequest, tokenCh chan<- string) (*CompletionResponse, error)
	Name() string
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
	Role       string      `json:"role"`                  // "user" | "assistant" | "tool"
	Content    interface{} `json:"content"`               // string or []ContentBlock
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
	Output    interface{} `json:"content,omitempty"`     // tool_result content
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type CompletionResponse struct {
	Content    string       `json:"content"`
	ToolCalls  []ToolCall   `json:"tool_calls,omitempty"`
	StopReason string       `json:"stop_reason"`
	Usage      *TokenUsage  `json:"usage,omitempty"`
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

// Registry holds registered LLM providers.
type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("LLM provider '%s' not registered", name)
	}
	return p, nil
}

// First returns an arbitrary registered provider, or nil if none are registered.
// Used as a fallback when no explicit provider is configured.
func (r *Registry) First() Provider {
	for _, p := range r.providers {
		return p
	}
	return nil
}
