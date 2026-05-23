package orchestrator

import (
	"context"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// MockTemporalClient implements TemporalEngineClient for testing.
type MockTemporalClient struct {
	OnStartWorkflow      func(run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (string, error)
	OnSendTriggerSignal  func(temporalID, trigger string, payload map[string]interface{}) error
	OnSendHITLSignal     func(temporalID string, sig asmtypes.HITLSignal) error
	OnSendChatSignal     func(temporalID, message, sender string) error
	OnTerminateWorkflow  func(temporalID string) error
}

func (m *MockTemporalClient) StartWorkflow(ctx context.Context, run *asmtypes.WorkflowRun, def *asmtypes.WorkflowDef) (string, error) {
	if m.OnStartWorkflow != nil {
		return m.OnStartWorkflow(run, def)
	}
	return "mock-temporal-id", nil
}

func (m *MockTemporalClient) SendTriggerSignal(ctx context.Context, temporalID, trigger string, payload map[string]interface{}) error {
	if m.OnSendTriggerSignal != nil {
		return m.OnSendTriggerSignal(temporalID, trigger, payload)
	}
	return nil
}

func (m *MockTemporalClient) SendHITLSignal(ctx context.Context, temporalID string, sig asmtypes.HITLSignal) error {
	if m.OnSendHITLSignal != nil {
		return m.OnSendHITLSignal(temporalID, sig)
	}
	return nil
}

func (m *MockTemporalClient) SendChatSignal(ctx context.Context, temporalID, message, sender string) error {
	if m.OnSendChatSignal != nil {
		return m.OnSendChatSignal(temporalID, message, sender)
	}
	return nil
}

func (m *MockTemporalClient) TerminateWorkflow(ctx context.Context, temporalID string) error {
	if m.OnTerminateWorkflow != nil {
		return m.OnTerminateWorkflow(temporalID)
	}
	return nil
}

func (m *MockTemporalClient) AwaitWorkflowCompletion(ctx context.Context, temporalID string) error {
	return nil
}

func (m *MockTemporalClient) Close() {}
