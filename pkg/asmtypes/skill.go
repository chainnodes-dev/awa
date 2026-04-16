package asmtypes

// SkillDef is the top-level descriptor for a reusable skill.
// A skill is a named, versioned, testable capability with a typed I/O contract.
// Its execution is implemented by a WorkflowDef referenced via WorkflowRef.
//
// Skills are stored and managed independently of workflows.  The platform
// treats them as a higher-level abstraction: a skill is "what" something does,
// the workflow is "how" it is done.
type SkillDef struct {
	APIVersion   string           `yaml:"apiVersion"   json:"apiVersion"`   // "asm/v1"
	Kind         string           `yaml:"kind"         json:"kind"`         // "Skill"
	Metadata     SkillMeta        `yaml:"metadata"     json:"metadata"`
	Inputs       []SkillPortDef   `yaml:"inputs"       json:"inputs"`
	Outputs      []SkillPortDef   `yaml:"outputs"      json:"outputs"`
	Capabilities []CapabilityDecl `yaml:"capabilities" json:"capabilities"`
	// WorkflowRef is the name of the WorkflowDef that implements this skill.
	WorkflowRef string `yaml:"workflow_ref" json:"workflow_ref"`
}

// SkillMeta holds identifying metadata for a skill.
type SkillMeta struct {
	Name        string `yaml:"name"                  json:"name"`
	Version     string `yaml:"version"               json:"version"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// SkillPortDef describes a single typed input or output port of a skill.
// Input ports map to required blackboard fields at run start.
// Output ports map to blackboard fields written by the time a terminal state is reached.
type SkillPortDef struct {
	Name        string `yaml:"name"                  json:"name"`
	Type        string `yaml:"type"                  json:"type"`        // string | number | bool | object
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"    json:"required,omitempty"`
}

// CapabilityDecl declares a dependency on a named MCP server from the registry.
// The platform uses this to validate that required tool servers are available
// before a skill run is started.
type CapabilityDecl struct {
	MCPServer string `yaml:"mcp_server" json:"mcp_server"` // logical name in the MCP registry
}

// SkillCallDef is the configuration embedded in a StateDef of type "skill_call".
// It launches a child workflow (the skill's implementing workflow) with mapped
// inputs, waits for it to complete, and maps the outputs back to the parent blackboard.
type SkillCallDef struct {
	// SkillRef is the name of the WorkflowDef (or SkillDef.WorkflowRef) to invoke.
	SkillRef string `yaml:"skill_ref" json:"skill_ref"`

	// InputMappings maps child skill input port names to parent blackboard field names.
	// Key = child port name, value = parent blackboard field that supplies the value.
	//
	// Example: { "company_name": "vendor" }
	// → child blackboard["company_name"] = parent bb["vendor"]
	InputMappings map[string]string `yaml:"input_mappings,omitempty" json:"input_mappings,omitempty"`

	// OutputMappings maps child skill output port names to parent blackboard field names.
	// Key = child port name, value = parent blackboard field that receives the value.
	//
	// Example: { "revenue_eur": "enriched_revenue" }
	// → parent bb["enriched_revenue"] = child terminal bb["revenue_eur"]
	OutputMappings map[string]string `yaml:"output_mappings,omitempty" json:"output_mappings,omitempty"`

	// CompletionTrigger is the transition trigger fired when the child workflow completes successfully.
	CompletionTrigger string `yaml:"completion_trigger" json:"completion_trigger"`

	// FailureTrigger is fired when the child workflow fails.
	// Defaults to "skill_failed" when empty.
	FailureTrigger string `yaml:"failure_trigger,omitempty" json:"failure_trigger,omitempty"`
}
