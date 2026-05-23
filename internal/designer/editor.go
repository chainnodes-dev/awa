package designer

import (
	"fmt"
	"strings"

	"github.com/asm-platform/asm/pkg/asmtypes"
)

// WorkflowEditor provides high-level methods to programmatically build
// and modify an asmtypes.WorkflowDef. It ensures that changes made by
// the LLM parameter extraction result in valid internal state.
type WorkflowEditor struct {
	Def *asmtypes.WorkflowDef
}

// NewWorkflowEditor creates a new editor. If the provided definition is empty,
// it initializes it with standard API version and kind.
func NewWorkflowEditor(def *asmtypes.WorkflowDef) *WorkflowEditor {
	if def == nil {
		def = &asmtypes.WorkflowDef{}
	}
	if def.APIVersion == "" {
		def.APIVersion = "chainnodes/v1"
	}
	if def.Kind == "" {
		def.Kind = "Workflow"
	}
	if def.Metadata.Version == "" {
		def.Metadata.Version = "v1"
	}
	return &WorkflowEditor{Def: def}
}

// DeleteState removes a state and all transitions associated with it.
func (e *WorkflowEditor) DeleteState(name string) {
	// Remove state
	for i, s := range e.Def.States {
		if s.Name == name {
			e.Def.States = append(e.Def.States[:i], e.Def.States[i+1:]...)
			break
		}
	}
	// Remove transitions where this state is either From or To
	newTransitions := make([]asmtypes.Transition, 0, len(e.Def.Transitions))
	for _, t := range e.Def.Transitions {
		if t.From != name && t.To != name {
			newTransitions = append(newTransitions, t)
		}
	}
	e.Def.Transitions = newTransitions
}

// AddState appends a new state to the workflow if it does not already exist.
func (e *WorkflowEditor) AddState(name string, stateType asmtypes.StateType, instructions, techReqs string) {
	if e.Def.StateByName(name) != nil {
		return // already exists
	}
	e.Def.States = append(e.Def.States, asmtypes.StateDef{
		Name:                  name,
		Type:                  stateType,
		Instructions:          instructions,
		TechnicalRequirements: techReqs,
	})
}

// UpdateState updates an existing state's fields.
func (e *WorkflowEditor) UpdateState(name string, stateType asmtypes.StateType, instructions, techReqs string) error {
	for i := range e.Def.States {
		if e.Def.States[i].Name == name {
			if stateType != "" {
				e.Def.States[i].Type = stateType
			}
			if instructions != "" {
				e.Def.States[i].Instructions = instructions
			}
			if techReqs != "" {
				e.Def.States[i].TechnicalRequirements = techReqs
			}
			return nil
		}
	}
	return fmt.Errorf("state %q not found", name)
}

// SetStateCode sets the code block for a state.
func (e *WorkflowEditor) SetStateCode(name, code string) error {
	for i := range e.Def.States {
		if e.Def.States[i].Name == name {
			e.Def.States[i].Code = &asmtypes.CodeDef{Code: code}
			return nil
		}
	}
	return fmt.Errorf("state %q not found", name)
}

// SetStateScript sets the script block for a state.
func (e *WorkflowEditor) SetStateScript(name, trigger string, updates map[string]string) error {
	for i := range e.Def.States {
		if e.Def.States[i].Name == name {
			e.Def.States[i].Script = &asmtypes.ScriptDef{
				Trigger: trigger,
				Updates: updates,
			}
			return nil
		}
	}
	return fmt.Errorf("state %q not found", name)
}

// SetStateAgent sets the agent and configures the agent definition.
func (e *WorkflowEditor) SetStateAgent(name, agentName string, mcpServers []string) error {
	for i := range e.Def.States {
		if e.Def.States[i].Name == name {
			e.Def.States[i].Agent = agentName
			
			// Find or create agent definition
			agentDef := e.Def.AgentByName(agentName)
			if agentDef == nil {
				e.Def.Agents = append(e.Def.Agents, asmtypes.AgentDef{
					Name: agentName,
				})
				agentDef = &e.Def.Agents[len(e.Def.Agents)-1]
			}

			// Assign MCP servers to the agent config
			if len(mcpServers) > 0 {
				if agentDef.Config == nil {
					agentDef.Config = make(map[string]string)
				}
				// Standard platform key for MCP server list
				agentDef.Config["mcp_servers"] = strings.Join(mcpServers, ",")
			}
			return nil
		}
	}
	return fmt.Errorf("state %q not found", name)
}


// AddTransition appends or updates a transition.
func (e *WorkflowEditor) AddTransition(from, to, trigger, guard string) {
	for i, t := range e.Def.Transitions {
		if t.From == from && t.Trigger == trigger {
			e.Def.Transitions[i].To = to
			if guard != "" {
				e.Def.Transitions[i].Guard = guard
			}
			return
		}
	}
	e.Def.Transitions = append(e.Def.Transitions, asmtypes.Transition{
		From:    from,
		To:      to,
		Trigger: trigger,
		Guard:   guard,
	})
}

// SetMetadata updates the workflow metadata.
func (e *WorkflowEditor) SetMetadata(name, abstract, description string) {
	if name != "" {
		e.Def.Metadata.Name = name
	}
	if abstract != "" {
		e.Def.Metadata.Description = abstract
	}
	if description != "" {
		e.Def.Metadata.ProcessDescription = description
	}
}

// SetBlackboardSchema sets the blackboard schema definition.
func (e *WorkflowEditor) SetBlackboardSchema(schema map[string]asmtypes.FieldDef) {
	e.Def.Blackboard.Schema = schema
}

// SetPorts sets the workflow input and output ports.
func (e *WorkflowEditor) SetPorts(inputs, outputs []asmtypes.PortDef) {
	e.Def.Inputs = inputs
	e.Def.Outputs = outputs
}

// SetCapabilities sets the global workflow-level MCP server dependencies.
func (e *WorkflowEditor) SetCapabilities(mcpServers []string) {
	e.Def.Capabilities = make([]asmtypes.CapabilityDecl, 0, len(mcpServers))
	for _, s := range mcpServers {
		e.Def.Capabilities = append(e.Def.Capabilities, asmtypes.CapabilityDecl{
			MCPServer: s,
		})
	}
}
