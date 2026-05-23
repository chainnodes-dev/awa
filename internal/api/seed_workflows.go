package api

import (
	"context"
	"log/slog"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// SeedWorkflows loads built-in example workflows into the store on first boot.
// It is a no-op if any workflow definitions already exist.
func SeedWorkflows(s store.Store, tenantID string) {
	ctx := store.WithTenantID(context.Background(), tenantID)

	existing, err := s.ListDefinitions(ctx, store.DefinitionFilter{Limit: 1})
	if err != nil || len(existing) > 0 {
		return
	}

	for _, wf := range builtinWorkflows {
		def, yamlSource, err := asmtypes.LoadFromYAML([]byte(wf))
		if err != nil {
			slog.Error("Failed to parse built-in workflow", "error", err)
			continue
		}
		if err := s.SaveDefinition(ctx, def, yamlSource); err != nil {
			slog.Error("Failed to seed workflow", "name", def.Metadata.Name, "error", err)
			continue
		}
		slog.Info("Seeded workflow", "name", def.Metadata.Name, "version", def.Metadata.Version)
	}
}

// builtinWorkflows contains YAML definitions for the built-in demo workflows.
// All transitions can be fired manually from the Monitor view — no agent
// infrastructure is required to run through the full flow.
var builtinWorkflows = []string{
	invoiceProcessingWorkflow,
	securityIncidentWorkflow,
	supportRouterWorkflow,
}

const invoiceProcessingWorkflow = `
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: invoice-processing
  version: 1
  description: |
    End-to-end invoice processing: intake → validation → enrichment → human review → approval.

    A single LLM agent (invoice-processor) handles both VALIDATING and ENRICHING.
    Per-state instructions specialise its behaviour at each step.
    INTAKE is a script state that auto-advances with no LLM call.
    Invoices over 10,000 (any currency) are routed to human review.

    Requirements:
      Set LLM_PROVIDER and the matching API key (ANTHROPIC_API_KEY / OPENAI_API_KEY)
      or point OLLAMA_URL at a local Ollama instance and set LLM_PROVIDER=ollama.

    Sample run input (paste into the "Start Run" dialog):
      {
        "invoice_id":      "INV-2024-001",
        "vendor_name":     "Acme GmbH",
        "vendor_address":  "Hauptstrasse 1, 10115 Berlin, Germany",
        "amount":          15000,
        "line_items": [
          { "description": "Software development services Q1", "qty": 1, "unit_price": 15000 }
        ]
      }

blackboard:
  schema:
    invoice_id:
      type: string
      required: true
    vendor_name:
      type: string
      required: true
    vendor_address:
      type: string
      required: true
    amount:
      type: number
      required: true
    currency:
      type: string
    line_items:
      type: object
    validated:
      type: bool
    validation_errors:
      type: object
    category:
      type: string
    approval_notes:
      type: string

states:
  - name: INTAKE
    type: initial
    script:
      # Auto-advance to VALIDATING immediately — no LLM call, no waiting.
      trigger: '"intake_complete"'

  - name: VALIDATING
    type: prompt
    agent: invoice-processor
    timeout: 2m
    on_timeout: validation_failed
    instructions: |
      Check that the invoice has ALL of the following required fields with
      non-empty, plausible values:
        - invoice_id     : non-empty string
        - vendor_name    : non-empty string
        - vendor_address : must include a recognisable city or country
        - amount         : positive number greater than zero
        - line_items     : at least one entry present

      If everything is present and valid:
        Set "validated" to true.
        Set "validation_errors" to an empty list.
        Fire trigger "validation_passed".

      If anything is missing or implausible:
        Set "validated" to false.
        Set "validation_errors" to a list of strings, one problem per entry
        (e.g. ["vendor_address is missing a country", "line_items is empty"]).
        Fire trigger "validation_failed".

  - name: ENRICHING
    type: prompt
    agent: invoice-processor
    timeout: 2m
    on_timeout: enrichment_done
    instructions: |
      Enrich the invoice with exactly two pieces of data, then stop:

      1. CURRENCY — if "currency" is absent or empty, infer the ISO 4217
         currency code from the country in vendor_address.
         Common mappings: Germany/Austria/France/Italy/Spain → EUR,
         United States → USD, United Kingdom → GBP, Switzerland → CHF,
         Japan → JPY, Canada → CAD, Australia → AUD.
         If the address is ambiguous, default to EUR.
         If "currency" is already set, keep the existing value unchanged.

      2. CATEGORY — classify the invoice into exactly one of these expense
         categories based on vendor_name and line_items descriptions:
         software | hardware | services | travel | office | other
         Set "category" to the chosen value.

      When done, always fire trigger "enrichment_done". Do not attempt any
      routing — the system will handle that automatically.

  - name: HUMAN_REVIEW
    type: hitl
    timeout: 72h
    on_timeout: rejected

  - name: APPROVED
    type: terminal

  - name: REJECTED
    type: terminal

transitions:
  - from: INTAKE
    to: VALIDATING
    trigger: intake_complete

  - from: VALIDATING
    to: ENRICHING
    trigger: validation_passed

  - from: VALIDATING
    to: REJECTED
    trigger: validation_failed

  - from: ENRICHING
    to: HUMAN_REVIEW
    trigger: enrichment_done
    guard: "amount > 10000"

  - from: ENRICHING
    to: APPROVED
    trigger: enrichment_done

  - from: HUMAN_REVIEW
    to: APPROVED
    trigger: approved

  - from: HUMAN_REVIEW
    to: REJECTED
    trigger: rejected

agents:
  - name: invoice-processor
    rules:
      - "Respond with valid JSON only — no prose, no markdown, no code fences."
      - "Never invent or fabricate data. Only use what is present on the blackboard."
      - "When in doubt about a field value, prefer the conservative option."
      - "validation_errors must be a JSON array of strings, never a plain string."
`

const securityIncidentWorkflow = `
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: security-incident-triage
  version: 1
  description: |
    Automated triage of security alerts: enrichment → severity assessment → remediation script → human approval.
    Showcases complex agent routing and multi-persona handovers.

blackboard:
  schema:
    incident_id: { type: string, required: true }
    source_ip: { type: string, required: true }
    alert_type: { type: string }
    threat_intel_report: { type: string }
    severity: { type: string }
    remediation_required: { type: bool }
    remediation_plan: { type: string }

states:
  - name: INTAKE
    type: initial
    script: { trigger: '"start"' }

  - name: THREAT_INTEL
    type: prompt
    agent: security-analyst
    instructions: |
      Analyse the source_ip using available threat intelligence.
      Set threat_intel_report with your findings.
      If the IP is a known malicious actor, set severity: 'CRITICAL' and trigger: 'intel_ready'.
      Otherwise, set severity: 'LOW' and trigger: 'intel_ready'.

  - name: REMEDIATION_SCAN
    type: prompt
    agent: security-analyst
    instructions: |
      Based on threat_intel_report, define a remediation plan.
      If the severity is CRITICAL, remediation_required should be true.
      Set remediation_plan as a markdown list of steps.
      Trigger 'plan_done'.

  - name: SOC_APPROVAL
    type: hitl
    instructions: "SOC Lead review required for Critical Incidents."

  - name: RESOLVED
    type: terminal

  - name: ESCALATED
    type: terminal

transitions:
  - { from: INTAKE, to: THREAT_INTEL, trigger: start }
  - { from: THREAT_INTEL, to: REMEDIATION_SCAN, trigger: intel_ready }
  - { from: REMEDIATION_SCAN, to: SOC_APPROVAL, trigger: plan_done, guard: "severity == 'CRITICAL'" }
  - { from: REMEDIATION_SCAN, to: RESOLVED, trigger: plan_done }
  - { from: SOC_APPROVAL, to: RESOLVED, trigger: approved }
  - { from: SOC_APPROVAL, to: ESCALATED, trigger: rejected }

agents:
  - name: security-analyst
    rules:
      - "Strict JSON output only."
      - "Prioritize system safety and containment."
`

const supportRouterWorkflow = `
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: support-router
  version: 1
  description: |
    Intelligent customer support ticket routing: sentiment analysis → urgency detection → agent assignment.

blackboard:
  schema:
    ticket_id: { type: string, required: true }
    customer_message: { type: string, required: true }
    sentiment: { type: string }
    urgency: { type: string }
    assigned_team: { type: string }

states:
  - name: ANALYSIS
    type: initial
    agent: router-bot
    instructions: |
      Analyse customer_message for sentiment (positive/neutral/negative/angry)
      and urgency (low/medium/high/critical).
      If sentiment is 'angry' or urgency is 'critical', set urgency to 'CRITICAL'.
      Trigger 'analysis_done'.

  - name: ROUTING
    type: prompt
    script:
      trigger: "urgency == 'CRITICAL' ? 'priority' : 'standard'"
      updates:
        assigned_team: "urgency == 'CRITICAL' ? 'Senior_Support' : 'General_Support'"

  - name: AUTO_REPLY
    type: prompt
    agent: router-bot
    instructions: "Draft a polite automated reply acknowledging the standard ticket."

  - name: DONE
    type: terminal

transitions:
  - { from: ANALYSIS, to: ROUTING, trigger: analysis_done }
  - { from: ROUTING, to: DONE, trigger: priority }
  - { from: ROUTING, to: AUTO_REPLY, trigger: standard }
  - { from: AUTO_REPLY, to: DONE, trigger: replied }

agents:
  - name: router-bot
    rules:
      - "Keep responses helpful and concise."
      - "If the user is angry, be exceptionally empathetic."
`
