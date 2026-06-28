package api

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// SeedWorkflows loads built-in example workflows from configs/seed-workflows/*.yaml
// into the store on startup if they do not already exist.
func SeedWorkflows(s store.Store, tenantID string) {
	ctx := store.WithTenantID(context.Background(), tenantID)

	dir := "configs/seed-workflows"
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		slog.Error("Failed to list seed workflows", "error", err)
		return
	}

	for _, file := range files {
		wfBytes, err := os.ReadFile(file)
		if err != nil {
			slog.Error("Failed to read seed workflow file", "file", file, "error", err)
			continue
		}

		def, yamlSource, err := asmtypes.LoadFromYAML(wfBytes)
		if err != nil {
			slog.Error("Failed to parse seed workflow", "file", file, "error", err)
			continue
		}

		if _, _, err := s.GetDefinition(ctx, def.Metadata.Name, "v1"); err == nil {
			// Already seeded, skip to avoid overwriting user modifications
			continue
		}

		if err := s.SaveDefinition(ctx, def, yamlSource); err != nil {
			slog.Error("Failed to seed workflow", "name", def.Metadata.Name, "error", err)
			continue
		}
		slog.Info("Seeded workflow dynamically", "name", def.Metadata.Name, "version", def.Metadata.Version, "file", file)
	}
}
