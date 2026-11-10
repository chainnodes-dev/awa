package mcp

import (
	"context"
	"testing"
	"time"
)

func TestStdioProcess_ExitedProcessDoesNotDeadlock(t *testing.T) {
	mgr := NewManager(nil)

	// Run a command that prints to stderr and exits with error code 1 immediately
	client, closer, err := mgr.TestStdioClientMap("sh", []string{"-c", "echo 'failed to start server' >&2; exit 1"}, nil)
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	defer closer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = client.Call(ctx, "initialize", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error on exited process, got nil")
	}

	t.Logf("Got expected error: %v", err)
}

