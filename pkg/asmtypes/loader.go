package asmtypes

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

// LoadFromFile parses a workflow YAML file, expanding {{ env.VAR }} references.
func LoadFromFile(path string) (*WorkflowDef, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read workflow file: %w", err)
	}
	return LoadFromYAML(data)
}

// LoadFromYAML parses workflow YAML bytes, expanding {{ env.VAR }} references.
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

// expandEnv replaces {{ env.KEY }} with os.Getenv("KEY").
func expandEnv(data []byte) ([]byte, error) {
	tmpl, err := template.New("workflow").
		Delims("{{", "}}").
		Funcs(template.FuncMap{
			"env": os.Getenv,
		}).
		Parse(string(data))
	if err != nil {
		return nil, err
	}

	// Build env map for template variables like {{ env.KEY }}
	type envWrapper struct{ Env map[string]string }
	envMap := envWrapper{Env: envFromOS()}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, envMap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func envFromOS() map[string]string {
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
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
