import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { api } from '@/stores/auth'

export const useExecutionStore = defineStore('execution', () => {
  const runs = ref([])
  const activeRun = ref(null)
  const transitions = ref([])      // history for activeRun
  const eventLog = ref([])         // all received events (capped at 200)
  const thinkingTokens = reactive({}) // runId → accumulated token string
  const pendingHITL = ref([])
  // runId → array of { id, stateName, agentName, method, toolName, input, output, duration, timestamp, isError }
  const llmCalls = reactive({})
  const mcpLogs = reactive({})
  const chatMessages = reactive({}) // runId → array of { sender, message, role, timestamp }

  async function fetchRuns(filter = {}) {
    const params = new URLSearchParams()
    if (filter.workflow)     params.set('workflow', filter.workflow)
    if (filter.status)       params.set('status', filter.status)
    if (filter.state)        params.set('state', filter.state)
    if (filter.started_from) params.set('started_from', filter.started_from)
    if (filter.started_to)   params.set('started_to', filter.started_to)
    const { data } = await api.get(`/runs?${params}`)
    runs.value = data ?? []
  }

  async function deleteRun(id) {
    await api.delete(`/runs/${id}`)
    runs.value = runs.value.filter(r => r.id !== id)
    pendingHITL.value = pendingHITL.value.filter(h => h.run_id !== id)
  }

  async function fetchRun(id) {
    const { data } = await api.get(`/runs/${id}`)
    activeRun.value = data
    return data
  }

  async function fetchHistory(id) {
    const { data } = await api.get(`/runs/${id}/history`)
    transitions.value = data ?? []
    return data
  }

  async function fetchMCPLogs(id) {
    const { data } = await api.get(`/runs/${id}/mcp-logs`)
    mcpLogs[id] = data ?? []
    return data
  }

  async function fetchPendingHITL(filter = {}) {
    const params = new URLSearchParams()
    if (filter.assignee) params.set('assignee', filter.assignee)
    const { data } = await api.get(`/hitl/pending?${params}`)
    pendingHITL.value = data ?? []
  }

  async function sendTrigger(runId, trigger, payload = {}) {
    await api.post(`/runs/${runId}/trigger`, { trigger, payload })
  }

  async function resolveHITL(runId, resolution, resolver = 'user', payload = {}) {
    await api.post(`/runs/${runId}/signal`, { resolution, resolver, payload })
    await fetchPendingHITL()
  }

  async function sendChat(runId, message) {
    await api.post(`/runs/${runId}/chat`, { message })
    // The agent.chat event will come back via WS and update the local state.
  }

  async function terminateRun(id) {
    await api.post(`/runs/${id}/terminate`)
    // Optionally local update, but WebSocket event will normally handle it
    const idx = runs.value.findIndex(r => r.id === id)
    if (idx !== -1) runs.value[idx].status = 'cancelled'
    if (activeRun.value?.id === id) activeRun.value.status = 'cancelled'
  }

  // Called by useWebSocket when a new event arrives
  function handleEvent(event) {
    // Append to global event log (cap at 200)
    eventLog.value.unshift(event)
    if (eventLog.value.length > 200) eventLog.value.length = 200

    const { type, data } = event

    switch (type) {
      case 'run.created':
        runs.value.unshift(data.run)
        break

      case 'run.completed':
      case 'run.failed':
      case 'run.cancelled': {
        // Payload shape differs between direct mode {run: {...}} and
        // Temporal mode {run_id, error}. Normalise to a run-id + error string.
        const runId  = data.run?.id ?? data.run_id
        const errMsg = data.run?.failure_reason ?? data.error ?? null
        const idx = runs.value.findIndex(r => r.id === runId)
        if (data.run) {
          if (idx !== -1) runs.value[idx] = data.run
          if (activeRun.value?.id === runId) activeRun.value = data.run
        } else {
          // Temporal mode: patch status + failure_reason in place
          if (idx !== -1) {
            runs.value[idx].status = type === 'run.failed' ? 'failed' : 
                                     type === 'run.cancelled' ? 'cancelled' : 'complete'
            if (errMsg) runs.value[idx].failure_reason = errMsg
          }
          if (activeRun.value?.id === runId) {
            activeRun.value = { 
              ...activeRun.value, 
              status: type === 'run.failed' ? 'failed' : 
                      type === 'run.cancelled' ? 'cancelled' : 'complete', 
              failure_reason: errMsg 
            }
          }
        }
        break
      }

      case 'state.changed': {
        // Update run in list
        const idx = runs.value.findIndex(r => r.id === data.run_id)
        if (idx !== -1) {
          runs.value[idx].current_state = data.to_state
          runs.value[idx].blackboard = data.blackboard
        }
        // Update active run detail
        if (activeRun.value?.id === data.run_id) {
          activeRun.value.current_state = data.to_state
          activeRun.value.blackboard = data.blackboard
        }
        // Append to transitions
        transitions.value.push({
          from_state: data.from_state,
          to_state: data.to_state,
          trigger: data.trigger,
          timestamp: event.timestamp,
          blackboard_snapshot: data.blackboard
        })
        break
      }

      case 'blackboard.updated': {
        if (activeRun.value?.id === data.run_id) {
          activeRun.value.blackboard = {
            ...activeRun.value.blackboard,
            [data.key]: data.value
          }
        }
        break
      }

      case 'agent.thinking': {
        if (!thinkingTokens[data.run_id]) thinkingTokens[data.run_id] = ''
        thinkingTokens[data.run_id] += data.token
        break
      }

      case 'agent.prompt': {
        console.debug('[debug] agent.prompt received', data.run_id, data.agent_name, data.state_name)
        if (!llmCalls[data.run_id]) llmCalls[data.run_id] = []
        llmCalls[data.run_id].push({
          id:        `${data.state_name}:${data.agent_name}:${Date.now()}`,
          stateName: data.state_name,
          agentName: data.agent_name,
          system:    data.system,
          messages:  data.messages,
          response:  null,
          timestamp: event.timestamp,
        })
        break
      }

      case 'agent.response': {
        console.debug('[debug] agent.response received', data.run_id, data.agent_name, data.trigger)
        const calls = llmCalls[data.run_id]
        if (calls) {
          // Find the most recent pending call for this state+agent
          for (let i = calls.length - 1; i >= 0; i--) {
            if (calls[i].stateName === data.state_name &&
                calls[i].agentName === data.agent_name &&
                calls[i].response === null) {
              calls[i].response = {
                content:   data.content,
                trigger:   data.trigger,
                reasoning: data.reasoning,
              }
              break
            }
          }
        }
        break
      }

      case 'hitl.waiting': {
        pendingHITL.value.push({ run_id: data.run_id, state_name: data.state_name, assignee: data.assignee })
        // Update run status so HITL controls appear immediately without a manual refresh
        const hwIdx = runs.value.findIndex(r => r.id === data.run_id)
        if (hwIdx !== -1) runs.value[hwIdx].status = 'waiting'
        if (activeRun.value?.id === data.run_id)
          activeRun.value = { ...activeRun.value, status: 'waiting' }
        break
      }

      case 'hitl.resolved': {
        // Remove from pending list and restore run status to 'running' so HITL
        // controls disappear immediately while the workflow transitions.
        pendingHITL.value = pendingHITL.value.filter(h => h.run_id !== data.run_id)
        const hrIdx = runs.value.findIndex(r => r.id === data.run_id)
        if (hrIdx !== -1 && runs.value[hrIdx].status === 'waiting')
          runs.value[hrIdx].status = 'running'
        if (activeRun.value?.id === data.run_id && activeRun.value.status === 'waiting')
          activeRun.value = { ...activeRun.value, status: 'running' }
        break
      }

      case 'agent.chat': {
        if (!chatMessages[data.run_id]) chatMessages[data.run_id] = []
        chatMessages[data.run_id].push({
          sender:    data.sender,
          message:   data.message,
          role:      data.role, // 'human' | 'agent'
          timestamp: event.timestamp,
        })
        break
      }
    }
  }

  return {
    runs, activeRun, transitions, eventLog, thinkingTokens, pendingHITL, llmCalls, mcpLogs, chatMessages,
    fetchRuns, fetchRun, fetchHistory, fetchPendingHITL, fetchMCPLogs,
    deleteRun, terminateRun, sendTrigger, resolveHITL, sendChat, handleEvent
  }
})
