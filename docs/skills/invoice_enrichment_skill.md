# Skill: Global Invoice Enrichment & Tax Compliance

This process describes the end-to-end handling of incoming business invoices, ensuring they are enriched with required regulatory identifiers, normalized for currency, and validated for tax compliance.

## 1. Input Requirements
The process begins when an invoice is received. The following parameter are required to start the process:
- text: text object containing the unstructered invoice. This is the only input field, all other values are added as part of the process.

To proceed, we must first evaluate if the passed text is an invoice or not. An invoice must contain the following data:
- customer: The customer name.
- invoice_number: The invoice reference number.
- items: At least one item entry with item name, quantity and unit price.
- total_amount: The total amount of the invoice.
These values must be extracted and stored on the blackboard.

The invoice might also contain the following optional fields that should be extracted if present:
- currency: Currency
- vat_id: VAT ID
- payment_terms: Payment terms
- payment_method: Payment method
- shipping_information: Shipping information
- billing_information: Billing information
- discount: discount information
- tax_information: Tax information

Stop the process if not all mandatory fields are present. Otherwise start validation of invoice-content. Present the extracted data in a structured format and ask user for confirmation (HITL).

## 2. Enrichment & Validation Steps

### Legal Entity Identification (GLEIF)
The first priority is to cross-reference the vendor name against the Global Legal Entity Identifier (GLEIF) database. 
- Try to automatically find and record the official LEI for the company.
- If the search returns multiple possible matches, or if no clear match can be found, pause the process and ask a human reviewer to select the correct entity from the list of suggestions.
- Use the available GLEIF MCP server to find the LEI.

### Currency Normalization (Exchange Rates)
If no currency is available in the data set the currency to Euro (EUR)
If the invoice was issued in any currency other than Euro (EUR), the platform must convert the amount to Euro.
- Use the latest available exchange rates from the ECB (via the exchange rate service).
- Record both the converted Euro value and the specific rate used for the conversion on the process blackboard.
- Use the available exchange rate MCP server to get the exchange rate.

### Tax Calculation (VAT)
Once the amount is normalized to Euro, calculate a standard 19% Value Added Tax (VAT).
- Calculate the specific VAT amount.
- Calculate the final gross (brut) total by adding the VAT to the Euro amount.
- Store these results for downstream ERP entry.

## 3. Operational Constraints

### Time-to-Process (SLA)
To ensure system responsiveness, every individual step in this logic (e.g., waiting for an external model to reply or a data source to respond) must complete within **1 minute**. 

### Fail-Safe Handling
If any step exceeds the 3-minute threshold, the process should be considered "Timed Out." In this case, stop further processing and route the item to a terminal "Timeout" state for investigation. Exclude HITL nodes from this timeout.
