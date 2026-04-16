import { nextTick } from 'vue'
import yaml from 'js-yaml'

export function useWorkflowIO({ nodes, edges, meta, bbSchema, agents, versions, isDirty, yamlSource, yamlError, saving, flowKey, showRunModal, view, wfStore, router }) {

  // Plain flag — not reactive; checked synchronously in watch(meta) inside useCanvasSync.
  const fromLoad = { value: false }

  function loadDefinition(def, src, updateYaml = true) {
    // Set the guard flag BEFORE any reactive mutations.
    fromLoad.value = true

    if (src && updateYaml) yamlSource.value = src

    meta.name          = def.metadata?.name           ?? ''
    meta.version       = def.metadata?.version        ?? '1.0.0'
    meta.versionNumber = def.metadata?.version_number ?? 0
    meta.description   = def.metadata?.description    ?? ''
    meta.systemPrompt  = def.metadata?.system_prompt  ?? ''

    for (const k in bbSchema) delete bbSchema[k]
    if (def.blackboard?.schema) Object.assign(bbSchema, def.blackboard.schema)

    nodes.value = (def.states ?? []).map(s => ({
      id:       s.name,
      type:     'stateNode',
      position: s.position ?? { x: 200, y: 100 },
      data:     { ...s },
    }))

    // Build the set of triggers that correspond to timeout / condition exception paths.
    // Used to tag edges so the canvas can style and toggle them independently.
    const exceptionTriggers = new Set(
      (def.states ?? []).flatMap(s => [s.on_timeout, s.on_condition].filter(Boolean))
    )

    const rawTransitions = [...(def.transitions ?? [])]
    if (def.states) {
      def.states.forEach(s => {
        if (s.transitions) rawTransitions.push(...s.transitions.map(t => ({ ...t, from: s.name })))
      })
    }

    edges.value = rawTransitions.flatMap(t => {
      const isTimeout = exceptionTriggers.has(t.trigger)
      if (t.to_nodes?.length > 0) {
        return t.to_nodes.map(target => ({
          id: `${t.from}__${t.trigger}__${target}`,
          source: t.from, target,
          type: 'transitionEdge', label: t.trigger,
          class: isTimeout ? 'timeout-edge' : '',
          data: { guard: t.guard ?? '', isTimeout },
        }))
      }
      const target = t.to || t.target || ''
      return [{ id: `${t.from}__${t.trigger}__${target}`, source: t.from, target, type: 'transitionEdge', label: t.trigger, class: isTimeout ? 'timeout-edge' : '', data: { guard: t.guard ?? '', isTimeout } }]
    })

    agents.value  = def.agents ?? []
    isDirty.value = false

    // Clear the guard flag AFTER all mutations.
    //
    // Timing matters: the first reactive mutation above causes Vue to schedule
    // flushJobs via Promise.resolve().then(flushJobs).  Any nextTick() called
    // *after* that point chains onto currentFlushPromise, so it runs *after*
    // the watchers — meaning fromLoad is still true when watch(meta) fires and
    // isDirty stays false.
    //
    // If we called nextTick() *before* the mutations (as before), it would race
    // with flushJobs on the same Promise.resolve() tick and clear the flag first.
    nextTick(() => {
      fromLoad.value = false
      // Belt-and-suspenders: ensure isDirty is clean even if a watcher slipped through.
      isDirty.value  = false
    })
  }

  function buildDefinition() {
    const cleanAgents = agents.value
      .filter(a => a.name)
      .map(a => {
        const obj = { name: a.name }
        if (a.model)      obj.model      = a.model
        if (a.task_queue) obj.task_queue = a.task_queue
        const { prompt, mcp_servers: mcp, provider } = a.config ?? {}
        if (prompt || mcp || provider) {
          obj.config = {}
          if (prompt)   obj.config.prompt      = prompt
          if (mcp)      obj.config.mcp_servers = mcp
          if (provider) obj.config.provider    = provider
        }
        return obj
      })

    return {
      apiVersion: 'asm/v1',
      kind:       'Workflow',
      metadata: {
        name:         meta.name,
        version:      meta.version || 'draft',
        description:  meta.description,
        system_prompt: meta.systemPrompt,
      },
      blackboard: { schema: { ...bbSchema } },
      states: nodes.value.map(n => ({
        name:         n.id,
        type:         n.data.type         ?? 'intermediate',
        instructions: n.data.instructions || undefined,
        agent:        n.data.agent        || undefined,
        script:       n.data.script       || undefined,
        code:         n.data.code         || undefined,
        condition:    n.data.condition    || undefined,
        timeout:      n.data.timeout      || undefined,
        on_timeout:   n.data.on_timeout   || undefined,
        skill_call:   n.data.skill_call   || undefined,
        position:     n.position,
      })),
      transitions: Array.from(
        edges.value.reduce((map, e) => {
          const key = `${e.source}__${e.label}`
          if (!map.has(key)) map.set(key, { from: e.source, trigger: e.label, guard: e.data?.guard || undefined, to_nodes: [] })
          map.get(key).to_nodes.push(e.target)
          return map
        }, new Map()).values()
      ).map(t => t.to_nodes.length === 1
        ? { from: t.from, to: t.to_nodes[0], trigger: t.trigger, guard: t.guard }
        : { from: t.from, to_nodes: t.to_nodes, trigger: t.trigger, guard: t.guard }
      ),
      agents: cleanAgents,
    }
  }

  async function saveWorkflow() {
    if (!meta.name) { alert('Workflow name is required'); return }

    const unassigned = nodes.value.filter(n =>
      (n.data.type === 'initial' || n.data.type === 'intermediate') && !n.data.agent
    )
    if (unassigned.length > 0) {
      const names = unassigned.map(n => n.id).join(', ')
      if (!confirm(`Warning: The following reasoning states have no agent assigned: ${names}.\n\nThese states will wait for external triggers and will NOT be automated by an LLM. Continue?`)) return
    }

    saving.value = true
    try {
      const def    = buildDefinition()
      const yamlStr = yaml.dump(def, { lineWidth: 120 })
      const result  = await wfStore.save(yamlStr)
      const newDef  = result.definition ?? result
      meta.version       = newDef.metadata?.version ?? meta.version
      meta.versionNumber = result.version_number ?? newDef.metadata?.version_number ?? meta.versionNumber
      isDirty.value = false
      try { versions.value = await wfStore.fetchVersions(meta.name) } catch { /* ignore */ }
      router.replace(`/designer/${meta.name}/${meta.version}`)
      return result
    } catch (e) {
      alert(e.message)
      throw e
    } finally {
      saving.value = false
    }
  }

  async function switchVersion(vnum) {
    if (isDirty.value && !confirm('You have unsaved changes. Discard and switch version?')) return
    try {
      const { definition, yaml: src } = await wfStore.fetchByVersion(meta.name, vnum)
      loadDefinition(definition, src)
      await nextTick()
      flowKey.value++
      router.replace(`/designer/${meta.name}/${meta.version}`)
    } catch (e) {
      alert(e.message)
    }
  }

  async function onRunClick() {
    if (isDirty.value) {
      if (confirm('You have unsaved changes. Save them before running?')) {
        try { await saveWorkflow() } catch { return }
      }
    }
    showRunModal.value = true
  }

  function onAIApply({ yamlStr, definition }) {
    try {
      const def = definition || yaml.load(yamlStr)
      if (!def) return
      loadDefinition(def, yamlStr)
      nextTick(() => { flowKey.value++ })
      isDirty.value = true
    } catch (e) {
      alert('Failed to apply AI-generated workflow: ' + e.message)
    }
  }

  function importYAML() {
    yamlError.value = ''
    try {
      const def = yaml.load(yamlSource.value)
      if (!def) throw new Error('YAML is empty or invalid.')
      loadDefinition(def, null, false)
      view.value = 'canvas'
      nextTick(() => { flowKey.value++ })
      isDirty.value = true
    } catch (e) {
      yamlError.value = e.message
      alert('Failed to import YAML: ' + e.message)
    }
  }

  return { fromLoad, loadDefinition, buildDefinition, saveWorkflow, switchVersion, onRunClick, onAIApply, importYAML }
}
