# Concept: ASM Platform as an Auto-Remediation Plugin & AgSOP Marketplace

## 1. Executive Summary
Instead of positioning the ASM Platform as a direct, general-purpose competitor to established orchestration tools like n8n or Zapier, we are pivoting to a highly specialized niche: **Autonomous DevOps & SRE Remediation**. 

In this model, ASM does not replace existing incident routing tools (like PagerDuty, Opsgenie, or GoAlert). Instead, it acts as a powerful **"Remediation Plugin"** that seamlessly connects to these tools via webhooks, serving as the intelligent "brawn" behind their incident routing "brains."

To scale this vision, we introduce the concept of **Agentic SOPs (AgSOPs)** and a community-driven **AgSOP Marketplace**.

## 2. The Core Concept: Agentic SOPs (AgSOPs)
Currently, SRE teams rely on static, written Standard Operating Procedures (SOPs) or wikis to resolve incidents at 3:00 AM. 
An **AgSOP** transforms a static wiki page into an executable, AI-driven workflow. 
- It is a pre-defined ASM sub-process tailored to a specific issue (e.g., "Postgres High CPU", "Kubernetes Pod Crash").
- It uses LLM reasoning to dynamically investigate the problem using strictly permissioned tools.
- It formulates a fix and uses ASM's native **Human-in-the-Loop (HITL)** state to ask the human engineer for final approval via ChatOps (e.g., Slack/Mattermost) before executing destructive or high-risk actions.

## 3. System Architecture
The platform is designed around a modular, plugin-friendly architecture:

1. **The Inbound Trigger:** An incident management tool (e.g., GoAlert, PagerDuty) triggers an alert and sends a webhook payload to the ASM Platform.
2. **The Triage Router:** A standard entry-point workflow built into the ASM base package. It uses a lightweight LLM or deterministic tag-matching to analyze the webhook payload (service name, issue type) and routes the incident to the appropriate AgSOP sub-process.
3. **The AgSOP Sub-Process:** The specialized workflow takes over. It autonomously queries logs, checks recent Git commits, or analyzes database locks.
4. **The HITL Handshake:** The AgSOP pauses and pings the on-call engineer (via Slack/Teams) with a summary and a proposed fix, awaiting a 1-click approval.

## 4. The MCP Security Paradigm
The primary concern with community-shared agentic workflows is the "blast radius" of unpredictable LLM behavior (e.g., a prompt injection causing the agent to drop a database).

We mitigate this entirely by relying on the **Model Context Protocol (MCP)** as a hard security perimeter:
- **Security at the Tool Layer:** AgSOPs do not possess inherent credentials or generic API access. They are restricted to invoking the MCP servers connected to the user's ASM instance.
- **Deterministic Guardrails:** If a user configures their Postgres MCP server as `Read-Only`, a malicious or hallucinated prompt within a community AgSOP physically cannot execute a `DROP TABLE` command. The prompt cannot override the deterministic security of the MCP server.
- This paradigm allows users to safely download and run community-written AI workflows, knowing the AI is boxed in by their strict infrastructure configurations.

## 5. The Community Marketplace Model
To drive rapid adoption and feature growth, we will utilize a marketplace model similar to n8n's community nodes, but tailored for SRE use cases.

### The Base Package
When a user installs the ASM Platform, they receive:
- The core orchestration engine and Triage Router.
- A curated suite of basic MCP servers (Postgres, MySQL, GitHub, GitLab, Jira, Datadog, Slack, Teams).
- A handful of official, highly vetted example AgSOPs to demonstrate value immediately.

### The AgSOP Library (Marketplace)
Users can browse, upload, and share custom AgSOPs.
- **Dependency Mapping:** When a user downloads an AgSOP (e.g., "Auto-Triage Jira Issues"), the ASM platform detects the required tools and prompts the user to map their existing, secure MCP servers to the AgSOP's requirements (e.g., *"This AgSOP requires an Issue Tracker. Please map your Jira MCP server here."*).
- **Community Growth:** Just as n8n grew massively because the community built integrations the core team couldn't get to, the ASM community will build highly specialized AgSOPs for niche infrastructure setups (e.g., "Redis Cluster Rebalance AgSOP"), instantly expanding the platform's utility without internal R&D overhead.

