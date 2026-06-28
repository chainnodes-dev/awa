package api

import (
	"context"
	"testing"

	"github.com/asm-platform/asm/internal/config"
	"github.com/asm-platform/asm/internal/store"
)

func TestLoadAndMergeMarketplace(t *testing.T) {
	cfg := &config.Config{
		MCPMarketSource: "../../configs/mcp-market.json",
	}
	dbURL := "postgres://chainnode:chainnode_secret@localhost:5433/chainnode?sslmode=disable"
	pgStore, err := store.NewPostgresStore(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("Error creating store: %v", err)
	}
	defer pgStore.Close()

	h := &Handlers{
		store: pgStore,
		cfg:   cfg,
	}
	entries, err := h.loadAndMergeMarketplace(context.Background())
	if err != nil {
		t.Fatalf("Merge error: %v", err)
	}

	if len(entries) == 0 {
		t.Errorf("Expected entries to be loaded, got 0")
	}

	// Verify Mattermost is present
	found := false
	for _, entry := range entries {
		if entry.ID == "mattermost" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'mattermost' to be merged into database entries, but it wasn't found")
	}
}
