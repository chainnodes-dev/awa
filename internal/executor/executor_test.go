package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/asm-platform/asm/internal/events"
	"github.com/asm-platform/asm/internal/executor/llm"
	"github.com/asm-platform/asm/internal/orchestrator"
	"github.com/asm-platform/asm/internal/secrets"
	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/internal/tools"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// ── Mock helpers ─────────────────────────────────────────────────────────────

// mockProvider is a configurable stub of llm.Provider that returns pre-set
// responses in order.
type mockProvider struct {
	name      string
	responses []*llm.CompletionResponse
	callCount int
	maxTokens int
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Complete(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return m.nextResponse()
}

func (m *mockProvider) MaxOutputTokens() int { 
	if m.maxTokens <= 0 {
		return 4096
	}
	return m.maxTokens 
}

func (m *mockProvider) Stream(_ context.Context, _ llm.CompletionRequest, _ chan<- string) (*llm.CompletionResponse, error) {
	return m.nextResponse()
}

func (m *mockProvider) nextResponse() (*llm.CompletionResponse, error) {
	if m.callCount >= len(m.responses) {
		return nil, fmt.Errorf("mockProvider: exhausted after %d call(s)", m.callCount+1)
	}
	r := m.responses[m.callCount]
	m.callCount++
	return r, nil
}

// finalJSONResponse builds a CompletionResponse whose Content is a valid
// AgentOutput JSON blob (no tool calls → executor treats it as the final answer).
func finalJSONResponse(trigger, reasoning string, updates map[string]interface{}) *llm.CompletionResponse {
	out := asmtypes.AgentOutput{
		Trigger:           trigger,
		Reasoning:         reasoning,
		BlackboardUpdates: updates,
	}
	b, _ := json.Marshal(out)
	return &llm.CompletionResponse{Content: string(b), StopReason: "end_turn"}
}

// toolCallResponse builds a CompletionResponse that requests one tool call.
func toolCallResponse(toolName string, input map[string]interface{}) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Content:    "",
		StopReason: "tool_use",
		ToolCalls: []llm.ToolCall{
			{ID: "call_001", Name: toolName, Input: input},
		},
	}
}

// mockTool records calls and returns a fixed JSON result.
type mockTool struct {
	name      string
	result    json.RawMessage
	callCount int
	lastInput json.RawMessage
}

func (t *mockTool) Name() string                 { return t.name }
func (t *mockTool) Description() string          { return "mock tool" }
func (t *mockTool) InputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *mockTool) Execute(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	t.callCount++
	t.lastInput = input
	return t.result, nil
}

// errorTool always returns an error from Execute.
type errorTool struct{ name string }

func (t *errorTool) Name() string                 { return t.name }
func (t *errorTool) Description() string          { return "always errors" }
func (t *errorTool) InputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *errorTool) Execute(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return nil, fmt.Errorf("tool execution failed")
}

// ctxCheckProvider returns ctx.Err() from Stream — used for cancellation tests.
type ctxCheckProvider struct{ name string }

func (p *ctxCheckProvider) Name() string { return p.name }
func (p *ctxCheckProvider) Complete(ctx context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return nil, ctx.Err()
}
func (p *ctxCheckProvider) MaxOutputTokens() int { return 4096 }

func (p *ctxCheckProvider) Stream(ctx context.Context, _ llm.CompletionRequest, _ chan<- string) (*llm.CompletionResponse, error) {
	return nil, ctx.Err()
}

// ── Constructor helpers ───────────────────────────────────────────────────────

func newExecutorWithProvider(providerName string, prov llm.Provider) *Executor {
	reg := llm.NewRegistry()
	reg.Register(prov)
	return NewExecutor(reg, events.NewLocalBus(), store.NewMemoryStore(), providerName, nil, nil, secrets.NewMemorySecretManager())
}

// makeTask builds a minimal AgentTask targeting the given provider name.
func makeTask(providerName string) orchestrator.AgentTask {
	return orchestrator.AgentTask{
		RunID: "run-test-001",
		AgentDef: asmtypes.AgentDef{
			Name:   "test-agent",
			Model:  "mock-model",
			Config: map[string]string{"provider": providerName},
		},
		StateDef: asmtypes.StateDef{
			Name: "TEST_STATE",
			Type: asmtypes.StatePrompt,
		},
		Blackboard: map[string]interface{}{"invoice_id": "INV-001"},
		Def:        &asmtypes.WorkflowDef{},
	}
}

// ── Execute tests ─────────────────────────────────────────────────────────────

