import { nextTick } from 'vue'

const HANDOFF_KEY = 'chainnodes_designer_handoff'
const HANDOFF_TTL_MS = 5 * 60 * 1000

function consumeHandoff() {
  const raw = sessionStorage.getItem(HANDOFF_KEY)
  if (!raw) return null
  sessionStorage.removeItem(HANDOFF_KEY)
  try {
    const data = JSON.parse(raw)
    if (!data?.definition) return null
    if (data.timestamp && Date.now() - data.timestamp > HANDOFF_TTL_MS) {
      return null
    }
    return data
  } catch {
    return null
  }
}

function targetDesignerPath(targetMode, { routeParams, meta }) {
  const prefix = targetMode === 'ai' ? '/ai-designer' : '/designer'
  const routeName = routeParams?.name && routeParams.name !== 'new' ? routeParams.name : ''
  const routeVersion = routeParams?.version || ''
  const name = routeName || (meta?.versionNumber ? meta?.name : '')
  const version = routeVersion || (meta?.versionNumber ? meta?.version : '')
  if (name && version) {
    return `${prefix}/${encodeURIComponent(name)}/${encodeURIComponent(version)}`
  }
  return prefix
}

/**
 * Shared state handoff composable for seamlessly switching between
 * AI Designer (vertical) and Classic Designer (horizontal) without losing
 * unsaved edits, node positions, or AI generation context.
 */
export function useDesignerHandoff({
  route,
  router,
  buildDefinition,
  loadDefinition,
  meta,
  yamlSource,
  versions,
  isDirty,
  showTimeoutEdges,
  selected,
  nodes,
  edges,
  flowKey,
  fitView,
  generatePrompt,
  generateHistory,
  generateInteractions,
  generateProvider,
  generatePipelineStep,
  generatePipelineWarning,
  onBeforeRestore,
  onAfterRestore,
}) {
  function switchToMode(targetMode) {
    const definition = buildDefinition()
    if (meta.versionNumber && definition?.metadata) {
      definition.metadata.version_number = meta.versionNumber
    }
    const payload = {
      definition,
      yamlSource: yamlSource.value ?? '',
      versionNumber: meta.versionNumber || 0,
      versions: Array.isArray(versions.value) ? versions.value : [],
      isDirty: !!isDirty.value,
      showTimeoutEdges: showTimeoutEdges.value ?? true,
      selectedId: selected.value?.id || null,
      selectedType: selected.value?.type || null,
      generateState: {
        prompt: generatePrompt.value,
        history: generateHistory.value,
        interactions: generateInteractions.value,
        provider: generateProvider.value,
        pipelineStep: generatePipelineStep.value,
        pipelineWarning: generatePipelineWarning.value,
      },
      timestamp: Date.now(),
    }
    sessionStorage.setItem(HANDOFF_KEY, JSON.stringify(payload))
    router.push(targetDesignerPath(targetMode, { routeParams: route.params, meta }))
  }

  async function restoreHandoffIfPresent() {
    const handoff = consumeHandoff()
    if (!handoff) return false

    onBeforeRestore?.()

    loadDefinition(handoff.definition, handoff.yamlSource)
    if (handoff.versionNumber) {
      meta.versionNumber = handoff.versionNumber
    }
    versions.value = handoff.versions ?? []
    showTimeoutEdges.value = handoff.showTimeoutEdges ?? true

    if (handoff.generateState) {
      generatePrompt.value = handoff.generateState.prompt ?? ''
      generateHistory.value = handoff.generateState.history ?? []
      generateInteractions.value = handoff.generateState.interactions ?? []
      generateProvider.value = handoff.generateState.provider ?? ''
      generatePipelineStep.value = handoff.generateState.pipelineStep ?? 0
      generatePipelineWarning.value = handoff.generateState.pipelineWarning ?? ''
    }

    // Wait two ticks so loadDefinition's nextTick clears fromLoad and deep watchers settle
    await nextTick()
    await nextTick()

    if (handoff.selectedId && handoff.selectedType) {
      if (handoff.selectedType === 'node') {
        const node = nodes.value.find(n => n.id === handoff.selectedId)
        if (node) selected.value = { type: 'node', id: node.id, data: { ...node.data } }
      } else if (handoff.selectedType === 'edge') {
        const edge = edges.value.find(e => e.id === handoff.selectedId)
        if (edge) {
          selected.value = {
            type: 'edge',
            id: edge.id,
            data: { label: edge.label, guard: edge.data?.guard ?? '', from: edge.source, to: edge.target },
          }
        }
      }
    }

    flowKey.value++
    await nextTick()
    fitView({ padding: 0.15, duration: 200 })

    // Wait a tick for Vue Flow mount updates before restoring final dirty state
    await nextTick()
    isDirty.value = !!handoff.isDirty
    await nextTick()
    onAfterRestore?.()
    return true
  }

  return {
    switchToMode,
    restoreHandoffIfPresent,
  }
}
