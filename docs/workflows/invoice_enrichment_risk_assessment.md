# Process Description: Invoice Enrichment & Customer Default Risk Assessment

## Goal
Automate the enrichment of incoming invoice details and perform a default risk assessment on the customer company using local and public data sources.

## Data Schema (Blackboard)

### Inputs
The process starts with:
- `invoice_id` (string, required)
- `company_name` (string, required)
- `amount` (number, required)
- `company_address` (string, optional)
- `lei` (string, optional)

### Outputs & Intermediate Fields
The workflow should populate the following:
- `lei_candidates` (array of objects): Used for storing possible matches during resolution.
- `lei_details` (object): Official entity records (legal name, address, country, status).
- `wikipedia_summary` (string): Context from Wikipedia.
- `sec_filings_summary` (string): Summary of recent SEC filings.
- `enrichment_notes` (string): Audit trail of operations.
- `risk_score` (string): Final default risk rating (`LOW`, `MEDIUM`, `HIGH`).
- `risk_reasoning` (string): Justification for the risk score.
- `final_enriched_invoice` (object): A consolidated report payload combining invoice inputs, corporate details, and risk metrics.

---

## Process Flow

1. **Input Validation**: Check that the mandatory inputs (`invoice_id`, `company_name`, `amount`) are present.
2. **LEI Resolution**: 
   - If an `lei` is already provided, skip resolution.
   - If `lei` is missing, search for the company's LEI using the GLEIF API (`lei` MCP server).
   - If exactly one LEI is found, use it.
   - If multiple candidates are found, pause and ask a human to select the correct corporate entity.
   - If no LEI is found, note the search failure in `enrichment_notes` and proceed without it.
3. **Data Enrichment**:
   - If an LEI is available, retrieve the official entity name, country, registered address, and status.
   - Search SEC EDGAR filings for financial facts and a list of recent files (e.g. 10-K, 10-Q) using the company name or LEI.
4. **Risk Evaluation**:
   - Assess the company's financial risk based on the retrieved status, SEC filing summary, Wikipedia summary, and the invoice amount.
   - Produce a default risk score (`LOW`, `MEDIUM`, or `HIGH`) and write a short reasoning.

## General Rules
- Every node in the workflow should have a timeout of **2 minutes** (`120s`) to prevent hangs, with transitions defining what happens if a timeout occurs.
- Assign appropriate domain agents (e.g., Accountant, Researcher, Risk Analyst) to different parts of the flow.
