# Workflow DSL

Chain Nodes workflows are defined using a declarative YAML schema. This allows for full version control, portability, and human readability.

## Basic Structure

```yaml
apiVersion: chainnodes/v1
kind: Workflow
metadata:
  name: financial-analysis
  version: 1
  description: "Extracts metrics from a financial report and performs validation"

inputs:
  - name: ticker
    type: string
  - name: report
    type: file

blackboard:
  schema:
    ticker: { type: string, required: true }
    revenue: { type: number }
    report: { type: file }
    approved: { type: bool }

states:
  - name: START
    type: initial
    agent: analyst
    instructions: "Read the report and extract the revenue for {{ ticker }}. Store in revenue."

  - name: REVIEW
    type: hitl
    assignee: finance-team
    instructions: "Verify the extracted revenue total."

  - name: TERMINAL
    type: terminal

transitions:
  - from: START
    to: REVIEW
    trigger: success

  - from: REVIEW
    to: TERMINAL
    trigger: approved
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
Chain Nodes uses the `expr` engine. You can access the blackboard data using `&#123;&#123; blackboard.key &#125;&#125;` in prompts or `blackboard.key` directly in scripts.

## Generative Workflows & Showcase Templates

Chain Nodes supports **AI-driven generation** that creates fully fleshed out, runnable workflows directly from plain-language Process Descriptions under the **Generate** tab in the Designer sidebar.

To help showcase and dogfood Chain Nodes's agentic orchestration, we maintain a set of curated showcase templates. These files are stored directly inside the repository under `docs/workflows/` and serve as perfect prompt blueprints:

1.  **[GitHub Triage & Auto-Responder](https://github.com/chainnodes-dev/awa/blob/main/docs/workflows/github_devops_triage.md)**: Automatically fetches unread GitHub repository issues, runs root-cause analysis, classifies severity, drafts replies, routes critical issues to a Human-in-the-Loop review pane, and posts approved comments using the GitHub MCP server.
2.  **[Gmail Inbox Sorter & Auto-Drafting](https://github.com/chainnodes-dev/awa/blob/main/docs/workflows/gmail_inbox_sorter.md)**: Automatically accesses unread inbox emails via Gmail MCP, assigns an importance rating, drafts professional replies, and prompts the user for review before transmitting.
3.  **[Crypto Market Sentiment Tracker](https://github.com/chainnodes-dev/awa/blob/main/docs/workflows/crypto_sentiment_tracker.md)**: Fetches statistics for the top 10 traded cryptocurrencies, pulls recent web/social sentiment articles, calculates overall market indices, and synthesizes a comprehensive Markdown report.
4.  **[Executive Brand Watcher & Competitor Radar](https://github.com/chainnodes-dev/awa/blob/main/docs/workflows/brand_watcher_radar.md)**: Monitors online mentions of brands or competitors, triages sentiment, alerts maintainers of critical negative events, and posts final executive briefs to Slack.
5.  **[Autonomous Invoice & Expense Auditor](https://github.com/chainnodes-dev/awa/blob/main/docs/workflows/invoice_expense_auditor.md)**: Uses LLM vision to extract invoice totals from local folders, performs mathematical validation via a JS code node, and raises a HITL gate on mathematical mismatches.
6.  **[Intelligent Research & Dossier Compiler](https://github.com/chainnodes-dev/awa/blob/main/docs/workflows/research_dossier_compiler.md)**: Decomposes complex topics, performs parallel Brave and Wikipedia searches, and synthesizes a clean academic Markdown file directly onto the local filesystem.

To try these out, simply copy-paste any template description directly into Chain Nodes Designer's **Generate** box!
