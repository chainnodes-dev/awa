/**
 * Converts an Phaxa Blackboard Schema (map of FieldDefs) to a standard JSON Schema (v7).
 * This allows the JsonSchemaForm component to render structured inputs for the blackboard.
 * 
 * @param {Object} phaxaSchema - The blackboard schema from the workflow definition.
 * @returns {Object|null} A standard JSON Schema object, or null if no schema is provided.
 */
export function convertPhaxaSchemaToJsonSchema(phaxaSchema, onlyRequired = false) {
  if (!phaxaSchema || Object.keys(phaxaSchema).length === 0) return null

  const properties = {}
  const required = []

  for (const [key, field] of Object.entries(phaxaSchema)) {
    if (onlyRequired && !field.required) continue

    // Basic mapping of Phaxa types to JSON Schema types
    let type = field.type || 'string'
    if (type === 'bool') type = 'boolean'

    properties[key] = {
      type,
      title: key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
      default: field.default
    }

    if (field.required) {
      required.push(key)
    }
  }

  if (Object.keys(properties).length === 0) return null

  return {
    type: 'object',
    properties,
    required,
    title: onlyRequired ? 'Required Inputs' : 'Blackboard Data'
  }
}
