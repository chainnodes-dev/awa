<template>
  <div class="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center z-[100] p-4" @click.self="$emit('close')">
    <div class="card w-full max-w-2xl max-h-[92vh] flex flex-col shadow-2xl border border-border bg-surface-1 rounded-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-200">
      
      <!-- Modal Header -->
      <header class="flex items-center justify-between px-6 py-4 border-b border-border bg-surface-2/60 shrink-0">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-lg bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center text-indigo-500">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="2" width="20" height="8" rx="2"/>
              <rect x="2" y="14" width="20" height="8" rx="2"/>
              <circle cx="6" cy="6" r="1" fill="currentColor"/>
              <circle cx="6" cy="18" r="1" fill="currentColor"/>
            </svg>
          </div>
          <div>
            <h3 class="text-sm font-bold text-text uppercase tracking-tight">
              {{ isEdit ? `Configure Server: ${form.name}` : 'Register / Install MCP Server' }}
            </h3>
            <p class="text-[10px] text-text-muted">
              {{ isEdit ? 'Manage environment variables, command arguments, and tools' : 'Configure connection details and environment parameters' }}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <button 
            type="button" 
            class="text-[10px] font-bold text-indigo-400 border border-indigo-500/30 px-2.5 py-1 rounded-lg hover:bg-indigo-500/10 transition-colors flex items-center gap-1.5"
            @click="showImport = !showImport"
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
            Import JSON
          </button>
          <button 
            type="button" 
            class="text-text-muted hover:text-text p-1.5 rounded-lg hover:bg-surface-2 transition-colors" 
            @click="$emit('close')"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
          </button>
        </div>
      </header>

      <!-- Claude Desktop Import Drawer -->
      <div v-if="showImport" class="p-4 mx-6 mt-4 bg-surface-2 rounded-xl border border-indigo-500/30 space-y-3 shrink-0">
        <div class="flex items-center justify-between">
          <label class="text-[10px] uppercase font-bold text-indigo-400 tracking-widest flex items-center gap-1.5">
            <span>Paste Claude Desktop JSON Configuration</span>
          </label>
          <span class="text-[9px] text-text-muted font-mono">{ "mcpServers": { ... } }</span>
        </div>
        <textarea 
          v-model="importText"
          class="input font-mono text-[10px] h-28 leading-tight custom-scrollbar" 
          placeholder='{
  "mcpServers": {
    "my-server": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres"],
      "env": { "DATABASE_URL": "postgresql://..." }
    }
  }
}'
        />
        <div class="flex justify-end gap-2">
          <button type="button" class="btn-ghost text-xs px-3 py-1" @click="showImport = false">Cancel</button>
          <button type="button" class="btn-primary text-xs px-4 py-1" @click="handleImport">Apply Config</button>
        </div>
      </div>

      <!-- Main Form Body -->
      <form class="flex-1 overflow-y-auto p-6 space-y-5 custom-scrollbar" @submit.prevent="save">
        
        <!-- General Info -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest flex items-center justify-between">
              <span>Logical Name</span>
              <span class="text-[9px] text-indigo-400 font-mono">unique identifier</span>
            </label>
            <input 
              v-model="form.name" 
              type="text" 
              required
              class="input font-mono text-xs" 
              placeholder="e.g. postgres, sec-edgar, slack"
            />
          </div>

          <div class="space-y-1.5">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest">Transport Protocol</label>
            <select v-model="form.transport" class="input text-xs font-mono">
              <option value="stdio">Local Process (stdio)</option>
              <option value="sse">Remote HTTP / SSE (Server-Sent Events)</option>
            </select>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest">Description</label>
            <input 
              v-model="form.description" 
              type="text" 
              class="input text-xs" 
              placeholder="Short description of capabilities"
            />
          </div>

          <div class="space-y-1.5">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest">Documentation URL</label>
            <input 
              v-model="form.doc_url" 
              type="url" 
              class="input font-mono text-xs" 
              placeholder="https://..."
            />
          </div>
        </div>

        <!-- Transport Specific Fields -->
        <div v-if="form.transport === 'stdio'" class="space-y-4 p-4 rounded-xl bg-surface-2/60 border border-border">
          <div class="text-[10px] font-bold text-text uppercase tracking-wider flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-accent"></span>
            Process Execution Config
          </div>
          
          <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div class="space-y-1">
              <label class="text-[9px] uppercase font-bold text-text-muted tracking-widest">Command / Executable</label>
              <input 
                v-model="form.command" 
                type="text" 
                required
                class="input font-mono text-xs" 
                placeholder="e.g. npx, uvx, docker, python3"
              />
            </div>

            <div class="md:col-span-2 space-y-1">
              <label class="text-[9px] uppercase font-bold text-text-muted tracking-widest">CLI Arguments</label>
              <input 
                v-model="form.args" 
                type="text" 
                class="input font-mono text-xs" 
                placeholder="e.g. -y @modelcontextprotocol/server-postgres"
              />
            </div>
          </div>
        </div>

        <div v-else class="space-y-2 p-4 rounded-xl bg-surface-2/60 border border-border">
          <div class="text-[10px] font-bold text-text uppercase tracking-wider flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-accent"></span>
            Remote Endpoint Config
          </div>
          <div class="space-y-1">
            <label class="text-[9px] uppercase font-bold text-text-muted tracking-widest">SSE Endpoint URL</label>
            <input 
              v-model="form.url" 
              type="url" 
              required
              class="input font-mono text-xs" 
              placeholder="e.g. http://localhost:8091/sse"
            />
          </div>
        </div>

        <!-- Scopes -->
        <div class="space-y-1.5">
          <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest flex items-center justify-between">
            <span>Permissions / Scopes</span>
            <span class="text-[9px] text-text-muted">comma-separated</span>
          </label>
          <input 
            v-model="form.scopes" 
            type="text" 
            class="input font-mono text-xs" 
            placeholder="e.g. read-only, write, analytics"
          />
        </div>

        <!-- Environment Variables & Secrets -->
        <div class="space-y-3 p-4 rounded-xl bg-surface-2/40 border border-border">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest">Environment Variables & Secrets</label>
              <span class="text-[9px] px-1.5 py-0.2 rounded bg-indigo-500/10 text-indigo-400 font-mono">{{ envList.filter(e => e.key).length }} configured</span>
            </div>
            <button 
              type="button" 
              class="text-[10px] font-bold text-indigo-400 hover:text-indigo-300 flex items-center gap-1 transition-colors"
              @click="addEnvVar"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
              Add Variable
            </button>
          </div>

          <div v-if="envList.length === 0" class="text-center py-4 text-xs text-text-muted italic border border-dashed border-border rounded-lg">
            No environment variables configured. Click "+ Add Variable" to configure API keys or database URLs.
          </div>

          <div v-else class="space-y-2">
            <div v-for="(e, idx) in envList" :key="idx" class="flex items-center gap-2">
              <div class="w-1/3">
                <input 
                  v-model="e.key" 
                  type="text"
                  class="input text-xs font-mono" 
                  placeholder="KEY (e.g. API_KEY)" 
                  autocomplete="off"
                />
              </div>
              <div class="flex-1 relative">
                <input 
                  v-model="e.val" 
                  :type="showSecrets[idx] ? 'text' : 'password'"
                  class="input text-xs font-mono !pr-8" 
                  placeholder="VALUE / SECRET" 
                  autocomplete="off"
                />
                <button 
                  type="button"
                  class="absolute right-2.5 top-1/2 -translate-y-1/2 text-text-muted hover:text-text p-1 transition-colors"
                  :title="showSecrets[idx] ? 'Hide Value' : 'Show Value'"
                  @click="showSecrets[idx] = !showSecrets[idx]"
                >
                  <svg v-if="showSecrets[idx]" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
                  <svg v-else width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                </button>
              </div>
              <button 
                type="button" 
                class="p-2 text-text-muted hover:text-red-400 transition-colors" 
                title="Remove variable"
                @click="removeEnvVar(idx)"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
          </div>
          <p class="text-[9px] text-text-muted leading-tight">
            Environment variables are passed to the server process upon startup. Ideal for tokens, credentials, and connection strings.
          </p>
        </div>

        <!-- Discovered Tools -->
        <div v-if="discoveredTools.length > 0" class="space-y-2 p-4 rounded-xl bg-surface-2/40 border border-border">
          <div class="flex items-center justify-between">
            <label class="text-[10px] uppercase font-bold text-text-muted tracking-widest flex items-center gap-1.5">
              <span>Discovered Tools</span>
              <span class="px-1.5 py-0.2 rounded-full bg-green-500/10 text-green-500 text-[9px] font-mono font-bold">{{ discoveredTools.length }} available</span>
            </label>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-2 max-h-36 overflow-y-auto p-1 custom-scrollbar">
            <div v-for="t in discoveredTools" :key="t.name" class="p-2 rounded-lg bg-surface-1 border border-border/80 flex flex-col gap-0.5">
              <span class="text-[11px] font-bold text-text font-mono truncate">{{ t.name }}</span>
              <span class="text-[9px] text-text-muted line-clamp-1">{{ t.description || 'No description provided' }}</span>
            </div>
          </div>
        </div>

        <!-- Error Message Banner -->
        <div v-if="error" class="p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-xs font-mono flex items-start gap-2">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="shrink-0 mt-0.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
          <span class="break-all">{{ error }}</span>
        </div>

        <!-- Success Message Banner -->
        <div v-if="savedSuccess" class="p-3 rounded-xl bg-green-500/10 border border-green-500/20 text-green-400 text-xs font-mono flex items-center gap-2">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
          <span>Configuration saved successfully!</span>
        </div>
      </form>

      <!-- Modal Footer -->
      <footer class="flex items-center justify-between px-6 py-4 border-t border-border bg-surface-2/60 shrink-0">
        <div class="flex items-center gap-2">
          <button 
            type="button" 
            :disabled="discovering || saving" 
            class="btn-ghost text-xs px-4 h-9 border border-border flex items-center gap-1.5"
            @click="discover"
          >
            <svg v-if="discovering" class="animate-spin w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" stroke-opacity="0.25"/><path d="M12 2a10 10 0 0 1 10 10"/></svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            {{ discovering ? 'Testing...' : 'Discover Tools' }}
          </button>

          <button 
            v-if="isEdit" 
            type="button" 
            :disabled="saving"
            class="btn-ghost text-red-400 hover:text-red-300 hover:bg-red-500/10 text-xs px-4 h-9 border border-red-500/20 flex items-center gap-1.5 transition-colors"
            @click="remove"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
            Delete Server
          </button>
        </div>

        <div class="flex items-center gap-2">
          <button 
            type="button" 
            class="btn-ghost text-xs px-4 h-9" 
            @click="$emit('close')"
          >
            Cancel
          </button>
          <button 
            type="button" 
            :disabled="saving" 
            class="btn-primary text-xs px-6 h-9 font-bold uppercase tracking-wider flex items-center gap-1.5 shadow-lg shadow-indigo-500/20"
            @click="save"
          >
            <svg v-if="saving" class="animate-spin w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" stroke-opacity="0.25"/><path d="M12 2a10 10 0 0 1 10 10"/></svg>
            {{ saving ? 'Saving...' : (isEdit ? 'Save Changes' : 'Install Server') }}
          </button>
        </div>
      </footer>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { api } from '@/stores/auth'