// TestExecute_DirectFinalResponse verifies that when the LLM returns a final
// JSON response on the first call (no tool calls), Execute returns the parsed
// AgentOutput immediately.
func TestExecute_DirectFinalResponse(t *testing.T) {
	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			finalJSONResponse("validation_passed", "all checks ok", map[string]interface{}{"validated": true}),
		},
	}
	exec := newExecutorWithProvider("mock", prov)

	output, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Trigger != "validation_passed" {
		t.Errorf("trigger = %q, want %q", output.Trigger, "validation_passed")
	}
	if output.Reasoning != "all checks ok" {
		t.Errorf("reasoning = %q, want %q", output.Reasoning, "all checks ok")
	}
	if output.BlackboardUpdates["validated"] != true {
		t.Errorf("blackboard_updates.validated = %v, want true", output.BlackboardUpdates["validated"])
	}
	if prov.callCount != 1 {
		t.Errorf("provider called %d time(s), want 1", prov.callCount)
	}
}

// TestExecute_OneToolCallThenDone verifies the single tool-call path:
// 1st LLM call → tool_use, 2nd call → final JSON.
func TestExecute_OneToolCallThenDone(t *testing.T) {
	tool := &mockTool{
		name:   "lookup_vendor",
		result: json.RawMessage(`{"vendor_name":"Acme Corp","credit_ok":true}`),
	}
	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			toolCallResponse("lookup_vendor", map[string]interface{}{"vendor_id": "ACME-42"}),
			finalJSONResponse("vendor_ok", "vendor lookup succeeded", map[string]interface{}{"vendor_name": "Acme Corp"}),
		},
	}
	exec := newExecutorWithProvider("mock", prov)
	exec.RegisterTool(tool)

	output, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Trigger != "vendor_ok" {
		t.Errorf("trigger = %q, want %q", output.Trigger, "vendor_ok")
	}
	if tool.callCount != 1 {
		t.Errorf("tool called %d time(s), want 1", tool.callCount)
	}
	if prov.callCount != 2 {
		t.Errorf("provider called %d time(s), want 2", prov.callCount)
	}
}

// TestExecute_MultipleSequentialToolCalls verifies that the LLM can call tools
// on multiple iterations before producing its final answer.
func TestExecute_MultipleSequentialToolCalls(t *testing.T) {
	toolA := &mockTool{name: "tool_a", result: json.RawMessage(`"result_a"`)}
	toolB := &mockTool{name: "tool_b", result: json.RawMessage(`"result_b"`)}

	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			toolCallResponse("tool_a", map[string]interface{}{"x": 1}),
			toolCallResponse("tool_b", map[string]interface{}{"x": 2}),
			finalJSONResponse("done", "both tools used", nil),
		},
	}
	exec := newExecutorWithProvider("mock", prov)
	exec.RegisterTool(toolA)
	exec.RegisterTool(toolB)

	output, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Trigger != "done" {
		t.Errorf("trigger = %q, want %q", output.Trigger, "done")
	}
	if toolA.callCount != 1 || toolB.callCount != 1 {
		t.Errorf("toolA called %d, toolB called %d; want 1 each", toolA.callCount, toolB.callCount)
	}
}

// TestExecute_ToolErrorContinues verifies that when a tool returns an error,
// the executor serialises it and feeds it back to the LLM rather than aborting.
func TestExecute_ToolErrorContinues(t *testing.T) {
	bad := &errorTool{name: "failing_tool"}
	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			toolCallResponse("failing_tool", map[string]interface{}{}),
			// LLM receives the serialised error and produces a final answer.
			finalJSONResponse("fallback", "tool failed, using fallback", nil),
		},
	}
	exec := newExecutorWithProvider("mock", prov)
	exec.RegisterTool(bad)

	output, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Trigger != "fallback" {
		t.Errorf("trigger = %q, want %q", output.Trigger, "fallback")
	}
	// Provider must have been called twice: once for the tool call, once after
	// the error result was fed back.
	if prov.callCount != 2 {
		t.Errorf("provider called %d time(s), want 2", prov.callCount)
	}
}

// TestExecute_UnknownToolErrorContinues verifies that calling a tool not
// present in the registry is also serialised as an error to the LLM.
func TestExecute_UnknownToolErrorContinues(t *testing.T) {
	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			toolCallResponse("nonexistent_tool", map[string]interface{}{}),
			finalJSONResponse("fallback", "tool not found", nil),
		},
	}
	exec := newExecutorWithProvider("mock", prov)
	// No tools registered at all.

	output, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Trigger != "fallback" {
		t.Errorf("trigger = %q, want %q", output.Trigger, "fallback")
	}
}

