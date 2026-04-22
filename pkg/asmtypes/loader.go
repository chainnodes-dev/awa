package asmtypes

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// LoadFromFile parses a workflow YAML file, expanding {{ env.VAR }} or {{ .Env.VAR }} references.
func LoadFromFile(path string) (*WorkflowDef, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read workflow file: %w", err)
	}
	return LoadFromYAML(data)
}

// LoadFromYAML parses workflow YAML bytes, expanding {{ env.VAR }} or {{ .Env.VAR }} references.
func LoadFromYAML(data []byte) (*WorkflowDef, string, error) {
	expanded, err := expandEnv(data)
	if err != nil {
		return nil, "", fmt.Errorf("expand env vars: %w", err)
	}

	var def WorkflowDef
	if err := yaml.Unmarshal(expanded, &def); err != nil {
		return nil, "", fmt.Errorf("parse yaml: %w", err)
	}

	if err := validate(&def); err != nil {
		return nil, "", fmt.Errorf("validate workflow: %w", err)
	}

	return &def, string(data), nil
}
// expandEnv replaces {{ env.KEY }} or {{ .Env.KEY }} with os.Getenv("KEY").
// It uses regex instead of text/template to avoid crashing on other {{...}} patterns
// that might be present in the YAML (e.g. documentation or AI-generated placeholders).
func expandEnv(data []byte) ([]byte, error) {
	// Pattern 1: {{ env.VAR }} or {{ .Env.VAR }}
	// Pattern 2: {{ env "VAR" }} or {{ env 'VAR' }}
	re := regexp.MustCompile(`{{\s*(?:\.?env\.?|env\s+["']?)([A-Z0-9_]+)["']?\s*}}`)
	
	expanded := re.ReplaceAllFunc(data, func(match []byte) []byte {
		submatches := re.FindSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		key := string(submatches[1])
		return []byte(os.Getenv(key))
	})
	
	return expanded, nil
}

func validate(def *WorkflowDef) error {
	if def.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if def.Metadata.Version == "" {
		return fmt.Errorf("metadata.version is required")
	}
	if len(def.States) == 0 {
		return fmt.Errorf("at least one state is required")
	}

	// Must have exactly one initial state
	initialCount := 0
	stateNames := make(map[string]bool)
	for _, s := range def.States {
		stateNames[s.Name] = true
		if s.Type == StateInitial {
			initialCount++
		}
	}
	if initialCount != 1 {
		return fmt.Errorf("exactly one state of type 'initial' is required (found %d)", initialCount)
	}

	// All transition endpoints must exist
	for _, t := range def.Transitions {
		if !stateNames[t.From] {
			return fmt.Errorf("transition references unknown state '%s'", t.From)
		}
		
		// Validate single 'to' target
		if t.To != "" {
			if !stateNames[t.To] {
				return fmt.Errorf("transition from '%s' references unknown state '%s'", t.From, t.To)
			}
		}

		// Validate parallel 'to_nodes' targets
		for _, target := range t.ToNodes {
			if !stateNames[target] {
				return fmt.Errorf("parallel transition from '%s' references unknown state '%s'", t.From, target)
			}
		}

		if t.To == "" && len(t.ToNodes) == 0 {
			return fmt.Errorf("transition from '%s' has no target states (neither 'to' nor 'to_nodes')", t.From)
		}

		if t.Trigger == "" {
			return fmt.Errorf("transition from '%s' has no trigger", t.From)
		}
	}

	// All agent references in states must exist
	agentNames := make(map[string]bool)
	for _, a := range def.Agents {
		agentNames[a.Name] = true
	}
	for _, s := range def.States {
		if s.Agent != "" && !agentNames[s.Agent] {
			return fmt.Errorf("state '%s' references unknown agent '%s'", s.Name, s.Agent)
		}
	}

	return nil
}
