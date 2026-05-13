package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openAIAPIURL = "https://api.openai.com/v1/chat/completions"

type OpenAIProvider struct {
	name            string // provider name returned by Name()
	apiKey          string
	apiURL          string // chat completions endpoint
	defaultModel    string
	maxOutputTokens int
	client          *http.Client
}

// NewOpenAIProvider creates a provider backed by the OpenAI API.
func NewOpenAIProvider(apiKey, defaultModel string, maxTokens int) *OpenAIProvider {
	if defaultModel == "" {
		defaultModel = "gpt-4o"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &OpenAIProvider{
		name:            "openai",
		apiKey:          apiKey,
		apiURL:          openAIAPIURL,
		defaultModel:    defaultModel,
		maxOutputTokens: maxTokens,
		client:          &http.Client{Timeout: 300 * time.Second},
	}
}

// NewOpenAICompatibleProvider creates a provider that speaks the OpenAI chat
// completions wire format but targets a custom base URL (e.g. Ollama, vLLM,
// LM Studio, Together, Groq …).
//
//	baseURL  – full URL to the chat completions endpoint,
//	           e.g. "http://mintbox:11434/v1/chat/completions"
//	apiKey   – optional; pass "" for local servers that don't require auth
//	name     – logical provider name registered in the LLM registry
//	model    – default model identifier (Ollama model tag, vLLM model ID, …)
func NewOpenAICompatibleProvider(baseURL, apiKey, name, defaultModel string, maxTokens int) *OpenAIProvider {
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &OpenAIProvider{
		name:            name,
		apiKey:          apiKey,
		apiURL:          baseURL,
		defaultModel:    defaultModel,
		maxOutputTokens: maxTokens,
		client:          &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return p.name }

func (p *OpenAIProvider) MaxOutputTokens() int { return p.maxOutputTokens }

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	body := p.buildRequest(req, false)
	respData, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	return p.parseResponse(respData)
}

func (p *OpenAIProvider) Stream(ctx context.Context, req CompletionRequest, tokenCh chan<- string) (*CompletionResponse, error) {
	// For simplicity, fall back to non-streaming and send as single token
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	select {
	case tokenCh <- resp.Content:
	case <-ctx.Done():
	}
	return resp, nil
}

func (p *OpenAIProvider) buildRequest(req CompletionRequest, stream bool) map[string]interface{} {
	msgs := []map[string]interface{}{}
	if req.SystemPrompt != "" {
		msgs = append(msgs, map[string]interface{}{"role": "system", "content": req.SystemPrompt})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, p.convertMessage(m))
	}

	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   msgs,
		"stream":     stream,
	}

	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				},
			}
		}
		body["tools"] = tools
	}
	return body
}

// convertMessage translates the provider-agnostic Message into OpenAI's wire format.
// Content blocks (Anthropic-style) are translated to OpenAI equivalents.
func (p *OpenAIProvider) convertMessage(m Message) map[string]interface{} {
	blocks, ok := m.Content.([]ContentBlock)
	if !ok {
		// Plain string content or already a raw value.
		msg := map[string]interface{}{"role": m.Role, "content": m.Content}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		return msg
	}

	// Translate content blocks.
	switch m.Role {
	case "assistant":
		// Collect text and tool_use blocks → OpenAI assistant message with tool_calls.
		var text string
		var toolCalls []map[string]interface{}
		for _, b := range blocks {
			switch b.Type {
			case "text":
				text += b.Text
			case "tool_use":
				argBytes, _ := json.Marshal(b.Input)
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   b.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      b.Name,
						"arguments": string(argBytes),
					},
				})
			}
		}
		msg := map[string]interface{}{"role": "assistant", "content": text}
		
		// Preserve DeepSeek reasoning_content if present in any block.
		for _, b := range blocks {
			if b.Reasoning != "" {
				msg["reasoning_content"] = b.Reasoning
				break
			}
		}

		if len(toolCalls) > 0 {
			msg["tool_calls"] = toolCalls
		}
		return msg

	case "user":
		// tool_result blocks → individual role:tool messages are added separately.
		// OpenAI doesn't support batching tool results inside a user message,
		// so we encode each as a JSON string inside a single user content for now.
		// The executor ensures tool results are sent as individual tool messages.
		var text string
		for _, b := range blocks {
			if b.Type == "tool_result" {
				resultBytes, _ := json.Marshal(b.Output)
				return map[string]interface{}{
					"role":         "tool",
					"tool_call_id": b.ToolUseID,
					"content":      string(resultBytes),
				}
			}
			if b.Type == "text" {
				text += b.Text
			}
		}
		return map[string]interface{}{"role": "user", "content": text}
	}

	return map[string]interface{}{"role": m.Role, "content": m.Content}
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s request: %w", p.name, err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s API error %d: %s", p.name, resp.StatusCode, string(respData))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *OpenAIProvider) parseResponse(data map[string]interface{}) (*CompletionResponse, error) {
	resp := &CompletionResponse{}

	choices, _ := data["choices"].([]interface{})
	if len(choices) == 0 {
		return resp, nil
	}

	choice, _ := choices[0].(map[string]interface{})
	msg, _ := choice["message"].(map[string]interface{})

	if content, ok := msg["content"].(string); ok {
		resp.Content = content
	}

	if reasoning, ok := msg["reasoning_content"].(string); ok {
		resp.Reasoning = reasoning
	}

	if toolCalls, ok := msg["tool_calls"].([]interface{}); ok {
		for _, tc := range toolCalls {
			call, _ := tc.(map[string]interface{})
			fn, _ := call["function"].(map[string]interface{})
			var input interface{}
			if argStr, ok := fn["arguments"].(string); ok {
				_ = json.Unmarshal([]byte(argStr), &input)
			}
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:    fmt.Sprint(call["id"]),
				Name:  fmt.Sprint(fn["name"]),
				Input: input,
			})
		}
	}

	if fr, ok := choice["finish_reason"].(string); ok {
		resp.StopReason = fr
	}

	return resp, nil
}
