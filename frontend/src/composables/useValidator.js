import yaml from 'js-yaml'

export function useValidator() {
  function validate(yamlStr) {
    const errors = []
    let data = null

    try {
      data = yaml.load(yamlStr)
    } catch (e) {
      return [{ message: e.message, line: e.mark?.line + 1 }]
    }

    if (!data) return []

    // 1. Basic Structure
    if (data.kind !== 'Workflow') {
      errors.push({ message: 'Root "kind" must be "Workflow"', path: 'kind' })
    }
    if (!data.metadata?.name) {
      errors.push({ message: 'Workflow name is missing (metadata.name)', path: 'metadata.name' })
    }

    // 2. States
    const stateNames = new Set()
    if (!Array.isArray(data.states)) {
      errors.push({ message: '"states" must be an array', path: 'states' })
    } else {
      data.states.forEach((s, i) => {
        if (!s.name) {
          errors.push({ message: `State #${i+1} is missing a "name"`, path: `states[${i}]` })
        } else {
          stateNames.add(s.name)
        }
        
        if (!s.type) {
          errors.push({ message: `State "${s.name || i}" is missing a "type"`, path: `states[${i}].type` })
        }

        if (s.type === 'skill_call' && !s.subprocess) {
          errors.push({ message: `Skill call "${s.name}" must specify a "subprocess"`, path: `states[${i}].subprocess` })
        }
      })
    }

    // 3. Transitions
    if (data.transitions && Array.isArray(data.transitions)) {
      data.transitions.forEach((t, i) => {
        if (!t.from) {
          errors.push({ message: `Transition #${i+1} is missing "from"`, path: `transitions[${i}].from` })
        } else if (!stateNames.has(t.from)) {
          errors.push({ message: `Transition from unknown state "${t.from}"`, path: `transitions[${i}].from` })
        }

        const targets = t.to_nodes || (t.to ? [t.to] : [])
        if (targets.length === 0) {
          errors.push({ message: `Transition #${i+1} has no destination ("to" or "to_nodes")`, path: `transitions[${i}]` })
        } else {
          targets.forEach(target => {
            if (!stateNames.has(target)) {
              errors.push({ message: `Transition to unknown state "${target}"`, path: `transitions[${i}]` })
            }
          })
        }
      })
    }

    return errors
  }

  return { validate }
}
