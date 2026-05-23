<template>
  <div class="flex flex-col flex-1 min-h-0">
    <!-- Messages -->
    <div class="flex-1 overflow-y-auto p-3 space-y-3" ref="messagesEl">
      <!-- Welcome -->
      <div v-if="!messages.length" class="text-center py-8 space-y-3">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#8b5cf6" stroke-width="1.5" class="mx-auto">
          <path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z"/>
        </svg>
        <p class="text-sm text-text-muted">AI Workflow Assistant</p>
        <p class="text-xs text-text-muted leading-relaxed max-w-[220px] mx-auto">
          {{ yamlSource ? 'Describe changes to make to this workflow. The AI will modify the existing definition.' : 'Describe a business process and the AI will generate a workflow for you.' }}
        </p>
      </div>

      <!-- Message list -->
      <div v-for="(msg, i) in messages" :key="i" class="space-y-1.5">
        <div class="flex items-center gap-1.5">
          <span v-if="msg.role === 'user'" class="text-[10px] font-medium text-text-muted uppercase">You</span>
          <span v-else class="text-[10px] font-medium text-violet-500 uppercase">AI</span>
        </div>
        <div :class="msg.role === 'user'
          ? 'text-xs text-text bg-surface-0 rounded-lg px-3 py-2 border border-border'
          : 'text-xs text-text-muted bg-violet-500/5 rounded-lg px-3 py-2 border border-violet-500/20'">
          {{ msg.content }}
        </div>
        <button v-if="msg.yaml" class="btn-primary text-xs gap-1.5 w-full" @click="applyResult(msg)">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          Apply to Workflow
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center gap-2 text-xs text-violet-400 animate-pulse py-2">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="animate-spin">
          <circle cx="12" cy="12" r="10" opacity="0.25"/>
          <path d="M12 2a10 10 0 0 1 10 10" opacity="0.75"/>
        </svg>
        Thinking...
      </div>
    </div>

    <!-- Input -->
    <div class="shrink-0 border-t border-border p-3">
      <div class="flex gap-2">
        <textarea
          class="input flex-1 text-xs resize-none min-h-[36px] max-h-[100px]"
          v-model="input"
          :placeholder="yamlSource ? 'Describe a change...' : 'Describe a workflow...'"
          rows="1"
          @input="autoResize"
          @keydown.enter.exact.prevent="send"
          :disabled="loading"
        />
        <button class="btn-primary p-2 shrink-0 self-end" @click="send" :disabled="!input.trim() || loading" title="Send (Enter)">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>
          </svg>
        </button>
      </div>
      <p class="text-[10px] text-text-muted mt-1.5">Press Enter to send, Shift+Enter for new line</p>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { api } from '@/stores/auth'

const props = defineProps({
  yamlSource:          { type: String, default: '' },
  workflowName:        { type: String, default: '' },
  workflowDescription: { type: String, default: '' },
})
const emit = defineEmits(['apply'])

const messages   = ref([])
const input      = ref('')
const loading    = ref(false)
const error      = ref('')
const messagesEl = ref(null)

watch(messages, async () => {
  await nextTick()
  if (messagesEl.value) messagesEl.value.scrollTop = messagesEl.value.scrollHeight
}, { deep: true })

function autoResize(e) {
  e.target.style.height = 'auto'
  e.target.style.height = Math.min(e.target.scrollHeight, 100) + 'px'
}

async function send() {
  const text = input.value.trim()
  if (!text || loading.value) return
  messages.value.push({ role: 'user', content: text })
  input.value = ''
  loading.value = true
  error.value = ''
  try {
    const payload = { description: text }
    if (props.yamlSource?.trim()) payload.existing_yaml = props.yamlSource
    if (props.workflowDescription) payload.workflow_description = props.workflowDescription
    const history = messages.value.slice(0, -1).flatMap(m => {
      if (m.role === 'assistant') {
        if (!m.yaml) return []
        return [{ role: 'assistant', content: '```yaml\n' + m.yaml + '\n```' }]
      }
      return [{ role: m.role, content: m.content }]
    })
    if (history.length) payload.history = history
    const { data } = await api.post('/designer/generate', payload)
    if (data.error) {
      error.value = data.error
      messages.value.push({ role: 'assistant', content: 'Error: ' + data.error })
    } else {
      messages.value.push({
        role: 'assistant',
        content: props.yamlSource ? 'I\'ve updated the workflow based on your request.' : 'I\'ve generated a workflow based on your description.',
        yaml: data.yaml,
        definition: data.definition,
      })
    }
  } catch (e) {
    const msg = e.response?.data?.error ?? e.message
    error.value = msg
    messages.value.push({ role: 'assistant', content: 'Error: ' + msg })
  } finally {
    loading.value = false
  }
}

function applyResult(msg) {
  emit('apply', { yamlStr: msg.yaml, definition: msg.definition })
}
</script>
