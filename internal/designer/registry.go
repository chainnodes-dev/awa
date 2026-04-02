package designer

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// MCPServerEntry is one entry in the MCP server registry.
type MCPServerEntry struct {
	// Name is the logical identifier used when referencing this server in workflow configs.
	Name string `yaml:"name" json:"name"`
	// EnvVar is the environment variable that holds the server URL.
	// Not exposed in the public API — for operator reference only.
	EnvVar string `yaml:"env_var" json:"-"`
	// Description is shown in the AI generator UI so the LLM knows what tools this server provides.
	Description string `yaml:"description" json:"description"`
}

// mcpRegistryFile wraps the top-level YAML structure.
type mcpRegistryFile struct {
	MCPServers []MCPServerEntry `yaml:"mcp_servers"`
}

var (
	registryOnce    sync.Once
	registryEntries []MCPServerEntry
	registryErr     error
)

// LoadMCPRegistry reads and parses the MCP server registry YAML file.
// Subsequent calls return the cached result.
func LoadMCPRegistry(path string) ([]MCPServerEntry, error) {
	registryOnce.Do(func() {
		registryEntries, registryErr = loadMCPRegistryFromFile(path)
	})
	return registryEntries, registryErr
}

func loadMCPRegistryFromFile(path string) ([]MCPServerEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read MCP registry file '%s': %w", path, err)
	}
	var reg mcpRegistryFile
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse MCP registry YAML: %w", err)
	}
	return reg.MCPServers, nil
}

// MCPServerByName returns the entry with the given logical name, or an error.
func MCPServerByName(entries []MCPServerEntry, name string) (*MCPServerEntry, error) {
	for i := range entries {
		if entries[i].Name == name {
			return &entries[i], nil
		}
	}
	return nil, fmt.Errorf("MCP server '%s' not found in registry", name)
}
