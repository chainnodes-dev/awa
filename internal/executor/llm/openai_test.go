package llm

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOpenAIProvider_ConvertMessage_MultipleToolResults(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "gpt-4o", 4096)

	msg := Message{
		Role: "user",
		Content: []ContentBlock{
			{
				Type:      "tool_result",
				ToolUseID: "call_001",
				Output:    map[string]interface{}{"status": "success"},
			},
			{
				Type:      "tool_result",
				ToolUseID: "call_002",
				Output:    map[string]interface{}{"data": "value"},
			},
			{
				Type: "text",
				Text: "This is some text.",
			},
		},
	}

	converted := provider.convertMessage(msg)

	// We expect 3 messages back:
	// 1. role: user, content: "This is some text."
	// 2. role: tool, tool_call_id: "call_001", content: `{"status":"success"}`
	// 3. role: tool, tool_call_id: "call_002", content: `{"data":"value"}`
	if len(converted) != 3 {
		t.Fatalf("expected 3 converted messages, got %d", len(converted))
	}

	// First message: User text
	if converted[0]["role"] != "user" || converted[0]["content"] != "This is some text." {
		t.Errorf("expected first message to be user text, got: %+v", converted[0])
	}

	// Second message: First tool result
	if converted[1]["role"] != "tool" || converted[1]["tool_call_id"] != "call_001" {
		t.Errorf("expected second message to be tool call_001, got: %+v", converted[1])
	}
	var out1 map[string]interface{}
	_ = json.Unmarshal([]byte(converted[1]["content"].(string)), &out1)
	expectedOut1 := map[string]interface{}{"status": "success"}
	if !reflect.DeepEqual(out1, expectedOut1) {
		t.Errorf("expected content to be status:success, got: %s", converted[1]["content"])
	}

	// Third message: Second tool result
	if converted[2]["role"] != "tool" || converted[2]["tool_call_id"] != "call_002" {
		t.Errorf("expected third message to be tool call_002, got: %+v", converted[2])
	}
	var out2 map[string]interface{}
	_ = json.Unmarshal([]byte(converted[2]["content"].(string)), &out2)
	expectedOut2 := map[string]interface{}{"data": "value"}
	if !reflect.DeepEqual(out2, expectedOut2) {
		t.Errorf("expected content to be data:value, got: %s", converted[2]["content"])
	}
}

func TestOpenAIProvider_ConvertMessage_AssistantWithToolCalls(t *testing.T) {
	provider := NewOpenAIProvider("test-key", "gpt-4o", 4096)

	msg := Message{
		Role: "assistant",
		Content: []ContentBlock{
			{
				Type: "text",
				Text: "Thinking...",
			},
			{
				Type: "tool_use",
				ID:   "call_001",
				Name: "my_tool",
				Input: map[string]interface{}{
					"arg": "val",
				},
			},
		},
	}

	converted := provider.convertMessage(msg)

	if len(converted) != 1 {
		t.Fatalf("expected 1 converted message for assistant, got %d", len(converted))
	}

	res := converted[0]
	if res["role"] != "assistant" || res["content"] != "Thinking..." {
		t.Errorf("expected assistant message, got: %+v", res)
	}

	toolCalls, ok := res["tool_calls"].([]map[string]interface{})
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("expected tool_calls in assistant message, got: %+v", res["tool_calls"])
	}

	tc := toolCalls[0]
	if tc["id"] != "call_001" || tc["type"] != "function" {
		t.Errorf("unexpected tool call: %+v", tc)
	}
}