const props = defineProps({
  server: { type: Object, default: null },
  servers: { type: Array, default: () => [] }
})

const emit = defineEmits(['close', 'saved', 'deleted'])

const isEdit = computed(() => !!props.server?.id)

const form = reactive({
  name: props.server?.name ?? '',
  transport: props.server?.transport ?? 'stdio',
  url: props.server?.url ?? '',
  command: props.server?.command ?? '',
  args: Array.isArray(props.server?.args) ? props.server.args.join(' ') : (props.server?.args ?? ''),
  description: props.server?.description ?? '',
  doc_url: props.server?.doc_url ?? '',
  scopes: Array.isArray(props.server?.scopes) ? props.server.scopes.join(', ') : (props.server?.scopes ?? '')
})

const envList = ref([])
const showSecrets = reactive({})
const discoveredTools = ref(props.server?.tools ?? [])
const saving = ref(false)
const savedSuccess = ref(false)
const discovering = ref(false)
const error = ref('')
const showImport = ref(false)
const importText = ref('')

onMounted(() => {
  if (props.server?.env_vars) {
    envList.value = Object.entries(props.server.env_vars).map(([key, val]) => ({ key, val: val ?? '' }))
  }
  if (envList.value.length === 0) {
    envList.value.push({ key: '', val: '' })
  }
})

