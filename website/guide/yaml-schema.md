# YAML Schema

Phaxa workflows are defined in YAML using the following structure:

```yaml
version: v1
metadata:
  name: my-workflow
  description: "Analyze and summarize data"
  reusable: true

inputs:
  - name: input_data
    type: string

blackboard:
  schema:
    summary: string

states:
  - name: analyze
    type: prompt
    config:
      system: "You are a data analyst."
      user: "Summarize this: &#123;&#123;input_data&#125;&#125;"
    output:
      mapping:
        result: summary
    next: completed

  - name: completed
    type: terminal
```

## Field Definitions

- **metadata**: Basic information about the workflow.
- **inputs**: Entry points for data when starting a run.
- **blackboard**: The shared state schema.
- **states**: A list of execution steps (nodes).
