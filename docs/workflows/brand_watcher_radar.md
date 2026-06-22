I want to create a workflow to monitor our brand and our competitors across the web and create a briefing on the blackboard.

1. First, search the web using Brave Search for recent mentions of a list of brand and competitor names (which will be passed in as a list). Gather the title, URL, publication date, and result snippet for each.
2. Filter the raw results to remove spam or noise. For each good mention, score its business relevance from 1 to 10 and determine if the sentiment is positive, negative, or neutral.
2a. Get information from SEC and include it into analysis if available
3. Have an LLM write a clean Markdown executive report that summarizes competitor momentum, breaks down positive mentions vs. negative complaints, and includes clickable links to the sources.
4. Scan the filtered mentions with a script. If there are any highly critical negative reviews (sentiment is negative and relevance score is 9 or 10), we need to escalate.
5. If there is an escalation, pause the workflow for human review. A maintainer needs to see the Markdown report in an editable textarea, be able to write critical incident override comments in a text input field, and approve the report before it's sent.
6. If anything fails or if the entire workflow takes longer than 5 minutes, it should time out and fail.
