# Process Description: Gmail Inbox Sorter & Auto-Drafting Assistant

## Abstract
Connects directly to your Google Mail inbox to retrieve recent unread emails, analyzes each email's sender, context, and urgency to assign an importance ranking, and drafts polite, context-aware email replies for highly important messages, ready for maintainer approval.

## Blackboard Schema Requirements
The blackboard must hold the following strictly typed variables to track state:
- `max_emails_to_fetch` (number): The limit of unread messages to retrieve (defaults to 10).
- `unread_emails` (list of objects): A structured array of fetched emails containing `id`, `sender`, `subject`, `date`, and `body`.
- `sorted_inbox` (list of objects): The fetched emails annotated with `importance` (e.g. "high", "medium", "low") and a short `triage_reason`.
- `high_importance_count` (number): Counter tracking how many critical emails require drafting.
- `draft_replies` (object): Key-value pairs mapping email `id`s to their generated draft reply text.
- `current_processing_index` (number): Iterator pointer for looping through emails.

## Step-by-Step Workflow Specification

1. **Start State (Fetch Gmail Inbox):**
   - Name: `fetch_unread_emails`
   - Type: `prompt`
   - Target Agent: `InboxAgent`
   - Description: Connect to the Google Mail MCP server and execute the `list_messages` or `get_unread_emails` tool, limiting the count to `max_emails_to_fetch`. Populate `unread_emails` with the subject, sender address, and body content for each message.
   - Transitions:
     - On successful retrieval (`trigger: success`), move to `triage_and_sort_emails`.
     - On connection error (`trigger: error`), transition to `terminal_failure`.

2. **Analysis State (Importance Triage):**
   - Name: `triage_and_sort_emails`
   - Type: `prompt`
   - Target Agent: `InboxAgent`
   - Description: Iterate through `unread_emails`. Assess sender authority, urgency signals, and the message content. For each email, assign an `importance` rating ("high", "medium", "low") and write a brief `triage_reason`. Store the resulting list in `sorted_inbox` and compute the total `high_importance_count`.
   - Transitions:
     - On triage completed (`trigger: done`), move to `check_drafting_requirements`.

3. **Routing State (Evaluate Loops):**
   - Name: `check_drafting_requirements`
   - Type: `code`
   - Description: Evaluates the `sorted_inbox`. If there are emails marked "high" that do not yet have a draft in `draft_replies`, set `current_processing_index` to the index of the next pending important email.
   - Transitions:
     - If an email needs drafting (`trigger: draft_needed`), transition to `draft_reply_for_email`.
     - If all high-importance emails are drafted (`trigger: complete`), transition to `maintainer_inbox_review`.

4. **Drafting State (Draft Reply):**
   - Name: `draft_reply_for_email`
   - Type: `prompt`
   - Target Agent: `WriterAgent`
   - Description: Read the current high-importance email indexed by `current_processing_index`. Draft a helpful, context-appropriate, and professional response addressing the sender's inquiries. Append the drafted text to `draft_replies` using the email `id` as the key.
   - Transitions:
     - On reply drafted (`trigger: done`), loop back to `check_drafting_requirements` to check the next email.

5. **Human-in-the-Loop Review State (Approval Panel):**
   - Name: `maintainer_inbox_review`
   - Type: `hitl`
   - Assignee: `maintainer`
   - Instructions: "Review the sorted email list and the drafted responses generated for high-importance messages. Adjust draft text, approve drafts for sending, or discard drafts."
   - Form Schema:
     - Display the list of `sorted_inbox` emails showing subject and importance.
     - Display draft reply boxes allowing inline editing of the drafted texts.
   - Transitions:
     - If the maintainer approves sending drafts (`trigger: approve_and_send`), transition to `send_approved_emails`.
     - If the maintainer closes the dashboard without sending (`trigger: close`), transition to `terminal_success`.

6. **Execution State (Send Emails):**
   - Name: `send_approved_emails`
   - Type: `prompt`
   - Target Agent: `InboxAgent`
   - Description: Connect to the Gmail MCP server. For each approved draft in `draft_replies`, execute the `create_draft` or `send_message` tool to transmit the final reply text to the original sender.
   - Transitions:
     - On emails sent successfully (`trigger: done`), transition to `terminal_success`.

7. **Terminal States:**
   - `terminal_success` (Type: `terminal`): Unread inbox triaged, high-importance messages drafted, and replies successfully created/sent.
   - `terminal_failure` (Type: `terminal`): Failed to authenticate with Google Mail or retrieve inbox items.
