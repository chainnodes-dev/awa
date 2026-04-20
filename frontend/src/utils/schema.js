/**
 * Converts an ASM Blackboard Schema (map of FieldDefs) to a standard JSON Schema (v7).
 * This allows the JsonSchemaForm component to render structured inputs for the blackboard.
 * 
 * @param {Object} asmSchema - The blackboard schema from the workflow definition.
 * @returns {Object|null} A standard JSON Schema object, or null if no schema is provided.
 */
export function convertAsmSchemaToJsonSchema(asmSchema) {
  if (!asmSchema || Object.keys(asmSchema).length === 0) return null

  const properties = {}
  const required = []

  for (const [key, field] of Object.entries(asmSchema)) {
    // Basic mapping of ASM types to JSON Schema types
    let type = field.type || 'string'
    if (type === 'bool') type = 'boolean'

    properties[key] = {
      type,
      // Create a readable title from the field name (e.g., "invoice_id" -> "Invoice Id")
      title: key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
      default: field.default
    }

    // Add to required list if the field is marked as required in ASM
    if (field.required) {
      required.push(key)
    }
  }

  return {
    type: 'object',
    properties,
    required,
    // Add a default title if none provided
    title: 'Blackboard Data'
  }
}
