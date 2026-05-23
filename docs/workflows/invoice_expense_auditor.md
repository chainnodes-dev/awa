# Process Description: Autonomous Invoice & Expense Auditor

## Abstract
Scans a target local directory for newly uploaded expense receipts or invoices, extracts crucial structured invoice fields using LLM multimodal capabilities, performs mathematical validation on the totals and tax rates via a JavaScript code node, and routes invoices with math errors to a Human-in-the-Loop review pane.

## Blackboard Schema Requirements
The blackboard must hold the following strictly typed variables to track state:
- `target_folder` (string, required): Absolute path to the local receipts folder.
- `unprocessed_files` (list of strings): Paths to invoice files waiting to be processed.
- `current_invoice_file` (string): Absolute file path of the invoice being audited.
- `extracted_vendor` (string): Name of the vendor.
- `extracted_date` (string): Transaction date.
- `extracted_items` (list of objects): Extracted line items containing `description`, `quantity`, `price`, and `tax`.
- `extracted_total` (number): The total invoice amount stated on the bill.
- `calculated_total` (number): The total calculated mathematically from the line items.
- `math_error` (boolean): Flag indicating if calculated total mismatches extracted total.
- `audit_approved` (boolean): Flag for ERP processing sign-off.

## Step-by-Step Workflow Specification

1. **Start State (List Directory Files):**
   - Name: `scan_target_folder`
   - Type: `prompt`
   - Target Agent: `AuditorAgent`
   - Description: Connect to the Filesystem MCP server. Scan `target_folder` and compile a list of all raw PDF, PNG, or JPG invoice files. Save this file list into `unprocessed_files`.
   - Transitions:
     - On directory scanned (`trigger: success`), move to `evaluate_queue`.
     - On read error (`trigger: error`), transition to `terminal_failure`.

2. **Routing State (Evaluate Queue Loop):**
   - Name: `evaluate_queue`
   - Type: `code`
   - Description: A lightweight script checks the length of `unprocessed_files`. 
     - If the list is empty, the run completes.
     - If files remain, pop the first file path, store it in `current_invoice_file`, and update `unprocessed_files`.
   - Transitions:
     - If a file is pending (`trigger: process_next`), transition to `extract_invoice_data`.
     - If the queue is empty (`trigger: empty`), transition to `terminal_success`.

3. **Extraction State (Multimodal Data Extraction):**
   - Name: `extract_invoice_data`
   - Type: `prompt`
   - Target Agent: `AuditorAgent`
   - Description: Read `current_invoice_file` using the Filesystem MCP server. The agent uses multimodal analysis to extract invoice metadata, populating `extracted_vendor`, `extracted_date`, `extracted_items` (including individual line items), and the stated `extracted_total`.
   - Transitions:
     - On extraction completed (`trigger: done`), move to `mathematical_audit`.

4. **Validation State (Deterministic Math Check):**
   - Name: `mathematical_audit`
   - Type: `code`
   - Description: A JavaScript script calculates the mathematical total by summing up `price * quantity` for all entries in `extracted_items` and adding the stated taxes. It stores this calculation in `calculated_total`. If `Math.abs(calculated_total - extracted_total) > 0.01`, set `math_error` to `true`, otherwise set it to `false`.
   - Transitions:
     - If math matches (`trigger: math_ok`), transition to `auto_approve_expense`.
     - If a discrepancy is found (`trigger: math_error`), transition to `audit_discrepancy_gate`.

5. **Human-in-the-Loop Review State (Discrepancy Correction Panel):**
   - Name: `audit_discrepancy_gate`
   - Type: `hitl`
   - Assignee: `accountant`
   - Instructions: "A math discrepancy was found on this invoice. The stated total does not match the sum of the line items. Please correct the values or flag the vendor."
   - Form Schema:
     - Provide editable numeric fields for `extracted_total` and `calculated_total`.
     - Display an editable JSON form showing all `extracted_items` line details.
   - Transitions:
     - On accountant submission (`trigger: resolved`), transition to `evaluate_queue` to process any remaining invoices.

6. **Execution State (Auto-Approve):**
   - Name: `auto_approve_expense`
   - Type: `prompt`
   - Target Agent: `AuditorAgent`
   - Description: Append the audited metadata, calculated totals, and file details to a master local spreadsheet `expenses_ledger.csv` using the Filesystem MCP server.
   - Transitions:
     - On complete (`trigger: done`), loop back to `evaluate_queue` to audit the next invoice.

7. **Terminal States:**
   - `terminal_success` (Type: `terminal`): All directory invoices successfully scanned, mathematically audited, and logged to the ledger.
   - `terminal_failure` (Type: `terminal`): Failed to access files or ledger spreadsheet.
