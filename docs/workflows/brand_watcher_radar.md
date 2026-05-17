# Process Description: Executive Brand Watcher & Competitor Radar

## Abstract
Monitors specified brand names and competitor products across the web using search MCP servers, filters mentions by sentiment and business relevance, formats a clean Markdown briefing, and pushes the final alert to the executive team via Slack or email.

## Blackboard Schema Requirements
The blackboard must hold the following strictly typed variables to track state:
- `target_brands` (list of strings, required): List of brand or competitor names to track (e.g. "Phaxa", "Temporal").
- `raw_mentions` (list of objects): A collected array of recent articles, forum posts, and tweets containing brand mentions.
- `filtered_mentions` (list of objects): Mentions annotated with relevance scores (1-10) and sentiment labels ("positive", "negative", "neutral").
- `dossier_markdown` (string): The structured, polished executive markdown briefing.
- `slack_channel` (string): Target Slack channel for delivery.

## Step-by-Step Workflow Specification

1. **Start State (Query Search Engines):**
   - Name: `search_brand_mentions`
   - Type: `prompt`
   - Target Agent: `RadarAgent`
   - Description: Iterate through the `target_brands` list. For each brand, execute web search queries using the Brave Search MCP server. Gather the title, URL, publication date, and snippet of matching results, saving them to `raw_mentions`.
   - Transitions:
     - On successful search (`trigger: success`), move to `filter_and_evaluate_mentions`.
     - On error (`trigger: error`), transition to `terminal_failure`.

2. **Analysis State (Filter & Triage):**
   - Name: `filter_and_evaluate_mentions`
   - Type: `prompt`
   - Target Agent: `RadarAgent`
   - Description: Read `raw_mentions` and filter out noise (such as spam or irrelevant words). For each valid mention, calculate a `relevance_score` from 1 to 10 and determine if the sentiment is "positive", "negative", or "neutral". Write the resulting list to `filtered_mentions`.
   - Transitions:
     - On triage completed (`trigger: done`), move to `generate_executive_briefing`.

3. **Synthesis State (Create Markdown Brief):**
   - Name: `generate_executive_briefing`
   - Type: `prompt`
   - Target Agent: `WriterAgent`
   - Description: Read `filtered_mentions`. Synthesize the findings into an executive report. The report must contain: 
     - A summary of current competitor momentum.
     - A categorized breakdown of positive mentions vs. negative complaints.
     - Clickable source links.
     Save this formatted markdown text to `dossier_markdown`.
   - Transitions:
     - On document generated (`trigger: done`), move to `evaluate_alert_urgency`.

4. **Routing State (Evaluate Urgency):**
   - Name: `evaluate_alert_urgency`
   - Type: `code`
   - Description: A JavaScript script scans `filtered_mentions` for any highly critical negative reviews or PR incidents (relevance score >= 9 with negative sentiment).
   - Transitions:
     - If a critical mention is found (`trigger: escalation`), transition to `maintainer_incident_review`.
     - Otherwise (`trigger: standard`), transition to `post_slack_briefing`.

5. **Human-in-the-Loop Review State (Crisis Intervention):**
   - Name: `maintainer_incident_review`
   - Type: `hitl`
   - Assignee: `maintainer`
   - Instructions: "A critical brand incident or highly negative competitor launch has been detected. Review the briefing and add maintainer comments before delivering this message to the executive team."
   - Form Schema:
     - Display the `dossier_markdown` briefing in an editable textarea.
     - Provide an input field for critical incident overrides.
   - Transitions:
     - On maintainer sign-off (`trigger: approved`), transition to `post_slack_briefing`.

6. **Execution State (Slack Hook Delivery):**
   - Name: `post_slack_briefing`
   - Type: `prompt`
   - Target Agent: `RadarAgent`
   - Description: Deliver the finalized `dossier_markdown` briefing directly into the `slack_channel` using a Slack MCP server or a Fetch MCP post request.
   - Transitions:
     - On message delivered (`trigger: done`), transition to `terminal_success`.

7. **Terminal States:**
   - `terminal_success` (Type: `terminal`): Competitive brand report successfully compiled and sent.
   - `terminal_failure` (Type: `terminal`): Failed to query web search APIs or transmit Slack payloads.
