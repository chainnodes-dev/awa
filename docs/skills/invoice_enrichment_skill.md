# Skill: Global Invoice Enrichment & Tax Compliance

This process describes the end-to-end handling of incoming business invoices, ensuring they are enriched with required regulatory identifiers, normalized for currency, and validated for tax compliance.

## 1. Input Requirements
The process begins when an invoice is received. To proceed, we must have the following information extracted:
- The customer or vendor name.
- The invoice reference number.
- The raw amount and its original currency.

## 2. Enrichment & Validation Steps

### Legal Entity Identification (GLEIF)
The first priority is to cross-reference the vendor name against the Global Legal Entity Identifier (GLEIF) database. 
- Try to automatically find and record the official LEI for the company.
- If the search returns multiple possible matches, or if no clear match can be found, pause the process and ask a human reviewer to select the correct entity from the list of suggestions.

### Global Market Data (OpenFIGI)
Check the line items and description for any references to financial instruments or stock tickers. If identified, use the OpenFIGI service to lookup the official FIGI codes and market identifiers. Mark the invoice as containing securities if matches are found.

### Currency Normalization (Exchange Rates)
If the invoice was issued in any currency other than Euro (EUR), the platform must convert the amount to Euro.
- Use the latest available exchange rates from the ECB (via the exchange rate service).
- Record both the converted Euro value and the specific rate used for the conversion on the process blackboard.

### Tax Calculation (VAT)
Once the amount is normalized to Euro, calculate a standard 19% Value Added Tax (VAT).
- Calculate the specific VAT amount.
- Calculate the final gross (brut) total by adding the VAT to the Euro amount.
- Store these results for downstream ERP entry.

## 3. Operational Constraints

### Time-to-Process (SLA)
To ensure system responsiveness, every individual step in this logic (e.g., waiting for an external model to reply or a data source to respond) must complete within **1 minute**. 

### Fail-Safe Handling
If any step exceeds the 1-minute threshold, the process should be considered "Timed Out." In this case, stop further processing and route the item to a terminal "Timeout" state for investigation.
