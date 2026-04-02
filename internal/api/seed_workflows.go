package api

import (
	"context"
	"log"

	"github.com/asm-platform/asm/internal/store"
	"github.com/asm-platform/asm/pkg/asmtypes"
)

// seedWorkflows loads built-in example workflows into the store on first boot.
// It is a no-op if any workflow definitions already exist.
func seedWorkflows(s store.Store) {
	ctx := context.Background()

	existing, err := s.ListDefinitions(ctx)
	if err != nil || len(existing) > 0 {
		return
	}

	for _, wf := range builtinWorkflows {
		def, yamlSource, err := asmtypes.LoadFromYAML([]byte(wf))
		if err != nil {
			log.Printf("[server] failed to parse built-in workflow: %v", err)
			continue
		}
		if err := s.SaveDefinition(ctx, def, yamlSource); err != nil {
			log.Printf("[server] failed to seed workflow '%s': %v", def.Metadata.Name, err)
			continue
		}
		log.Printf("[server] seeded workflow '%s' v%s", def.Metadata.Name, def.Metadata.Version)
	}
}

// builtinWorkflows contains YAML definitions for the built-in demo workflows.
// All transitions can be fired manually from the Monitor view — no agent
// infrastructure is required to run through the full flow.
var builtinWorkflows = []string{
	invoiceProcessingWorkflow,
}

const invoiceProcessingWorkflow = `
apiVersion: asm/v1
kind: Workflow
metadata:
  name: invoice-processing
  version: 1.0.0
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
    type: intermediate
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
    type: intermediate
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
