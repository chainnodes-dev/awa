package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
	"github.com/gin-gonic/gin"
)

// SeedWorkflows loads built-in example workflows from configs/seed-workflows/*.yaml
// into the store on startup or on demand if they do not already exist.
func SeedWorkflows(s store.Store, tenantID string) ([]string, error) {
	ctx := store.WithTenantID(context.Background(), tenantID)

	candidates := []string{
		"configs/seed-workflows",
		"../../configs/seed-workflows",
	}

	var dir string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			dir = c
			break
		}
	}

	if dir == "" {
		slog.Warn("Configs/seed-workflows directory not found in candidates")
		return nil, fmt.Errorf("configs/seed-workflows directory not found")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		slog.Error("Failed to list seed workflows", "error", err)
		return nil, err
	}

	var seeded []string
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

		version := def.Metadata.Version
		if version == "" {
			version = "v1"
		}

		if _, _, err := s.GetDefinition(ctx, def.Metadata.Name, version); err == nil {
			// Already seeded, skip to avoid overwriting user modifications
			continue
		}
		if version != "v1" {
			if _, _, err := s.GetDefinition(ctx, def.Metadata.Name, "v1"); err == nil {
				continue
			}
		}

		if err := s.SaveDefinition(ctx, def, yamlSource); err != nil {
			slog.Error("Failed to seed workflow", "name", def.Metadata.Name, "error", err)
			continue
		}
		seeded = append(seeded, def.Metadata.Name)
		slog.Info("Seeded workflow dynamically", "name", def.Metadata.Name, "version", def.Metadata.Version, "file", file)
	}

	return seeded, nil
}

// SeedWorkflowsHandler triggers seeding of seed workflows for the caller's tenant.
func (h *Handlers) SeedWorkflows(c *gin.Context) {
	tenantID := store.TenantIDFromContext(c.Request.Context())
	if tenantID == "" {
		tenantID = "default"
	}

	seeded, err := SeedWorkflows(h.store, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Seed workflows processed",
		"seeded":  seeded,
		"count":   len(seeded),
	})
}