// TestExecute_MaxIterationsExceeded verifies that the executor returns an error
// when the LLM never stops requesting tool calls.
func TestExecute_MaxIterationsExceeded(t *testing.T) {
	tool := &mockTool{name: "loop_tool", result: json.RawMessage(`"ok"`)}

	responses := make([]*llm.CompletionResponse, maxToolIterations+2)
	for i := range responses {
		responses[i] = toolCallResponse("loop_tool", map[string]interface{}{})
	}
	prov := &mockProvider{name: "mock", responses: responses}

	exec := newExecutorWithProvider("mock", prov)
	exec.RegisterTool(tool)

	_, err := exec.Execute(context.Background(), makeTask("mock"))
	if err == nil {
		t.Fatal("expected error for max iterations exceeded, got nil")
	}
}

// TestExecute_InvalidJSONFallback verifies that when the LLM emits non-JSON
// content on the final (no-tool-call) response, Execute returns a hard error
// so the Temporal activity can be retried rather than silently propagating an
// empty trigger into the workflow engine.
func TestExecute_InvalidJSONFallback(t *testing.T) {
	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			{Content: "Sorry, I cannot help with that.", StopReason: "end_turn"},
		},
	}
	exec := newExecutorWithProvider("mock", prov)

	_, err := exec.Execute(context.Background(), makeTask("mock"))
	if err == nil {
		t.Fatal("expected a hard error when LLM returns non-JSON content, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse agent JSON output") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestExecute_UnknownProvider verifies an error is returned when no matching
// provider is registered.
func TestExecute_UnknownProvider(t *testing.T) {
	reg := llm.NewRegistry()
	exec := NewExecutor(reg, events.NewLocalBus(), store.NewMemoryStore(), "anthropic", nil, nil, secrets.NewMemorySecretManager())

	task := makeTask("missing-provider")
	_, err := exec.Execute(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for missing provider, got nil")
	}
}

// TestExecute_GlobalToolsAvailableToAllAgents verifies that tools registered
// via RegisterTool are available to every agent without per-agent config.
func TestExecute_GlobalToolsAvailableToAllAgents(t *testing.T) {
	tool := &mockTool{name: "global_lookup", result: json.RawMessage(`"found"`)}
	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			toolCallResponse("global_lookup", map[string]interface{}{"key": "abc"}),
			finalJSONResponse("ok", "global tool worked", nil),
		},
	}
	exec := newExecutorWithProvider("mock", prov)
	exec.RegisterTool(tool) // registered globally, not per-agent

	output, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Trigger != "ok" {
		t.Errorf("trigger = %q, want %q", output.Trigger, "ok")
	}
	if tool.callCount != 1 {
		t.Errorf("global tool called %d time(s), want 1", tool.callCount)
	}
}

