/**
 * Converts a Blackboard Schema (map of FieldDefs) to a standard JSON Schema (v7).
 * This allows the JsonSchemaForm component to render structured inputs for the blackboard.
 * 
 * @param {Object} bbSchema - The blackboard schema from the workflow definition.
 * @returns {Object|null} A standard JSON Schema object, or null if no schema is provided.
 */
export function convertBlackboardSchemaToJsonSchema(bbSchema, onlyRequired = false) {
  if (!bbSchema || Object.keys(bbSchema).length === 0) return null

  const properties = {}
  const required = []

  for (const [key, field] of Object.entries(bbSchema)) {
    if (onlyRequired && !field.required) continue

    properties[key] = convertField(field, key)
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

function convertField(field, key = '') {
  let type = field.type || 'string'
  let items = undefined
  let properties = undefined
  let required = undefined

  if (type === 'bool') type = 'boolean'
  if (type === 'file') type = 'file' // Keep as 'file' for our custom SchemaField handler
  
  if (type === 'list' || type === 'array') {
    type = 'array'
    if (field.items) {
      items = convertField(field.items)
    } else {
      items = { type: 'string' }
    }
  }

  if (type === 'object' && field.properties) {
    const subSchema = convertBlackboardSchemaToJsonSchema(field.properties)
    if (subSchema) {
      properties = subSchema.properties
      required = subSchema.required
    }
  }

  return {
    type,
    items,
    properties,
    required,
    title: field.title || (key ? key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()) : undefined),
    default: field.default
  }
}
