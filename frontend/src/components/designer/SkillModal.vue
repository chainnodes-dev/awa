<template>
  <div class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="card w-[600px] max-h-[90vh] flex flex-col">

      <!-- Header -->
      <div class="flex items-center gap-3 px-6 py-4 border-b border-border shrink-0">
        <svg class="text-violet-400 shrink-0" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
        </svg>
        <h3 class="font-semibold text-slate-200 flex-1">{{ isEdit ? 'Edit Skill' : 'New Skill' }}</h3>
        <button class="btn-ghost p-1.5" @click="$emit('close')">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto px-6 py-5 space-y-5">

        <!-- Dependents warning -->
        <div v-if="isEdit && props.skill?.dependents_count > 0"
             class="flex items-start gap-2 px-3 py-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-400 text-xs">
          <svg class="shrink-0 mt-0.5" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
            <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          <span>
            Used by <strong>{{ props.skill.dependents_count }} skill{{ props.skill.dependents_count === 1 ? '' : 's' }}</strong>.
            Changes to ports or behavior may break callers.
          </span>
        </div>

        <!-- Name -->
        <div class="space-y-1.5">
          <label class="text-xs text-slate-500">Name <span class="text-red-400">*</span></label>
          <input
            class="input font-mono"
            v-model="form.name"
            placeholder="company-lookup"
            :disabled="isEdit"
          />
          <p v-if="!isEdit" class="text-[10px] text-slate-600">Unique identifier — used as the skill_ref in skill_call nodes.</p>
        </div>

        <!-- Description -->
        <div class="space-y-1.5">
          <label class="text-xs text-slate-500">Description</label>
          <textarea
            class="input text-xs leading-relaxed resize-none h-24"
            v-model="form.description"
            placeholder="Describe what this skill does in plain English…"
          />
        </div>

        <!-- Workflow Ref -->
        <div class="space-y-1.5">
          <label class="text-xs text-slate-500">Workflow Ref</label>
          <div class="flex gap-2">
            <input
              class="input font-mono flex-1"
              v-model="form.workflow_ref"
              placeholder="company-lookup-workflow"
            />
            <RouterLink
              v-if="form.workflow_ref"
              :to="`/designer/${form.workflow_ref}/latest`"
              class="btn-ghost text-xs px-3 border border-border shrink-0"
              title="Open workflow in Designer"
            >Open ↗</RouterLink>
          </div>
          <p class="text-[10px] text-slate-600">The workflow that implements this skill. Leave blank to set later.</p>
        </div>

        <!-- Inputs -->
        <div class="space-y-2 pt-1 border-t border-border">
          <div class="flex items-center gap-2">
            <h4 class="text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1">Inputs</h4>
            <button class="text-[10px] text-indigo-400 hover:text-indigo-300 transition-colors" @click="form.inputs.push({name:'',type:'string',description:''})">+ Add</button>
          </div>
          <p v-if="!form.inputs.length" class="text-[10px] text-slate-600 italic">No input ports defined.</p>
          <div v-for="(p, i) in form.inputs" :key="'in'+i" class="flex gap-1.5 items-center">
            <input class="input font-mono text-xs py-1 w-28 shrink-0" v-model="p.name" placeholder="field" />
            <select class="input text-xs py-1 w-20 shrink-0" v-model="p.type">
              <option>string</option><option>number</option><option>bool</option><option>object</option>
            </select>
            <input class="input text-xs py-1 flex-1" v-model="p.description" placeholder="description" />
            <label class="flex items-center gap-1 text-[10px] text-slate-500 shrink-0 cursor-pointer" title="Required">
              <input type="checkbox" v-model="p.required" class="accent-indigo-500" />
              req
            </label>
            <button class="text-slate-600 hover:text-red-400 text-xs px-1 transition-colors" @click="form.inputs.splice(i,1)">✕</button>
          </div>
        </div>

        <!-- Outputs -->
        <div class="space-y-2 pt-1 border-t border-border">
          <div class="flex items-center gap-2">
            <h4 class="text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1">Outputs</h4>
            <button class="text-[10px] text-indigo-400 hover:text-indigo-300 transition-colors" @click="form.outputs.push({name:'',type:'string',description:''})">+ Add</button>
          </div>
          <p v-if="!form.outputs.length" class="text-[10px] text-slate-600 italic">No output ports defined.</p>
          <div v-for="(p, i) in form.outputs" :key="'out'+i" class="flex gap-1.5 items-center">
            <input class="input font-mono text-xs py-1 w-28 shrink-0" v-model="p.name" placeholder="field" />
            <select class="input text-xs py-1 w-20 shrink-0" v-model="p.type">
              <option>string</option><option>number</option><option>bool</option><option>object</option>
            </select>
            <input class="input text-xs py-1 flex-1" v-model="p.description" placeholder="description" />
            <button class="text-slate-600 hover:text-red-400 text-xs px-1 transition-colors" @click="form.outputs.splice(i,1)">✕</button>
          </div>
        </div>

        <!-- Capabilities (MCP servers) -->
        <div class="space-y-2 pt-1 border-t border-border">
          <div class="flex items-center gap-2">
            <h4 class="text-xs font-semibold text-slate-400 uppercase tracking-wider flex-1">Required MCP Servers</h4>
            <button class="text-[10px] text-indigo-400 hover:text-indigo-300 transition-colors" @click="form.capabilities.push({mcp_server:''})">+ Add</button>
          </div>
          <p v-if="!form.capabilities.length" class="text-[10px] text-slate-600 italic">No capability dependencies declared.</p>
          <div v-for="(cap, i) in form.capabilities" :key="'cap'+i" class="flex gap-1.5 items-center">
            <select class="input text-xs py-1 flex-1 font-mono" v-model="cap.mcp_server">
              <option value="">— select server —</option>
              <option v-for="srv in availableServers" :key="srv.name" :value="srv.name">{{ srv.name }}</option>
            </select>
            <button class="text-slate-600 hover:text-red-400 text-xs px-1 transition-colors" @click="form.capabilities.splice(i,1)">✕</button>
          </div>
        </div>

      </div>

      <!-- Footer -->
      <div class="flex items-center gap-3 px-6 py-4 border-t border-border shrink-0">
        <p v-if="error" class="text-xs text-red-400 flex-1">{{ error }}</p>
        <div v-else class="flex-1" />
        <button class="btn-ghost" @click="$emit('close')" :disabled="saving">Cancel</button>
        <button class="btn-primary px-6" @click="save" :disabled="saving || !form.name.trim()">
          {{ saving ? 'Saving…' : (isEdit ? 'Save Changes' : 'Create Skill') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '@/stores/auth'

const props = defineProps({
  skill: { type: Object, default: null }, // null = create, object = edit
})
const emit = defineEmits(['close', 'saved'])

const isEdit = computed(() => !!props.skill)
const saving = ref(false)
const error  = ref('')
const availableServers = ref([])

const form = reactive({
  name:         props.skill?.name         ?? '',
  description:  props.skill?.description  ?? '',
  workflow_ref: props.skill?.workflow_ref  ?? '',
  inputs:       props.skill?.inputs?.map(p => ({ ...p })) ?? [],
  outputs:      props.skill?.outputs?.map(p => ({ ...p })) ?? [],
  capabilities: props.skill?.capabilities?.map(c => ({ ...c })) ?? [],
})

onMounted(async () => {
  try {
    const { data } = await api.get('/mcp-servers')
    availableServers.value = data ?? []
  } catch { /* non-fatal — user can type manually */ }
})

async function save() {
  if (!form.name.trim()) { error.value = 'Name is required'; return }
  saving.value = true
  error.value  = ''
  try {
    const payload = {
      name:         form.name.trim(),
      description:  form.description,
      workflow_ref: form.workflow_ref.trim(),
      inputs:       form.inputs.filter(p => p.name),
      outputs:      form.outputs.filter(p => p.name),
      capabilities: form.capabilities.filter(c => c.mcp_server),
    }
    const res = isEdit.value
      ? await api.put(`/skills/${form.name}`, payload)
      : await api.post('/skills', payload)
    emit('saved', res.data)
  } catch (e) {
    error.value = e.response?.data?.error ?? e.message
  } finally {
    saving.value = false
  }
}
</script>
