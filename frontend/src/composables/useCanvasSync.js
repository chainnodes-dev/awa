import { watch, nextTick } from 'vue'
import yaml from 'js-yaml'
import { useValidator } from './useValidator'

export function useCanvasSync({ nodes, edges, meta, isDirty, view, yamlSource, yamlErrors, flowKey, fromLoad, loadDefinition, buildDefinition }) {
  const { validate } = useValidator()

  // Prevents the canvas→YAML watch from firing when the canvas was updated
  // as a result of a YAML parse (would overwrite what the user typed).
  let fromYamlUpdate = false
  let yamlDebounce   = null

  // Canvas → YAML: sync textarea.
  // isDirty is NOT set here — real mutations set it explicitly.
  watch([nodes, edges], () => {
    if (fromYamlUpdate) return
    yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
  }, { deep: true })

  // meta → YAML + dirty: fires when user edits workflow metadata fields.
  // Suppressed during clean loads (fromLoad) and YAML-parse-triggered updates (fromYamlUpdate).
  watch(meta, () => {
    if (fromYamlUpdate || fromLoad.value) return
    isDirty.value = true
    yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
  }, { deep: true })

  // View switch: sync YAML ↔ canvas when changing modes.
  watch(view, (v, oldV) => {
    if (v === 'yaml' || v === 'split') {
      yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
    } else if (oldV === 'yaml' && v === 'canvas') {
      const errors = validate(yamlSource.value)
      if (errors.length > 0) {
        yamlErrors.value = errors
        alert('Workflow has validation errors. Please fix them before switching to Canvas.')
        view.value = 'yaml'
        return
      }
      const def = yaml.load(yamlSource.value)
      if (def) loadDefinition(def, null, false)
      yamlErrors.value = []
    }
  })

  // YAML → Canvas: debounced parse when user edits the YAML textarea in split view.
  function onYamlInput() {
    isDirty.value = true
    if (view.value !== 'split') return
    yamlErrors.value = []
    clearTimeout(yamlDebounce)
    yamlDebounce = setTimeout(() => {
      const errors = validate(yamlSource.value)
      yamlErrors.value = errors
      if (errors.length > 0) return

      try {
        const def = yaml.load(yamlSource.value)
        if (!def) return
        fromYamlUpdate = true
        loadDefinition(def, null, false)
        nextTick(() => { fromYamlUpdate = false })
      } catch (e) {
        // Should be caught by validate() above, but safety first
        yamlErrors.value = [{ message: e.message }]
      }
    }, 400)
  }

  return { onYamlInput }
}
