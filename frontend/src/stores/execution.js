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

  async function fetchRuns(filter = {}) {
    const params = new URLSearchParams()
    if (filter.workflow) params.set('workflow', filter.workflow)
    if (filter.status)   params.set('status', filter.status)
    const { data } = await api.get(`/runs?${params}`)
    runs.value = data ?? []
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

  async function fetchPendingHITL() {
    const { data } = await api.get('/hitl/pending')
    pendingHITL.value = data ?? []
  }

  async function sendTrigger(runId, trigger, payload = {}) {
    await api.post(`/runs/${runId}/trigger`, { trigger, payload })
  }

  async function resolveHITL(runId, resolution, resolver = 'user') {
    await api.post(`/runs/${runId}/signal`, { resolution, resolver })
    await fetchPendingHITL()
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
      case 'run.failed': {
        const idx = runs.value.findIndex(r => r.id === data.run.id)
        if (idx !== -1) runs.value[idx] = data.run
        if (activeRun.value?.id === data.run.id) activeRun.value = data.run
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

      case 'hitl.waiting':
        pendingHITL.value.push({ run_id: data.run_id, state_name: data.state_name })
        break
    }
  }

  return {
    runs, activeRun, transitions, eventLog, thinkingTokens, pendingHITL,
    fetchRuns, fetchRun, fetchHistory, fetchPendingHITL,
    sendTrigger, resolveHITL, handleEvent
  }
})
