package api

import (
	"github.com/asm-platform/asm/internal/designer"
	"github.com/asm-platform/asm/internal/store"
)

// SeedMCPServers was removed as registered servers are now managed via the Marketplace UI.
func SeedMCPServers(s store.Store, tenantID string, registryEntries []designer.MCPServerEntry) {
	// No-op: Automatic seeding is disabled to respect user-managed registries.
}
