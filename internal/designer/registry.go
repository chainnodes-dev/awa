package designer

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// MCPServerEntry is one entry in the MCP server registry.
type MCPServerEntry struct {
	Name        string   `yaml:"name" json:"name"`
	EnvVar      string   `yaml:"env_var" json:"-"`
	Description string   `yaml:"description" json:"description"`
	Transport   string   `yaml:"transport" json:"transport,omitempty"`
	Command     string   `yaml:"command" json:"command,omitempty"`
	Args        []string `yaml:"args" json:"args,omitempty"`
	URL         string   `yaml:"url" json:"url,omitempty"`
	Tools       []string `yaml:"tools" json:"tools,omitempty"`
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
