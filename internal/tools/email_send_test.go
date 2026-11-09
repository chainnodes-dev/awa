package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestEmailSendTool_SchemaAndSimulation(t *testing.T) {
	tool := NewEmailSendTool()

	if tool.Name() != "email:send" {
		t.Errorf("expected name email:send, got %s", tool.Name())
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(tool.InputSchema(), &schema); err != nil {
		t.Fatalf("invalid schema json: %v", err)
	}

	// Test simulation execution without SMTP credentials configured
	input := map[string]interface{}{
		"to":      []string{"recipient@example.com"},
		"subject": "Status Report",
		"body":    "All systems nominal.",
	}
	inputBytes, _ := json.Marshal(input)

	res, err := tool.Execute(context.Background(), inputBytes)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	var resMap map[string]interface{}
	if err := json.Unmarshal(res, &resMap); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}

	if resMap["status"] != "simulated_sent" && resMap["status"] != "sent" {
		t.Errorf("unexpected status: %v", resMap["status"])
	}
}
