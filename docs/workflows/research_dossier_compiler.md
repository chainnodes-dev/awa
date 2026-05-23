# Process Description: Intelligent Research & Dossier Compiler

## Abstract
Takes a complex academic, market, or historical research topic provided by the user, decomposes it into highly specific target sub-questions, executes parallel search pipelines via search and informational MCP servers, filters and validates facts, synthesizes the results, and writes a beautifully formatted Markdown dossier directly as a local file.

## Blackboard Schema Requirements
The blackboard must hold the following strictly typed variables to track state:
- `research_topic` (string, required): The comprehensive topic to investigate.
- `sub_questions` (list of strings): A list of decomposed research questions.
- `gathered_facts` (list of objects): Extracted facts, data points, and quotes, associated with source URLs.
- `final_synthesis` (string): The structured academic Markdown dossier text.
- `output_filepath` (string, required): Absolute file path to save the final dossier on your local machine.

## Step-by-Step Workflow Specification

1. **Start State (Research Decomposition):**
   - Name: `decompose_research_topic`
   - Type: `prompt`
   - Target Agent: `ResearchAgent`
   - Description: Analyze the `research_topic` provided on the blackboard. Decompose it into exactly 3-5 highly specific questions covering historical context, current state, and future outlook. Write these queries into `sub_questions`.
   - Transitions:
     - On successful decomposition (`trigger: success`), move to `query_information_sources`.

2. **Data Gathering State (Execute Parallel Search Pipelines):**
   - Name: `query_information_sources`
   - Type: `prompt`
   - Target Agent: `ResearchAgent`
   - Description: Iterate through the `sub_questions` list. For each question, query the Brave Search MCP server and Wikipedia MCP server to retrieve relevant informational resources. Extract facts, key statistics, and citations, saving them into `gathered_facts`.
   - Transitions:
     - On data gathered successfully (`trigger: success`), move to `synthesize_factual_dossier`.
     - On error (`trigger: error`), transition to `terminal_failure`.

3. **Synthesis State (Fact Filtering & Formatting):**
   - Name: `synthesize_factual_dossier`
   - Type: `prompt`
   - Target Agent: `WriterAgent`
   - Description: Read `gathered_facts`. Synthesize the findings into a comprehensive research report. Ensure the report has a logical academic flow, a table of contents, embedded headers, bold statistics, and a dedicated bibliography section listing all source URLs. Save this text to `final_synthesis`.
   - Transitions:
     - On synthesis complete (`trigger: done`), move to `write_dossier_to_file`.

4. **Execution State (Write Dossier to Desktop):**
   - Name: `write_dossier_to_file`
   - Type: `prompt`
   - Target Agent: `ResearchAgent`
   - Description: Connect to the Filesystem MCP server. Create a new markdown file at `output_filepath` and write the full contents of `final_synthesis` into it.
   - Transitions:
     - On file successfully saved (`trigger: done`), transition to `terminal_success`.
     - On file write permission error (`trigger: error`), transition to `terminal_failure`.

5. **Terminal States:**
   - `terminal_success` (Type: `terminal`): Research dossier compiled and successfully saved to the user's local filesystem.
   - `terminal_failure` (Type: `terminal`): Failed to access search APIs or save the final markdown file locally.