## 6. Go-To-Market Value Proposition
- **For SREs:** Reduces Mean Time To Resolution (MTTR) drastically. While the engineer is waking up, the AgSOP has already correlated the logs, found the bad commit, and drafted the revert PR.
- **For Enterprises:** Maintains strict compliance and security. The AI investigates, but the human approves (HITL), and the Time Travel Scrubber provides an immutable audit log of exactly what the AI did.
- **The Pitch:** *"Don't rip and replace PagerDuty. Just plug ASM into it, and give your alerts a brain."*

### 7. Strategic Niche Evaluation: Autonomous SRE Remediation
*   **The Use Case:** "Level 1 On-Call Agent." PagerDuty triggers an alert, the ASM workflow wakes up, uses MCP to pull Datadog logs and read GitHub, writes a patch, and asks the sleeping engineer via Slack/Teams: "Approve to deploy?"
*   **Incumbent Competitors:** Resolve AI, Rootly, incident.io, PagerDuty Operations Cloud.
*   **How ASM Differentiates:** Incumbents are mostly "alert routers" heavily relying on 3rd-party webhook scripts for remediation. ASM provides an **AgSOP (Agentic SOP) Marketplace** and deep MCP sandboxing, making it an actual remediation engine rather than just an alert summarizer.
*   **Market Entry Success Estimate: 9/10 (Highly Viable)** - Engineering platform teams are actively seeking to reduce MTTR and are the most willing demographic to adopt AI-coding assistants for backend operations.

### 7.1 Pros & Cons
**5 Pros:**
1. Clear, measurable ROI (reduces MTTR from hours to minutes).
2. Highly viral growth model among engineering teams.
3. Perfectly aligned with the "AgSOP Marketplace" architecture.
4. Developer tools have the most robust APIs for MCP servers.
5. High willingness to pay for tools that prevent production downtime.

**5 Cons:**
1. Engineers are highly skeptical of AI touching production infrastructure.
2. The stakes are immense; a hallucination could bring down production.
3. Requires relying heavily on the API stability of 3rd party tools (AWS, Datadog).
4. Hard to market to non-technical buyers (e.g., CFOs).
5. Vulnerable to PagerDuty or Datadog eventually building this exact feature natively.

### 7.2 Gap Analysis & Feature Status
*   **[COMPLETED] Headless HITL Integrations:** Robust interactive Slack/Teams messages so engineers can approve fixes directly from chat (implemented native Slack/Teams button alerts and `slack-resolve` webhook endpoints).
*   **[COMPLETED] Dry-Run / Sandbox Mode:** A simulation mode where the AI writes the fix but all write-actions are intercepted, allowing SREs to build trust (implemented execution-layer write query interception and active MCP scope validation).
*   **[PLANNED] Hardened SRE MCP Servers:** Officially supported, highly secure MCP servers for Kubernetes, AWS, Datadog, PagerDuty, and GitHub.

---

## 8. Implementation Roadmap: Missing Technical Pieces
To realize this vision and create a viable, secure product for this segment, we need to build the following core components into the ASM Platform:

### 8.1 AgSOP Manifests (Sub-Process Contracts)
Workflows can no longer just be standalone YAML files. We must introduce an **AgSOP Manifest** schema. When a community author creates an AgSOP, the manifest must declare:
- **Required MCP Interfaces:** The AgSOP must request abstract capabilities (e.g., `requires: { type: "database", scope: "read-only" }`) rather than hardcoded local server names.
- **Input/Output Schema:** The expected JSON structure it receives from the Triage Router and what it returns upon completion.
- **Environment Variables:** Required non-MCP secrets (if any) needed for execution.

### 8.2 Enhanced MCP Server Management & Mapping UI
The current MCP installation process must be upgraded to support the AgSOP handshake:
- **Permission Granularity:** When a user installs an MCP server, the UI must allow them to explicitly toggle the server's permission scopes (e.g., turning off write-access tools at the server level to enforce the security paradigm).
- **The Installation Handshake:** When a user clicks "Install AgSOP" from the marketplace, a wizard must prompt them to satisfy the manifest requirements by mapping their configured, secure MCP servers to the AgSOP's declared needs.

