package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

func TestSeedWorkflowsValidity(t *testing.T) {
	// Locate seed-workflows directory relative to test package or repository root
	candidates := []string{
		"../../configs/seed-workflows",
		"configs/seed-workflows",
	}

	var dir string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			dir = c
			break
		}
	}

	if dir == "" {
		t.Skip("configs/seed-workflows directory not found from test runner")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		t.Fatalf("Failed to glob seed workflows: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("No yaml files found in %s", dir)
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			def, _, err := asmtypes.LoadFromFile(file)
			if err != nil {
				t.Fatalf("Failed to load/validate %s: %v", file, err)
			}
			if def.Metadata.Name == "" {
				t.Errorf("%s has empty metadata.name", file)
			}
			if len(def.States) == 0 {
				t.Errorf("%s has no states", file)
			}
		})
	}
}

func TestSeedWorkflowsFunction(t *testing.T) {
	memStore := store.NewMemoryStore()
	seeded, err := SeedWorkflows(memStore, "test-tenant")
	if err != nil {
		t.Fatalf("SeedWorkflows returned error: %v", err)
	}
	if len(seeded) == 0 {
		t.Fatalf("Expected seeded workflows, got 0")
	}

	// Calling again should return 0 new workflows (idempotent)
	reseeded, err := SeedWorkflows(memStore, "test-tenant")
	if err != nil {
		t.Fatalf("Second SeedWorkflows returned error: %v", err)
	}
	if len(reseeded) != 0 {
		t.Errorf("Expected 0 newly seeded workflows on second run, got %d", len(reseeded))
	}
}

