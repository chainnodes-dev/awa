Develop a "Corporate Financial Analysis" workflow with the following logic:

1. Initial Setup: The workflow starts by accepting a "quarterly_report" of type "file" (PDF).

2. Data Extraction: Use a 'prompt' node (ANALYSE_REPORT) to extract Revenue and Net Income from the report using the 'read_file' tool.

3. AI Sentiment Analysis: Use a 'prompt' node (AI_SENTIMENT) to gather current market sentiment via a Search MCP server.

4. Expert Review & Entry (HITL): Add a 'hitl' node (EXPERT_REVIEW). 
   - Assign to "analysts". 
   - Show the AI's findings (Revenue, Net Income, AI Sentiment).
   - FORM ENTRY: Add a required field "expert_sentiment" (enum: Positive, Neutral, Negative) for the human to enter their own judgment.
   - Triggers: "Approve" or "Reject".

5. Data Integration: 
   - If "Approve", use a 'prompt' node (UPDATE_SPREADSHEET) to store: Revenue, Net Income, AI_Sentiment, and EXPERT_SENTIMENT in Google Sheets.
   - If "Reject", transition to REJECTED.

6. Termination: Transition to SUCCESS after storage.

Technical Requirements:
- Use uppercase snake_case for state names.
- Ensure 'quarterly_report' is type 'file'.
- The 'EXPERT_REVIEW' node must define a 'form_schema' for the 'expert_sentiment' field.
- Use a global transitions block.