### 8.3 Sub-Process Execution Isolation
The engine's runtime must be modified to enforce the manifest during execution:
- When an AgSOP sub-process is spawned, the engine must inject *only* the explicitly mapped MCP tools into the LLM's context window. It must be sandboxed from the rest of the user's global MCP environment to prevent cross-contamination or unauthorized access.

### 8.4 Dry-Run / Simulation Mode
Because AgSOPs deal with critical infrastructure, users will not trust them blindly. We must implement a "Simulation Mode":
- Users can fire a mock webhook (e.g., a dummy PagerDuty alert) into the system.
- The AgSOP runs, but the ASM engine intercepts all MCP `write` or `execute` tool calls, logging them instead of executing them.
- The user reviews the Time Travel Scrubber to verify the agent reasoned correctly before flipping the AgSOP to "Production."

### 8.5 Headless HITL (ChatOps Integration)
To truly act as a background plugin, users shouldn't have to log into the ASM dashboard at 3:00 AM to click "Approve."
- We must build robust Slack/Mattermost/Teams interactive integrations.
- When an AgSOP hits a HITL state, it must generate an interactive message block with the summary and an "Approve / Reject" button.
- Clicking the button must fire a secure API callback to ASM, resolving the HITL state entirely from within the chat interface.

### 8.6 Centralized Marketplace Backend
- A public registry API and frontend gallery where community members can publish their AgSOP YAMLs, version them, and receive reviews or trust scores (similar to Docker Hub or npm) to establish quality and safety metrics.

### 8.7 Domain-Specific Prompt Profiles (AI Designer Optimization)
Because a generic prompt is insufficient to generate state machines that adhere to specific industry standards or use cases (such as DevOps auto-remediation, crypto trading bots, or legal compliance), we will support **Domain-Specific Prompt Profiles**:
- **The Concept:** Admins or users can select a "Prompt Profile" (e.g., SRE / DevOps, Algorithmic Trading, CRM Operations) for their workspace.
- **How it Works:** Switching profiles swaps out the system prompts (`workflow_generator_base`, `workflow_decompose`, `workflow_categorise`, `workflow_debugger`) loaded by the AI designer.
- **Best-Practice Enforcement:** For example, the SRE prompt profile will force the AI generator to always place Human-in-the-Loop (HITL) gate nodes before executing any write actions or restarting clusters, and format the investigation states to request the correct abstract database/kubernetes capabilities.
- **Dynamic Override Integration:** This leverages our existing dynamic prompt management APIs (`/api/v1/designer/prompts`) to dynamically load and save these overrides at runtime.

## 9. Open-Source Proof of Concept (PoC) Stack
To build a full, end-to-end Proof of Concept (PoC) without paying for enterprise SaaS licenses, we can use a completely open-source stack.

Here is what the PoC stack looks like:

