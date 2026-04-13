import { watch, nextTick } from 'vue'
import yaml from 'js-yaml'

export function useCanvasSync({ nodes, edges, meta, isDirty, view, yamlSource, yamlError, flowKey, fromLoad, loadDefinition, buildDefinition }) {

  // Prevents the canvas→YAML watch from firing when the canvas was updated
  // as a result of a YAML parse (would overwrite what the user typed).
  let fromYamlUpdate = false
  let yamlDebounce   = null

  // Canvas → YAML: sync textarea in split view.
  // isDirty is NOT set here — real mutations set it explicitly.
  watch([nodes, edges], () => {
    if (fromYamlUpdate) return
    if (view.value === 'split') {
      yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
    }
  }, { deep: true })

  // meta → YAML + dirty: fires when user edits workflow metadata fields.
  // Suppressed during clean loads (fromLoad) and YAML-parse-triggered updates (fromYamlUpdate).
  watch(meta, () => {
    if (fromYamlUpdate || fromLoad.value) return
    isDirty.value = true
    if (view.value === 'split') {
      yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
    }
  }, { deep: true })

  // View switch: sync YAML ↔ canvas when changing modes.
  watch(view, (v, oldV) => {
    if (v === 'yaml' || v === 'split') {
      yamlSource.value = yaml.dump(buildDefinition(), { lineWidth: 120 })
    } else if (oldV === 'yaml' && v === 'canvas') {
      try {
        const def = yaml.load(yamlSource.value)
        if (def) loadDefinition(def, null, false)
        yamlError.value = ''
      } catch (e) {
        yamlError.value = e.message
        alert('YAML Syntax Error: ' + e.message + '\n\nReturning to editor.')
        view.value = 'yaml'
      }
    }
  })

  // YAML → Canvas: debounced parse when user edits the YAML textarea in split view.
  function onYamlInput() {
    isDirty.value = true
    if (view.value !== 'split') return
    yamlError.value = ''
    clearTimeout(yamlDebounce)
    yamlDebounce = setTimeout(() => {
      try {
        const def = yaml.load(yamlSource.value)
        if (!def) return
        fromYamlUpdate = true
        loadDefinition(def, null, false)
        nextTick(() => { fromYamlUpdate = false })
        yamlError.value = ''
      } catch (e) {
        yamlError.value = e.message
      }
    }, 400)
  }

  return { onYamlInput }
}
