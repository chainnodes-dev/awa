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

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

type AnthropicProvider struct {
	apiKey          string
	defaultModel    string
	maxOutputTokens int
	client          *http.Client
}

func NewAnthropicProvider(apiKey, defaultModel string, maxTokens int) *AnthropicProvider {
	if defaultModel == "" {
		defaultModel = "claude-sonnet-4-5"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &AnthropicProvider{
		apiKey:          apiKey,
		defaultModel:    defaultModel,
		maxOutputTokens: maxTokens,
		client:          &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

func (p *AnthropicProvider) MaxOutputTokens() int { return p.maxOutputTokens }

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	body := p.buildRequest(req, false)
	respData, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	return p.parseResponse(respData)
}

func (p *AnthropicProvider) Stream(ctx context.Context, req CompletionRequest, tokenCh chan<- string) (*CompletionResponse, error) {
	body := p.buildRequest(req, true)
	respData, err := p.doStreamRequest(ctx, body, tokenCh)
	if err != nil {
		return nil, err
	}
	return p.parseResponse(respData)
}

func (p *AnthropicProvider) buildRequest(req CompletionRequest, stream bool) map[string]interface{} {
	msgs := make([]map[string]interface{}, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = map[string]interface{}{"role": m.Role, "content": m.Content}
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
	if req.SystemPrompt != "" {
		body["system"] = req.SystemPrompt
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.InputSchema,
			}
		}
		body["tools"] = tools
	}
	return body
}

func (p *AnthropicProvider) doRequest(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	data, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(respData))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *AnthropicProvider) doStreamRequest(ctx context.Context, body map[string]interface{}, tokenCh chan<- string) (map[string]interface{}, error) {
	data, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	var fullText string
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			break
		}
		if t, ok := event["type"].(string); ok && t == "content_block_delta" {
			if delta, ok := event["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					fullText += text
					select {
					case tokenCh <- text:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"content": []interface{}{map[string]interface{}{"type": "text", "text": fullText}},
	}, nil
}

func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

func (p *AnthropicProvider) parseResponse(data map[string]interface{}) (*CompletionResponse, error) {
	resp := &CompletionResponse{}

	contentArr, _ := data["content"].([]interface{})
	for _, block := range contentArr {
		b, ok := block.(map[string]interface{})
		if !ok {
			continue
		}
		switch b["type"] {
		case "text":
			resp.Content += fmt.Sprint(b["text"])
		case "tool_use":
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:    fmt.Sprint(b["id"]),
				Name:  fmt.Sprint(b["name"]),
				Input: b["input"],
			})
		}
	}

	if sr, ok := data["stop_reason"].(string); ok {
		resp.StopReason = sr
	}
	if usage, ok := data["usage"].(map[string]interface{}); ok {
		resp.Usage = &TokenUsage{}
		if v, ok := usage["input_tokens"].(float64); ok {
			resp.Usage.InputTokens = int(v)
		}
		if v, ok := usage["output_tokens"].(float64); ok {
			resp.Usage.OutputTokens = int(v)
		}
	}

	return resp, nil
}