// TestExecute_ToolReceivesCorrectInput verifies that the JSON input the LLM
// specifies is forwarded unchanged to the tool's Execute method.
func TestExecute_ToolReceivesCorrectInput(t *testing.T) {
	tool := &mockTool{name: "echo_tool", result: json.RawMessage(`"echoed"`)}
	inputMap := map[string]interface{}{"amount": float64(1500), "currency": "USD"}

	prov := &mockProvider{
		name: "mock",
		responses: []*llm.CompletionResponse{
			toolCallResponse("echo_tool", inputMap),
			finalJSONResponse("done", "echo checked", nil),
		},
	}
	exec := newExecutorWithProvider("mock", prov)
	exec.RegisterTool(tool)

	_, err := exec.Execute(context.Background(), makeTask("mock"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(tool.lastInput, &got); err != nil {
		t.Fatalf("tool lastInput is not valid JSON: %v", err)
	}
	if got["amount"] != float64(1500) {
		t.Errorf("tool input amount = %v, want 1500", got["amount"])
	}
	if got["currency"] != "USD" {
		t.Errorf("tool input currency = %v, want USD", got["currency"])
	}
}

// TestExecute_ContextCancellation verifies that a pre-cancelled context
// propagates cleanly as an error.
func TestExecute_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel before Execute is called

	reg := llm.NewRegistry()
	reg.Register(&ctxCheckProvider{name: "mock"})
	exec := NewExecutor(reg, events.NewLocalBus(), store.NewMemoryStore(), "mock-llm", nil, nil, secrets.NewMemorySecretManager())

	_, err := exec.Execute(ctx, makeTask("mock"))
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ── parseAgentOutput unit tests ───────────────────────────────────────────────

func TestParseAgentOutput_ValidJSON(t *testing.T) {
	content := `some preamble {"trigger":"go","reasoning":"ok","blackboard_updates":{"x":1}} trailing`
	out, err := parseAgentOutput(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "go" {
		t.Errorf("trigger = %q, want %q", out.Trigger, "go")
	}
}

func TestParseAgentOutput_PureJSON(t *testing.T) {
	content := `{"trigger":"done","reasoning":"finished","blackboard_updates":{}}`
	out, err := parseAgentOutput(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "done" {
		t.Errorf("trigger = %q, want %q", out.Trigger, "done")
	}
}

func TestParseAgentOutput_NoJSON(t *testing.T) {
	_, err := parseAgentOutput("no json here at all")
	if err == nil {
		t.Fatal("expected error for no JSON, got nil")
	}
}

func TestParseAgentOutput_NestedJSON(t *testing.T) {
	content := `{"trigger":"ok","reasoning":"nested","blackboard_updates":{"data":{"key":"val"}}}`
	out, err := parseAgentOutput(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Trigger != "ok" {
		t.Errorf("trigger = %q, want %q", out.Trigger, "ok")
	}
}

func TestParseAgentOutput_InvalidJSON(t *testing.T) {
	// Has braces but is not valid JSON.
	_, err := parseAgentOutput("{not valid json}")
	if err == nil {
		t.Fatal("expected error for invalid JSON object, got nil")
	}
}

// ── buildAssistantBlocks unit tests ──────────────────────────────────────────

func TestBuildAssistantBlocks_TextOnly(t *testing.T) {
	resp := &llm.CompletionResponse{Content: "thinking...", ToolCalls: nil}
	blocks := buildAssistantBlocks(resp)
	if len(blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(blocks))
	}
	if blocks[0].Type != "text" || blocks[0].Text != "thinking..." {
		t.Errorf("unexpected block: %+v", blocks[0])
	}
}

func TestBuildAssistantBlocks_ToolCallsOnly(t *testing.T) {
	resp := &llm.CompletionResponse{
		Content: "",
		ToolCalls: []llm.ToolCall{
			{ID: "c1", Name: "lookup", Input: map[string]interface{}{"q": "test"}},
		},
	}
	blocks := buildAssistantBlocks(resp)
	if len(blocks) != 1 {
		t.Fatalf("want 1 block, got %d", len(blocks))
	}
	if blocks[0].Type != "tool_use" || blocks[0].ID != "c1" || blocks[0].Name != "lookup" {
		t.Errorf("unexpected block: %+v", blocks[0])
	}
}

func TestBuildAssistantBlocks_TextAndToolCalls(t *testing.T) {
	resp := &llm.CompletionResponse{
		Content: "Let me check that.",
		ToolCalls: []llm.ToolCall{
			{ID: "c1", Name: "check", Input: nil},
			{ID: "c2", Name: "verify", Input: nil},
		},
	}
	blocks := buildAssistantBlocks(resp)
	// Expect: 1 text block + 2 tool_use blocks
	if len(blocks) != 3 {
		t.Fatalf("want 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != "text" {
		t.Errorf("first block type = %q, want text", blocks[0].Type)
	}
	if blocks[1].Type != "tool_use" || blocks[2].Type != "tool_use" {
		t.Errorf("blocks 1/2 not tool_use: %q %q", blocks[1].Type, blocks[2].Type)
	}
}

// ── tools.Registry integration tests ─────────────────────────────────────────

// TestRegistryMerge_AgentOverridesGlobal mirrors what the executor does when
// merging global + agent-specific registries: agent tools win on name collision.
func TestRegistryMerge_AgentOverridesGlobal(t *testing.T) {
	globalReg := tools.NewRegistry()
	globalReg.Register(&mockTool{name: "shared_tool", result: json.RawMessage(`"from-global"`)})

	agentReg := tools.NewRegistry()
	agentReg.Register(&mockTool{name: "shared_tool", result: json.RawMessage(`"from-agent"`)})

	merged := globalReg.Merge(agentReg)

	resolved, err := merged.Get("shared_tool")
	if err != nil {
		t.Fatalf("tool not found after merge: %v", err)
	}
	out, execErr := resolved.Execute(context.Background(), json.RawMessage(`{}`))
	if execErr != nil {
		t.Fatalf("Execute error: %v", execErr)
	}
	if string(out) != `"from-agent"` {
		t.Errorf("merged tool result = %s, want %q (agent should override global)", out, "from-agent")
	}
}

// TestRegistryMerge_GlobalToolPreservedWhenNoAgentConflict verifies that
// global tools are present in the merged registry when the agent adds different
// tools.
func TestRegistryMerge_GlobalToolPreservedWhenNoAgentConflict(t *testing.T) {
	globalReg := tools.NewRegistry()
	globalReg.Register(&mockTool{name: "global_only", result: json.RawMessage(`"g"`)})

	agentReg := tools.NewRegistry()
	agentReg.Register(&mockTool{name: "agent_only", result: json.RawMessage(`"a"`)})

	merged := globalReg.Merge(agentReg)

	if !merged.Has("global_only") {
		t.Error("global_only tool missing from merged registry")
	}
	if !merged.Has("agent_only") {
		t.Error("agent_only tool missing from merged registry")
	}
}