function addEnvVar() {
  envList.value.push({ key: '', val: '' })
}

function removeEnvVar(idx) {
  envList.value.splice(idx, 1)
}

function getEnvMap() {
  const m = {}
  envList.value.forEach(e => {
    const k = e.key.trim()
    if (k) m[k] = e.val
  })
  return m
}

function handleImport() {
  try {
    let data = JSON.parse(importText.value)
    if (data.mcpServers) {
      const keys = Object.keys(data.mcpServers)
      if (keys.length === 0) throw new Error('No servers found in mcpServers JSON')
      const name = keys[0]
      const config = data.mcpServers[name]
      if (!form.name) form.name = name
      form.command = config.command || form.command
      form.args = Array.isArray(config.args) ? config.args.join(' ') : (config.args || form.args)
      if (config.env) {
        envList.value = Object.entries(config.env).map(([key, val]) => ({ key, val }))
      }
    } else {
      form.command = data.command || form.command
      form.args = Array.isArray(data.args) ? data.args.join(' ') : (data.args || form.args)
      if (data.env) {
        envList.value = Object.entries(data.env).map(([key, val]) => ({ key, val }))
      }
    }
    showImport.value = false
    importText.value = ''
    error.value = ''
  } catch (e) {
    error.value = 'Invalid JSON: ' + e.message
  }
}

