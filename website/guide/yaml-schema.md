# YAML Schema

Chain Nodes workflows are defined in declarative YAML using the following standard structure:

```yaml
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: support-ticket-handler
  version: 1
  description: "Durable customer sentiment and routing agent"

inputs:
  - name: customer_message
    type: string
    description: "The raw support inquiry entered by the customer"
    required: true

blackboard:
  schema:
    customer_message: { type: string, required: true }
    sentiment: { type: string }
    urgency: { type: string }
    assigned_team: { type: string }
    resolution_notes: { type: string }

states:
  - name: INTAKE
    type: initial
    script:
      trigger: '"start"'

  - name: ANALYZE_SENTIMENT
    type: prompt
    agent: support-bot
    instructions: |
      Read 'customer_message' from the blackboard.
      Detect the customer's sentiment (positive/neutral/negative) and urgency (low/medium/high).
      Store the detected values in the 'sentiment' and 'urgency' blackboard keys.
      When done, always fire trigger 'analysis_done'.

  - name: ROUTE_TICKET
    type: script
    script:
      trigger: "urgency == 'high' ? 'priority' : 'standard'"
      updates:
        assigned_team: "urgency == 'high' ? 'Senior_Support' : 'General_Support'"

  - name: MANUAL_REVIEW
    type: hitl
    assignee: "support-leads"
    instructions: "High-urgency ticket review required by a support lead."

  - name: COMPLETE
    type: terminal

transitions:
  - from: INTAKE
    to: ANALYZE_SENTIMENT
    trigger: start

  - from: ANALYZE_SENTIMENT
    to: ROUTE_TICKET
    trigger: analysis_done

  - from: ROUTE_TICKET
    to: MANUAL_REVIEW
    trigger: priority

  - from: ROUTE_TICKET
    to: COMPLETE
    trigger: standard

  - from: MANUAL_REVIEW
    to: COMPLETE
    trigger: approved

agents:
  - name: support-bot
    rules:
      - "Always output valid JSON matches."
      - "Remain concise and professional."
```

## Top-Level Fields

- **apiVersion**: Always `chainnodes/v1`
- **kind**: Always `Workflow`
- **metadata**: Setup info:
  - `name`: Unique, url-friendly process name.
  - `version`: Version integer or string.
  - `description`: Multi-line text describing the workflow goals.
- **inputs**: Declares initial user-collectible fields to pre-fill the start form.
- **blackboard**: The durable key-value schema. Each key contains a `type` (`string`, `number`, `bool`, `object`, `file`) and optional `required` flag.
- **states**: List of execution nodes.
- **transitions**: Global transition registry wiring state nodes together.
- **agents**: AI Agent persona definitions with specific operational rule directives.
