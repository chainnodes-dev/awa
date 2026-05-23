package asmtypes

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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
	// SANITIZATION: Replace tabs with spaces and remove invisible characters
	// to prevent common YAML unmarshal errors ("found character that cannot start any token").
	s := string(data)
	s = strings.ReplaceAll(s, "\t", "  ")
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "—", "-")
	s = strings.ReplaceAll(s, "–", "-")
	data = []byte(s)

	expanded, err := expandEnv(data)
	if err != nil {
		return nil, "", fmt.Errorf("expand env vars: %w", err)
	}

	var def WorkflowDef
	if err := yaml.Unmarshal(expanded, &def); err != nil {
		return nil, "", fmt.Errorf("parse yaml: %w", err)
	}

	// SYNTHESIS: Convert shorthand "to_nodes" in states into formal Transitions
	// if they aren't already explicitly defined in the transitions block.
	for _, s := range def.States {
		// Include formal transitions defined inside the state
		for _, t := range s.Transitions {
			t.From = s.Name
			def.Transitions = append(def.Transitions, t)
		}

		if s.To != "" || len(s.ToNodes) > 0 {
			// Check if we already have an explicit transition from this state
			hasExplicit := false
			for _, t := range def.Transitions {
				if t.From == s.Name && t.Trigger == "" { // Default trigger
					hasExplicit = true
					break
				}
			}
			if !hasExplicit {
				def.Transitions = append(def.Transitions, Transition{
					From:    s.Name,
					To:      s.To,
					ToNodes: s.ToNodes,
					Trigger: "", // Default/Completion trigger
				})
			}
		}
		if len(s.ElseToNodes) > 0 {
			def.Transitions = append(def.Transitions, Transition{
				From:    s.Name,
				ToNodes: s.ElseToNodes,
				Trigger: "else",
			})
		}
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