async function discover() {
  discovering.value = true
  error.value = ''
  try {
    const parsedArgs = form.args.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(s => s.replace(/^"|"$/g, '')) || []
    const payload = {
      name: form.name,
      transport: form.transport,
      url: form.url,
      command: form.command,
      args: parsedArgs,
      env_vars: getEnvMap()
    }
    const { data } = await api.post('/mcp-servers/discover', payload)
    if (data.error) {
      error.value = data.error
    } else {
      if (data.description && !form.description) {
        form.description = data.description
      }
      discoveredTools.value = data.raw_tools || []
    }
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    discovering.value = false
  }
}

async function save() {
  if (!form.name.trim()) {
    error.value = 'Name is required'
    return
  }

  const normName = form.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-+|-+$)/g, '')
  const duplicate = props.servers.some(s => 
    s.id !== props.server?.id && 
    s.name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-+|-+$)/g, '') === normName
  )
  if (duplicate) {
    error.value = `An MCP server named "${form.name}" is already registered.`
    return
  }

  saving.value = true
  error.value = ''
  try {
    const parsedArgs = form.args.match(/(?:[^\s"]+|"[^"]*")+/g)?.map(s => s.replace(/^"|"$/g, '')) || []
    const parsedScopes = form.scopes ? form.scopes.split(',').map(s => s.trim()).filter(Boolean) : []
    const payload = {
      name: form.name.trim(),
      transport: form.transport,
      url: form.url.trim(),
      command: form.command.trim(),
      args: parsedArgs,
      description: form.description.trim(),
      doc_url: form.doc_url.trim(),
      env_vars: getEnvMap(),
      tools: discoveredTools.value,
      scopes: parsedScopes
    }

    let res
    if (isEdit.value) {
      res = await api.put('/mcp-servers/' + props.server.id, payload)
    } else {
      res = await api.post('/mcp-servers', payload)
    }

    savedSuccess.value = true
    emit('saved', res.data)
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!props.server?.id) return
  if (!confirm(`Are you sure you want to delete the "${form.name}" MCP server? Any workflows relying on it will no longer have access.`)) {
    return
  }
  saving.value = true
  error.value = ''
  try {
    await api.delete('/mcp-servers/' + props.server.id)
    emit('deleted', props.server.id)
    emit('close')
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 9999px;
}
</style>
