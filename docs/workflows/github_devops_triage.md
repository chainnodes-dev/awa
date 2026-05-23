# Process Description: GitHub DevOps Triage & Auto-Responder

## Abstract
Automatically triages incoming GitHub repository issues, analyzes their technical severity, drafts helpful engineering responses, routes critical/high-severity issues to a Human-in-the-Loop maintainer review panel, and posts approved responses back to the GitHub issue using the GitHub MCP server.

## Blackboard Schema Requirements
The blackboard must hold the following strictly typed variables to track state:
- `issue_number` (number, required): The target repository issue ID to process.
- `issue_title` (string): The title of the fetched issue.
- `issue_body` (string): The full markdown text/body of the issue.
- `severity` (string): The classified priority rating ("low", "medium", "high", "critical").
- `technical_analysis` (string): The triage agent's engineering assessment of the problem.
- `draft_comment` (string): The proposed Markdown reply drafted by the agent.
- `human_comments` (string): Any adjustments or notes made by the reviewer.

## Step-by-Step Workflow Specification

1. **Start State (Fetch Issue Details):**
   - Name: `fetch_issue_details`
   - Type: `prompt`
   - Target Agent: `TriageAgent`
   - Description: The workflow initiates with an input `issue_number`. Using the GitHub MCP server, the `TriageAgent` must fetch the issue title and body and save them to the blackboard.
   - Transitions:
     - On successful fetch (`trigger: success`), move to `analyze_and_classify`.
     - On error (`trigger: error`), move to `terminal_failure`.

2. **Analysis State (Triage & Technical Audit):**
   - Name: `analyze_and_classify`
   - Type: `prompt`
   - Target Agent: `TriageAgent`
   - Description: The `TriageAgent` analyzes `issue_title` and `issue_body` on the blackboard. It must determine the root cause, write a thorough `technical_analysis` to the blackboard, and classify the `severity` exactly as one of the following: "low", "medium", "high", or "critical".
   - Transitions:
     - On classification complete (`trigger: done`), move to `draft_engineering_response`.

3. **Draft Response State (Write Auto-Reply):**
   - Name: `draft_engineering_response`
   - Type: `prompt`
   - Target Agent: `DeveloperAgent`
   - Description: The `DeveloperAgent` reads the `technical_analysis` and `issue_body`. It must draft a professional, polite, and technically helpful engineering reply containing code snippets or troubleshooting steps, saving it into `draft_comment`.
   - Transitions:
     - On draft complete (`trigger: done`), move to `evaluate_severity_routing`.

4. **Routing State (Automated Gatekeeper):**
   - Name: `evaluate_severity_routing`
   - Type: `code`
   - Description: A lightweight JavaScript router evaluates `severity`. 
     - If `severity === "high"` or `severity === "critical"`, it must route to the human approval gate.
     - If `severity === "low"` or `severity === "medium"`, it bypasses human approval for hands-free automation.
   - Transitions:
     - If high/critical priority (`trigger: require_review`), transition to `maintainer_review_gate`.
     - If low/medium priority (`trigger: auto_approve`), transition to `post_github_comment`.

5. **Human-in-the-Loop Review State (Maintainer Control Panel):**
   - Name: `maintainer_review_gate`
   - Type: `hitl`
   - Assignee: `maintainer`
   - Instructions: "Review the drafted engineering response for the High/Critical severity issue. You can edit the response text inline or reject the response if it requires manual engineering investigation."
   - Form Schema:
     - Provide a text area displaying the `draft_comment` allowing the human to edit and finalize it.
     - Provide a text field for optional `human_comments`.
   - Transitions:
     - If the maintainer approves the comment (`trigger: approved`), transition to `post_github_comment`.
     - If the maintainer rejects the automated triage (`trigger: rejected`), transition to `manual_escalation`.

6. **Execution State (Post to GitHub):**
   - Name: `post_github_comment`
   - Type: `prompt`
   - Target Agent: `DeveloperAgent`
   - Description: Connects to the GitHub MCP server and calls the `create_issue_comment` tool using the finalized `draft_comment` and `issue_number` from the blackboard.
   - Transitions:
     - On successful comment post (`trigger: done`), transition to `terminal_success`.

7. **Manual Escalation State (Mark for Review):**
   - Name: `manual_escalation`
   - Type: `prompt`
   - Target Agent: `TriageAgent`
   - Description: For issues where the maintainer rejected the draft, add a maintainer review label to the issue via the GitHub MCP server to escalate it for custom engineering follow-up.
   - Transitions:
     - Once labeled (`trigger: done`), transition to `terminal_escalated`.

8. **Terminal States:**
   - `terminal_success` (Type: `terminal`): The issue was triaged and replied to successfully.
   - `terminal_escalated` (Type: `terminal`): High-severity issue flagged and escalated for manual support.
   - `terminal_failure` (Type: `terminal`): System error or failure fetching issue details.
