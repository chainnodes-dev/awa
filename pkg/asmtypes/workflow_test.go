package asmtypes

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAgentDef_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name       string
		yamlData   string
		expectErr  bool
		expectName string
		expectConf map[string]string
	}{
		{
			name: "Standard string properties",
			yamlData: `
name: test-agent
model: gpt-4
config:
  provider: openai
  temperature: "0.7"
`,
			expectErr:  false,
			expectName: "test-agent",
			expectConf: map[string]string{
				"provider":    "openai",
				"temperature": "0.7",
			},
		},
		{
			name: "Sequence config property",
			yamlData: `
name: slack-agent
config:
  mcp_servers:
    - slack
    - brave
`,
			expectErr:  false,
			expectName: "slack-agent",
			expectConf: map[string]string{
				"mcp_servers": "slack,brave",
			},
		},
		{
			name: "Mixed configuration types",
			yamlData: `
name: mixed-agent
tools:
  - web-search
rules:
  - be concise
config:
  mcp_servers:
    - slack
  provider: local
  max_runs: 5
  debug: true
`,
			expectErr:  false,
			expectName: "mixed-agent",
			expectConf: map[string]string{
				"mcp_servers": "slack",
				"provider":    "local",
				"max_runs":    "5",
				"debug":       "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var agent AgentDef
			err := yaml.Unmarshal([]byte(tt.yamlData), &agent)
			if (err != nil) != tt.expectErr {
				t.Fatalf("yaml.Unmarshal error = %v, expectErr %v", err, tt.expectErr)
			}
			if err != nil {
				return
			}

			if agent.Name != tt.expectName {
				t.Errorf("expected Name %q, got %q", tt.expectName, agent.Name)
			}

			if len(agent.Config) != len(tt.expectConf) {
				t.Errorf("expected config length %d, got %d", len(tt.expectConf), len(agent.Config))
			}

			for k, v := range tt.expectConf {
				gotVal, ok := agent.Config[k]
				if !ok {
					t.Errorf("expected config key %q to exist", k)
				} else if gotVal != v {
					t.Errorf("expected config key %q to have value %q, got %q", k, v, gotVal)
				}
			}
		})
	}
}
