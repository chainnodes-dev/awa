import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useExecutionStore } from '@/stores/execution'
import { useAuthStore } from '@/stores/auth'

let ws = null
let reconnectTimer = null
const connected = ref(false)

export function useWebSocket() {
  function connect() {
    if (ws && ws.readyState === WebSocket.OPEN) return

    const authStore = useAuthStore()
    const token = sessionStorage.getItem('access_token')
    if (!token) return  // not authenticated — don't connect

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    ws = new WebSocket(`${protocol}//${location.host}/ws?token=${encodeURIComponent(token)}`)

    const execStore = useExecutionStore()

    ws.onopen = () => {
      connected.value = true
      console.log('[ws] connected')
      if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
    }

    ws.onmessage = ({ data }) => {
      try {
        const event = JSON.parse(data)
        execStore.handleEvent(event)
      } catch (e) {
        console.warn('[ws] bad message', e)
      }
    }

    ws.onclose = (ev) => {
      connected.value = false
      // Code 1008 (Policy Violation) = auth rejected — don't retry, send to login.
      if (ev.code === 1008) {
        console.warn('[ws] auth rejected — redirecting to login')
        authStore.logout()
        useRouter().push('/login')
        return
      }
      console.log('[ws] disconnected — reconnecting in 3s')
      reconnectTimer = setTimeout(connect, 3000)
    }

    ws.onerror = () => ws.close()
  }

  function disconnect() {
    if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
    if (ws) { ws.close(); ws = null }
    connected.value = false
  }

  return { connected, connect, disconnect }
}
