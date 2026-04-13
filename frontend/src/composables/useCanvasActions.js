import { nextTick } from 'vue'

export function useCanvasActions({ nodes, edges, agents, isDirty, selected, pendingConnection, showAddState, bbSchema, fitView, layout }) {

  // ── Layout ──────────────────────────────────────────────────────────────────
  function applyLayout(markDirty = true) {
    if (!nodes.value.length) return
    nodes.value = layout(nodes.value, edges.value)
    nextTick(() => fitView({ padding: 0.15, duration: 300 }))
    if (markDirty) isDirty.value = true
  }

  // ── Canvas events ────────────────────────────────────────────────────────────
  function onNodeClick({ node }) {
    pendingConnection.value = null
    const ownNode = nodes.value.find(n => n.id === node.id)
    selected.value = { type: 'node', id: node.id, data: { ...(ownNode?.data ?? node.data) } }
  }

  function onEdgeClick({ edge }) {
    pendingConnection.value = null
    selected.value = {
      type: 'edge', id: edge.id,
      data: { label: edge.label, guard: edge.data?.guard ?? '', from: edge.source, to: edge.target },
    }
  }

  function onPaneClick() {
    selected.value = null
  }

  function onNodesChange(changes) {
    if (changes.some(c => c.type !== 'select' && c.type !== 'dimensions')) {
      isDirty.value = true
    }
  }

  function onConnect(params) {
    pendingConnection.value = { source: params.source, target: params.target }
    selected.value = null
  }

  function confirmConnection({ trigger, guard }) {
    const { source, target } = pendingConnection.value
    edges.value.push({
      id: `${source}__${trigger}__${target}`,
      source, target,
      type: 'transitionEdge', label: trigger,
      data: { guard: guard || '', isTimeout: false },
    })
    pendingConnection.value = null
    isDirty.value = true
  }

  function cancelConnection() {
    pendingConnection.value = null
  }

  // ── Agent helpers ────────────────────────────────────────────────────────────
  function getAgent(name) {
    if (!name) return null
    return agents.value.find(a => a.name === name) ?? null
  }

  function upsertAgent(def) {
    if (!def?.name) return
    const idx = agents.value.findIndex(a => a.name === def.name)
    if (idx !== -1) agents.value[idx] = def
    else agents.value.push(def)
  }

  // ── Node / edge mutations ────────────────────────────────────────────────────
  function updateNode({ node, agentDef }) {
    const currentSelected = selected.value
    if (!currentSelected || currentSelected.type !== 'node') return

    const idx   = nodes.value.findIndex(n => n.id === currentSelected.id)
    if (idx === -1) return

    const newId = node.name
    const oldId = nodes.value[idx].id

    if (newId !== oldId && nodes.value.some(n => n.id === newId)) {
      alert(`A state named "${newId}" already exists.`)
      return
    }

    if (newId === oldId) {
      nodes.value[idx] = { ...nodes.value[idx], data: { ...node } }
      isDirty.value = true
    } else {
      nodes.value[idx] = { ...nodes.value[idx], id: newId, data: { ...node }, selected: true }
      edges.value = edges.value.map(e => {
        if (e.source === oldId || e.target === oldId) {
          return {
            ...e,
            id:     `${e.source === oldId ? newId : e.source}__${e.label}__${e.target === oldId ? newId : e.target}`,
            source: e.source === oldId ? newId : e.source,
            target: e.target === oldId ? newId : e.target,
          }
        }
        return e
      })
    }

    upsertAgent(agentDef)
    nextTick(() => {
      selected.value = { type: 'node', id: newId, data: { ...node } }
      isDirty.value  = true
    })
  }

  function updateEdge(data) {
    const currentSelected = selected.value
    if (!currentSelected || currentSelected.type !== 'edge') return

    const idx = edges.value.findIndex(e => e.id === currentSelected.id)
    if (idx === -1) return

    const newId = `${edges.value[idx].source}__${data.label}__${edges.value[idx].target}`
    edges.value[idx] = {
      ...edges.value[idx],
      id: newId, label: data.label,
      data: { ...edges.value[idx].data, guard: data.guard },
      selected: true,
    }
    nextTick(() => {
      selected.value = {
        type: 'edge', id: newId,
        data: { label: data.label, guard: data.guard, from: edges.value[idx].source, to: edges.value[idx].target },
      }
      isDirty.value = true
    })
  }

  function deleteSelected() {
    if (!selected.value) return
    if (selected.value.type === 'node') {
      const id = selected.value.id
      nodes.value = nodes.value.filter(n => n.id !== id)
      edges.value = edges.value.filter(e => e.source !== id && e.target !== id)
    } else {
      edges.value = edges.value.filter(e => e.id !== selected.value.id)
    }
    selected.value = null
    isDirty.value  = true
  }

  // ── Add state ────────────────────────────────────────────────────────────────
  function addState() { showAddState.value = true }

  function onAddState(stateData) {
    const existingX = nodes.value.map(n => n.position.x)
    const x = existingX.length ? Math.max(...existingX) + 200 : 200
    nodes.value.push({ id: stateData.name, type: 'stateNode', position: { x, y: 200 }, data: { ...stateData } })
    showAddState.value = false
    isDirty.value = true
  }

  // ── Blackboard ───────────────────────────────────────────────────────────────
  function addBBField({ name, type, required }) {
    bbSchema[name] = { type, required }
    isDirty.value = true
  }

  function removeBBField(key) {
    delete bbSchema[key]
    isDirty.value = true
  }

  return {
    applyLayout,
    onNodeClick, onEdgeClick, onPaneClick, onNodesChange,
    onConnect, confirmConnection, cancelConnection,
    getAgent,
    updateNode, updateEdge, deleteSelected,
    addState, onAddState,
    addBBField, removeBBField,
  }
}
