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
    const targetNode = nodes.value.find(n => n.id === params.target)
    if (targetNode && (targetNode.data?.type === 'timeout_node' || targetNode.data?.is_timeout_node)) {
      const source = params.source
      const target = params.target
      
      const sourceNode = nodes.value.find(n => n.id === source)
      if (sourceNode && sourceNode.data.type !== 'initial' && sourceNode.data.type !== 'terminal' && sourceNode.data.type !== 'timeout_node') {
        sourceNode.data.timeout = targetNode.data.default_timeout || '10s'
        sourceNode.data.on_timeout = 'timeout'
      }

      const exists = edges.value.some(e => e.source === source && e.target === target)
      if (!exists) {
        edges.value.push({
          id: `${source}__timeout__${target}`,
          source,
          target,
          sourceHandle: 'timeout',
          type: 'transitionEdge',
          label: 'timeout',
          class: 'timeout-edge',
          data: { guard: '', isTimeout: true }
        })
      }
      pendingConnection.value = null
      isDirty.value = true
      return
    }

    pendingConnection.value = { source: params.source, target: params.target, sourceHandle: params.sourceHandle }
    selected.value = null
  }

  function confirmConnection({ trigger, guard }) {
    const { source, target, sourceHandle } = pendingConnection.value
    const targetNode = nodes.value.find(n => n.id === target)
    const isTimeout = trigger === 'timeout' || (targetNode && (targetNode.data?.type === 'timeout_node' || targetNode.data?.is_timeout_node))
    const finalTrigger = isTimeout ? 'timeout' : trigger

    if (isTimeout) {
      const sourceNode = nodes.value.find(n => n.id === source)
      if (sourceNode && sourceNode.data.type !== 'initial' && sourceNode.data.type !== 'terminal' && sourceNode.data.type !== 'timeout_node') {
        sourceNode.data.timeout = (targetNode && targetNode.data?.default_timeout) || '10s'
        sourceNode.data.on_timeout = 'timeout'
      }
    }

    edges.value.push({
      id: `${source}__${finalTrigger}__${target}`,
      source, target,
      sourceHandle: isTimeout ? 'timeout' : undefined,
      type: 'transitionEdge', label: finalTrigger,
      class: isTimeout ? 'timeout-edge' : '',
      data: { guard: guard || '', isTimeout },
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
      nodes.value = [...nodes.value]
      isDirty.value = true
    } else {
      const oldNode = nodes.value[idx]
      const updatedNode = { ...oldNode, id: newId, data: { ...node }, selected: true }
      
      nodes.value = [
        ...nodes.value.filter(n => n.id !== oldId),
        updatedNode
      ]

      edges.value = edges.value.map(e => {
        if (e.source === oldId || e.target === oldId) {
          const newSource = e.source === oldId ? newId : e.source
          const newTarget = e.target === oldId ? newId : e.target
          return {
            ...e,
            id:     `${newSource}__${e.label}__${newTarget}`,
            source: newSource,
            target: newTarget,
          }
        }
        return e
      })
    }

    // Cascading default timeout distribution
    if (node.type === 'timeout_node' || node.is_timeout_node) {
      const timeoutVal = node.default_timeout || node.defaultTimeout || '10s'
      nodes.value = nodes.value.map(n => {
        if (
          n.id !== newId &&
          n.data.type !== 'initial' &&
          n.data.type !== 'terminal' &&
          n.data.type !== 'timeout_node'
        ) {
          return {
            ...n,
            data: {
              ...n.data,
              timeout: timeoutVal,
              on_timeout: 'timeout'
            }
          }
        }
        return n
      })

      // Add/update timeout edges
      const eligibleNodes = nodes.value.filter(n =>
        n.id !== newId &&
        n.data.type !== 'initial' &&
        n.data.type !== 'terminal' &&
        n.data.type !== 'timeout_node'
      )

      eligibleNodes.forEach(en => {
        const existingIdx = edges.value.findIndex(e => e.source === en.id && e.label === 'timeout')
        if (existingIdx !== -1) {
          edges.value[existingIdx] = {
            ...edges.value[existingIdx],
            id: `${en.id}__timeout__${newId}`,
            target: newId,
            sourceHandle: 'timeout',
            class: 'timeout-edge',
            data: { ...edges.value[existingIdx].data, isTimeout: true }
          }
        } else {
          edges.value.push({
            id: `${en.id}__timeout__${newId}`,
            source: en.id,
            target: newId,
            sourceHandle: 'timeout',
            type: 'transitionEdge',
            label: 'timeout',
            class: 'timeout-edge',
            data: { guard: '', isTimeout: true }
          })
        }
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
    const isTimeout = data.label === 'timeout'
    edges.value[idx] = {
      ...edges.value[idx],
      id: newId, label: data.label,
      sourceHandle: isTimeout ? 'timeout' : undefined,
      data: { ...edges.value[idx].data, guard: data.guard, isTimeout },
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
      const deletedNode = nodes.value.find(n => n.id === id)
      nodes.value = nodes.value.filter(n => n.id !== id)
      edges.value = edges.value.filter(e => e.source !== id && e.target !== id)

      // If Timeout Node is deleted, clear other node timeouts and transitions
      if (deletedNode && (deletedNode.data.type === 'timeout_node' || deletedNode.data.is_timeout_node)) {
        nodes.value = nodes.value.map(n => {
          if (n.data.timeout || n.data.on_timeout) {
            return {
              ...n,
              data: {
                ...n.data,
                timeout: undefined,
                on_timeout: undefined,
              }
            }
          }
          return n
        })
        edges.value = edges.value.filter(e => e.label !== 'timeout')
      }
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

    // Auto-apply default timeout if Timeout Node exists
    const timeoutNode = nodes.value.find(n => n.data.type === 'timeout_node')
    if (timeoutNode && stateData.type !== 'initial' && stateData.type !== 'terminal' && stateData.type !== 'timeout_node') {
      stateData.timeout = timeoutNode.data.default_timeout || '10s'
      stateData.on_timeout = 'timeout'
      
      edges.value.push({
        id: `${stateData.name}__timeout__${timeoutNode.id}`,
        source: stateData.name,
        target: timeoutNode.id,
        sourceHandle: 'timeout',
        type: 'transitionEdge',
        label: 'timeout',
        class: 'timeout-edge',
        data: { guard: '', isTimeout: true }
      })
    }

    const newNodeObj = { id: stateData.name, type: 'stateNode', position: { x, y: 200 }, data: { ...stateData } }
    nodes.value.push(newNodeObj)
    selected.value = { type: 'node', id: stateData.name, data: newNodeObj.data }
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
