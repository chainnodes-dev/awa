# Dividend Validation Workflow

This workflow ensures that dividend data is accurate and reliable before it impacts financial records. It combines automated checks with a critical human-in-the-loop (HITL) step for high-value transactions.

## Input data (Required Fields)
Capture the Instrument ID, Gross Amount, Currency, Ex-Dividend Date, Publication Date and Payable Date. These form the "Golden Record" template.

## Automated Validation

### Completenes Check
Ensure all required fields are populated and correctly formatted (e.g., dates in YYYY-MM-DD, amounts as positive numbers). If failed move to state rejected with reason "Incomplete or Invalid Data". Otherwise move to Logic Check.

### Logic Check
Confirm dates follow a linear timeline (Ex-Date < Payable Date). If failed move to state rejected with reason "Date Logic Error".

### High Value Check
If the Gross Amount > 2.00 USD, flag the record for HITL review. Otherwise, move to confirmed.

## HITL Step (The "High-Value" Filter)
Let Human reviewer approve or reject. If approved move to confirmed. If rejected move to state rejected with reason "HITL Rejection".
