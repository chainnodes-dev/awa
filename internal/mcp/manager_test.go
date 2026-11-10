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

func TestResolveArgs(t *testing.T) {
	// 1. Expansion of ${REDIS_URL}
	env := map[string]string{"REDIS_URL": "redis://my-redis:6379"}
	args := []string{"-y", "@modelcontextprotocol/server-redis", "${REDIS_URL}"}
	res := resolveArgs(args, env)
	if len(res) != 3 || res[2] != "redis://my-redis:6379" {
		t.Fatalf("expected expanded REDIS_URL, got %v", res)
	}

	// 2. Auto-appending REDIS_URL when omitted from args
	argsNoArg := []string{"-y", "@modelcontextprotocol/server-redis"}
	resAuto := resolveArgs(argsNoArg, env)
	if len(resAuto) != 3 || resAuto[2] != "redis://my-redis:6379" {
		t.Fatalf("expected auto-appended REDIS_URL, got %v", resAuto)
	}

	// 3. Do not double append when explicit arg is already present
	argsExplicit := []string{"-y", "@modelcontextprotocol/server-redis", "redis://explicit:6379"}
	resExplicit := resolveArgs(argsExplicit, env)
	if len(resExplicit) != 3 || resExplicit[2] != "redis://explicit:6379" {
		t.Fatalf("expected explicit URL preserved without duplicate, got %v", resExplicit)
	}

	// 4. Auto-appending Postgres
	pgEnv := map[string]string{"DATABASE_URL": "postgresql://usr:pwd@host:5432/db"}
	pgArgs := []string{"-y", "@modelcontextprotocol/server-postgres"}
	pgRes := resolveArgs(pgArgs, pgEnv)
	if len(pgRes) != 3 || pgRes[2] != "postgresql://usr:pwd@host:5432/db" {
		t.Fatalf("expected auto-appended DATABASE_URL, got %v", pgRes)
	}
}