### 9.1 The Alerting & Routing Layer: GoAlert
*   **What it is:** [GoAlert](https://goalert.me/) is currently the leading open-source alternative to PagerDuty for on-call scheduling and alert routing. (It is also written in Go, which aligns nicely with our tech stack).
*   **How we use it:** We configure a mock service in GoAlert. When an alert triggers, GoAlert uses its native outbound webhook feature to send the incident payload directly to our ASM Platform API.

### 9.2 The Remediation Engine: ASM Platform (Our Tool)
*   **What it is:** Our durable agentic workflow orchestrator.
*   **How we use it:** ASM acts as the primary triage gateway. Grafana or Prometheus alerts are sent directly to ASM. The gatekeeper agent analyzes the incoming alert, triggers the appropriate read-only AgSOP (like `db-cpu-spike-investigator`), and attempts to identify the culprit. If manual SRE intervention is required (e.g., in a HITL state or when triage fails), ASM triggers GoAlert to page the on-call engineer and sends interactive options to Mattermost.

### 9.3 The ChatOps / Headless Approval Layer: Mattermost
*   **What it is:** [Mattermost](https://mattermost.com/) is a highly popular open-source alternative to Slack.
*   **How we use it:** When ASM enters a HITL state, it posts an alert directly to Mattermost with interactive buttons. SREs can approve recommendations (e.g. approving a query optimization hint) with one click from their mobile or desktop chat app.

### 9.4 The Alert Routing & Escalation Layer: GoAlert
*   **What it is:** [GoAlert](https://goalert.me/) is our open-source on-call scheduling and paging tool.
*   **How we use it:** If ASM determines that the alert cannot be auto-triaged, or if a critical diagnostic report is generated that requires active paging, ASM uses GoAlert's Inbound API to create an incident and page the current on-call engineer according to rotation schedules.

### 9.5 Testing the Concept Locally
We can spin all three of these up locally using Docker Compose:

```yaml
# docker-compose.yml example snippet
services:
  goalert:
    image: goalert/goalert:v0.34.0
    ports:
      - "8094:8081"
  mattermost:
    image: mattermost/mattermost-team-edition:9.5
    ports:
      - "8065:8065"
```

```bash
docker-compose up goalert mattermost
# Run ASM Platform locally
```

This stack allows us to completely simulate the modern auto-remediation firewall topology. We prove that alerts from monitoring can trigger ASM first, keeping GoAlert quiet unless a high-priority on-call page is dynamically escalated.

### 9.5 End-to-End Demo Setup Instructions

#### Step 1: Start the Demo Containers
Run the following command to boot Grafana and Mattermost alongside their respective databases in the background:
```bash
docker compose -f docker-compose.demo.yaml up -d
```
*   **Grafana Console:** http://localhost:3000
*   **Mattermost Workspace:** http://localhost:8065

#### Step 2: Configure Grafana (Alert Source Webhook)
1. Open http://localhost:3000 in your browser.
2. Log in with the default credentials:
   - **Username:** `admin`
   - **Password:** `admin`
   *(You can skip the password change prompt).*
3. Go to **Alerting** $\rightarrow$ **Contact points** in the left sidebar menu, and click **Add contact point**.
4. Configure the contact point:
   - **Name:** `ASM Auto-Remediation`
   - **Integration:** `Webhook`
   - **URL:** `http://host.docker.internal:8080/api/v1/triggers/webhook/00000000-0000-0000-0000-000000000001/devops-gatekeeper/v1/acknowledged`
5. *(Optional)* Click **Test** to send a test alert payload to ASM. This will immediately trigger the `devops-gatekeeper` AgSOP workflow! Click **Save contact point** when finished.
#### Step 3: Configure Mattermost (Outbound ChatOps)
1. Open http://localhost:8065, create a team, and open the `Town Square` channel.
2. Go to the Main Menu $\rightarrow$ **Integrations** $\rightarrow$ **Incoming Webhooks** $\rightarrow$ **Add Incoming Webhook**:
   - **Title:** `ASM Incident Alert Bot`
   - **Channel:** `Town Square`
3. Copy the generated Mattermost Webhook URL (e.g., `http://localhost:8065/hooks/...`).
4. Map this URL into ASM's environment variables or secrets manager under `SLACK_BOT_TOKEN` or `MATTERMOST_WEBHOOK_URL` so the agent can post the interactive approval button directly into the channel when the `hitl` state triggers.

#### Step 4: Run the Remediation Flow
To simulate a production incident triggering ASM:
1. You can trigger the flow by clicking the **Test** button in the Grafana Contact Point dialog. (In the Grafana test payload modal, you can optionally customize the JSON payload to include `"incident_id": "INC-893"`, `"service_name": "auth-service"`, and `"alert_title": "CrashLoopBackOff: pod/auth-service-5942f is restarting constantly"`).
2. Alternatively, post the incident alert payload directly to the ASM Webhook endpoint using `curl`:
   ```bash
   curl -X POST -H "Content-Type: application/json" -d '{
     "incident_id": "INC-893",
     "service_name": "auth-service",
     "alert_title": "CrashLoopBackOff: pod/auth-service-5942f is restarting constantly"
   }' http://localhost:8080/api/v1/triggers/webhook/00000000-0000-0000-0000-000000000001/devops-gatekeeper/v1/acknowledged
   ```
3. **What happens next:**
   - The `devops-gatekeeper` workflow launches automatically.
   - The gatekeeper agent triages the payload, sees `CrashLoopBackOff` and `auth-service`, and dynamically spawns the child subprocess workflow: `k8s-pod-crash-resolver`.
   - The resolver agent collects the pod logs (read-only), scans GitHub commits, identifies the faulty code commit, and constructs a patch.
   - It enters the `REQUEST_APPROVAL` HITL state and fires an interactive block with an **Approve / Reject** button to your Mattermost `Town Square` channel.
   - Click **Approve** directly inside Mattermost. It calls back to ASM, resolving the state and triggering the write-actions to deploy the fix!

## 10. SRE MCP Server Requirements
To build a fully functional SRE auto-remediation platform, we need to map our target MCP tool ecosystem. Below is a comparison of the SRE-relevant MCP servers we have already implemented versus the missing servers required for a production-grade rollout.

### 10.1 Already Implemented SRE-Relevant Servers
*   **github:** Reading files, viewing diffs, checking commit history, and creating PRs.
*   **postgres & redis:** Direct database inspection and query execution (sandboxed with scopes).
*   **slack:** Posting alert summaries and orchestrating approval alerts.
*   **filesystem & playwright:** Reviewing local directories and orchestrating web console diagnostics.

### 10.2 Missing SRE-Relevant Servers (To Be Implemented)
*   **kubernetes:** Interacting with clusters (`kubectl` operations) to fetch pod logs, describe resources, and trigger restarts.
*   **aws / gcp / azure:** Cloud-native provider access to view resource statuses, fetch instances, and restart virtual machines.
*   **datadog / splunk / elasticsearch:** Direct log and trace retrieval to allow the SRE agent to diagnose the root cause of an anomaly.
*   **prometheus / grafana:** Fetching timeseries metrics to determine resource trends (e.g. Memory/CPU leaks).
*   **pagerduty / opsgenie / jira:** SRE incident updates (e.g. creating tickets, updating incident statuses, adding notes).
*   **mysql:** Database interaction for MySQL-backed applications.

---

## 11. Monitoring & Alerting Tool Integrations
The ASM Platform operates as an automated remediation plugin. It integrates with industry-standard monitoring and alerting tools (e.g., Splunk On-Call/VictorOps, Atlassian Opsgenie/Jira Service Management, PagerDuty, Grafana, Datadog) using two distinct integration paths:

### 11.1 Inbound Webhook Routing (Incident Triage)
Every major alerting system supports outbound webhooks when an alert state transitions (e.g., `Triggered`, `Acknowledged`, `Resolved`).
*   **Integration Method:** We expose public webhook endpoints `/api/v1/triggers/webhook/:tenant_id/...` matching the alert provider's payload structure.
*   **VictorOps (Splunk On-Call):** We parse the webhook payload's JSON structure, extracting fields like `message_type` (CRITICAL), `entity_id` (Database CPU), and routing it to the appropriate AgSOP.
*   **Atlassian Opsgenie:** Parses the incoming Opsgenie event JSON (e.g. alert description, priority, tags) and kicks off a corresponding diagnostic workflow.
*   **PagerDuty & Grafana:** Native payloads are mapped into the Triage Router using simple jsonpath extraction schemas to populate the Blackboard.

### 11.2 Outbound Incident Management (via Alerting MCP Servers)
When an AgSOP runs, it needs to update the source incident management tool to keep human operators aligned.
*   **Integration Method:** We use dedicated alerting MCP servers (e.g. PagerDuty MCP, Jira Service Desk MCP).
*   **Actionable Tools:** The SRE agent can invoke tools like:
    - `add_incident_note(incident_id, "ASM Agent is investigating logs...")`
    - `ack_incident(incident_id)`
    - `transition_ticket_status(ticket_id, "In Progress")`
    - `resolve_incident(incident_id, "Remediation applied successfully: index created.")`

---

## 12. Competitor Analysis (SRE Automation Space)
Unlike generic integration workflows, the auto-remediation space is highly specialized. The following competitors represent the primary landscape:

1.  **Resolve AI**
    *   *What it is:* Closed-source AI agents designed to act as on-call engineers.
    *   *The Gap:* Resolve runs entirely in the cloud and has full write capabilities by default. ASM differentiates by offering **Model Context Protocol (MCP)** sandboxing with strict local scopes, keeping infrastructure access fully secure and audited locally.
2.  **Rootly & incident.io**
    *   *What it is:* incident management portals with automation rules (e.g., "if PagerDuty fires, create a Slack channel").
    *   *The Gap:* These tools are deterministic (if-this-then-that) and lack native LLM-based reasoning and automated diagnostic agents. ASM is AI-native and can reason through complex logs to formulate a fix.
3.  **PagerDuty Operations Cloud / AIOps**
    *   *What it is:* Enterprise incident routing and alert noise reduction.
    *   *The Gap:* Focuses on routing and alert consolidation, not execution. It tells you there is a problem; ASM actually *remediates* the problem.
4.  **Shoreline.io**
    *   *What it is:* Runbook automation executing shell commands across servers.
    *   *The Gap:* Requires SREs to manually write bash/python remediation scripts. ASM dynamically creates scripts, validates them in dry-run mode, and asks for approval via ChatOps without requiring pre-written code for every failure mode.

---

## 13. Trust Ramp: Diagnostics-First Strategy
To drive enterprise adoption, the ASM Platform positions itself with a **Diagnostics-First** trust ramp. 

Instead of requiring risky write-access to deploy code or mutate infrastructure in v1, the workflow templates focus purely on read-only diagnostics and alerting:
*   **Security Alignment:** By requesting only `read-only` scopes for Kubernetes, databases, cloud APIs, and Git repositories, security compliance groups can approve the tool instantly.
*   **Root-Cause Hints:** LLM agents focus on parsing logs, correlating metric time-series, and comparing git diffs to generate a high-precision diagnostic report (e.g., *"This database lock is caused by PID 1024 running an un-indexed query from John's last commit on `users.go:120`"*).
*   **Human Action:** Rather than executing the fix automatically, the agent posts this detailed context block directly to Mattermost/Slack, leaving the physical execution of the remedy to the human engineer. This ensures zero risk of automated hallucinations in production.

## 14. Demo Stack Integration Guide (Mattermost Setup)
To run the SRE gatekeeper demo locally, we utilize a pre-configured integration stack featuring Grafana, GoAlert, and Mattermost.

### Docker Compose Configuration
The demo containers are managed via `docker-compose.demo.yaml`. This configuration includes the required environment variables to unlock the Mattermost API for MCP-based bots out of the box:
```yaml
  mattermost:
    image: mattermost/mattermost-team-edition:9.5
    environment:
      # Enables the local Unix socket administration interface
      MM_SERVICESETTINGS_ENABLELOCALMODE: "true"
      # Enables Personal Access Tokens (User Access Tokens) for external API/MCP clients
      MM_SERVICESETTINGS_ENABLEUSERACCESSTOKENS: "true"
```

### Setup Steps & Token Generation
1. **Boot the Demo Services:**
   Run the following command to launch Grafana and Mattermost:
   ```bash
   make demo
   ```
2. **Access Mattermost & Create Admin User:**
   - Open [http://localhost:8065](http://localhost:8065) in your browser.
   - Create your administrator user account (e.g. username `bauhaus28`).
   - Complete the team setup (e.g. name the team `Chain Nodes` which resolves to slug `chain-nodes`).
3. **Generate Personal Access Token (PAT):**
   Execute `mmctl` inside the container as the `mattermost` user to create a token for your admin user:
   ```bash
   docker exec -u mattermost chainnode-mattermost mmctl token generate --local bauhaus28 "mcp-token"
   ```
   *Note: This returns the generated token (e.g., `8xx1wdgwrpyqtepgnnhq8rfn8a`).*
4. **Register the Token in the Platform:**
   - In the Visual Designer dashboard, navigate to **MCP Server Configurations**.
   - Input the following configuration details:
     - **`MATTERMOST_URL`**: `http://localhost:8065/api/v4`
     - **`MATTERMOST_TOKEN`**: The generated token from step 3.
     - **`MATTERMOST_TEAM_ID`**: The ID of the team (retrieve via DB query: `docker exec chainnode-mattermost-postgres psql -U mmuser -d mattermost -c "SELECT id FROM teams;"`).

