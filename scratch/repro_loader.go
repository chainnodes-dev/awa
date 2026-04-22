package main

import (
	"fmt"
	"os"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

func main() {
	// 1. Problematic YAML that crashes the current loader
	problemYaml := `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: repro-workflow
  version: "1.0.0"
blackboard:
  schema:
    foo: { type: string }
states:
  - name: START
    type: initial
    instructions: "Hello {{blackboard.foo}}"
transitions: []
`

	fmt.Println("--- Attempting to load problematic YAML (should fail currently) ---")
	_, _, err := asmtypes.LoadFromYAML([]byte(problemYaml))
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("Error: YAML loaded successfully (it shouldn't have!)")
	}

	// 2. YAML with valid env var
	os.Setenv("TEST_VAR", "expanded_value")
	envYaml := `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: env-workflow
  version: "1.0.0"
blackboard:
  schema:
    url: { type: string, default: "{{ env.TEST_VAR }}" }
states:
  - name: START
    type: initial
transitions: []
`
	fmt.Println("\n--- Attempting to load YAML with env var ---")
	def, _, err := asmtypes.LoadFromYAML([]byte(envYaml))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Success! Expanded value: %s\n", def.Blackboard.Schema["url"].Default)
	}
}
