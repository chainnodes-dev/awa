# Workflow DSL

Phaxa workflows are defined using a declarative YAML schema. This allows for version control, portability, and readability.

## Basic Structure

```yaml
name: "Financial Analysis"
description: "Extracts metrics from a report"
metadata:
  reusable: true

blackboard:
  schema:
    ticker: "string"
    revenue: "number"
    report: "file"

states:
  START:
    type: "prompt"
    agent: "analyst"
    prompt: "Read the report and extract revenue for {{ blackboard.ticker }}."
    transitions:
      success: "REVIEW"
      
  REVIEW:
    type: "hitl"
    assignee: "finance_team"
    form_schema:
      approved: "boolean"
    transitions:
      success: "TERMINAL"
```

## Node Types

| Type | Description |
| :--- | :--- |
| `prompt` | Executes an LLM agent with optional tools (MCP). |
| `hitl` | Pauses execution for human input or judgment. |
| `script` | Executes a small JS expression for data manipulation. |
| `wait` | Pauses for a specific duration or external signal. |
| `terminal` | The end state of a workflow. |

## Expressions
Phaxa uses the `expr` engine. You can access the blackboard data using `&#123;&#123; blackboard.key &#125;&#125;` in prompts or `blackboard.key` directly in scripts.

## Generative Workflows & Showcase Templates

Phaxa supports **AI-driven generation** that creates fully fleshed out, runnable workflows directly from plain-language Process Descriptions under the **Generate** tab in the Designer sidebar.

To help showcase and dogfood Phaxa's agentic orchestration, we maintain a set of curated showcase templates. These files are stored directly inside the repository under `docs/workflows/` and serve as perfect prompt blueprints:

1.  **[GitHub Triage & Auto-Responder](file:///Users/egongemmel/dev/claude/asm-platform/docs/workflows/github_devops_triage.md)**: Automatically fetches unread GitHub repository issues, runs root-cause analysis, classifies severity, drafts replies, routes critical issues to a Human-in-the-Loop review pane, and posts approved comments using the GitHub MCP server.
2.  **[Gmail Inbox Sorter & Auto-Drafting](file:///Users/egongemmel/dev/claude/asm-platform/docs/workflows/gmail_inbox_sorter.md)**: Automatically accesses unread inbox emails via Gmail MCP, assigns an importance rating, drafts professional replies, and prompts the user for review before transmitting.
3.  **[Crypto Market Sentiment Tracker](file:///Users/egongemmel/dev/claude/asm-platform/docs/workflows/crypto_sentiment_tracker.md)**: Fetches statistics for the top 10 traded cryptocurrencies, pulls recent web/social sentiment articles, calculates overall market indices, and synthesizes a comprehensive Markdown report.
4.  **[Executive Brand Watcher & Competitor Radar](file:///Users/egongemmel/dev/claude/asm-platform/docs/workflows/brand_watcher_radar.md)**: Monitors online mentions of brands or competitors, triages sentiment, alerts maintainers of critical negative events, and posts final executive briefs to Slack.
5.  **[Autonomous Invoice & Expense Auditor](file:///Users/egongemmel/dev/claude/asm-platform/docs/workflows/invoice_expense_auditor.md)**: Uses LLM vision to extract invoice totals from local folders, performs mathematical validation via a JS code node, and raises a HITL gate on mathematical mismatches.
6.  **[Intelligent Research & Dossier Compiler](file:///Users/egongemmel/dev/claude/asm-platform/docs/workflows/research_dossier_compiler.md)**: Decomposes complex topics, performs parallel Brave and Wikipedia searches, and synthesizes a clean academic Markdown file directly onto the local filesystem.

To try these out, simply copy-paste any template description directly into Phaxa Designer's **Generate** box!
