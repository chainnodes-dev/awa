<template>
  <div class="p-4 space-y-4 flex-1 overflow-y-auto">

    <!-- Header -->
    <div class="flex items-center gap-2">
      <svg class="text-violet-400 shrink-0" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
      </svg>
      <span class="text-xs font-semibold text-slate-300 flex-1">Skill</span>
    </div>

    <p class="text-[10px] text-slate-600 leading-relaxed">
      Describe your skill in plain English. Click <strong class="text-slate-400">Analyse</strong> to
      decompose it into an optimised ASM workflow — LLM nodes only where reasoning is needed,
      code/script nodes everywhere else.
    </p>

    <!-- ── Description editor ────────────────────────────────────────── -->
    <div class="space-y-1">
      <div class="flex items-center gap-1.5">
        <label class="text-xs text-slate-500 flex-1">Skill Description</label>
        <button
          class="text-slate-500 hover:text-violet-400 transition-colors"
          title="Open in fullscreen editor"
          @click="showDescModal = true"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/>
            <line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/>
          </svg>
        </button>
      </div>
      <CodeEditor
        v-model="description"
        language="text"
        height="200px"
      />
    </div>

    <!-- ── I/O Contract ───────────────────────────────────────────────── -->
    <div class="space-y-2.5 pt-2 border-t border-border">
      <h4 class="text-[10px] font-semibold text-slate-500 uppercase tracking-wider">
        Input / Output Contract
        <span class="text-slate-700 font-normal normal-case">(optional)</span>
      </h4>

      <div class="space-y-1">
        <div class="text-[10px] text-slate-600">Inputs — data required at invocation</div>
        <div v-for="(p, i) in inputs" :key="'in'+i" class="flex gap-1.5 items-center">
          <input class="input font-mono text-[10px] py-0.5 w-24 shrink-0" v-model="p.name" placeholder="field" />
          <select class="input text-[10px] py-0.5 w-20 shrink-0" v-model="p.type">
            <option>string</option><option>number</option><option>bool</option><option>object</option>
          </select>
          <input class="input text-[10px] py-0.5 flex-1" v-model="p.description" placeholder="description" />
          <button class="text-slate-600 hover:text-red-400 text-xs px-1 transition-colors" @click="inputs.splice(i,1)">✕</button>
        </div>
        <button class="w-full text-[10px] py-1 border border-dashed border-border hover:border-violet-500/40 text-slate-600 hover:text-slate-400 rounded transition-colors" @click="inputs.push({name:'',type:'string',description:''})">+ Add input</button>
      </div>

      <div class="space-y-1">
        <div class="text-[10px] text-slate-600">Outputs — data produced on completion</div>
        <div v-for="(p, i) in outputs" :key="'out'+i" class="flex gap-1.5 items-center">
          <input class="input font-mono text-[10px] py-0.5 w-24 shrink-0" v-model="p.name" placeholder="field" />
          <select class="input text-[10px] py-0.5 w-20 shrink-0" v-model="p.type">
            <option>string</option><option>number</option><option>bool</option><option>object</option>
          </select>
          <input class="input text-[10px] py-0.5 flex-1" v-model="p.description" placeholder="description" />
          <button class="text-slate-600 hover:text-red-400 text-xs px-1 transition-colors" @click="outputs.splice(i,1)">✕</button>
        </div>
        <button class="w-full text-[10px] py-1 border border-dashed border-border hover:border-violet-500/40 text-slate-600 hover:text-slate-400 rounded transition-colors" @click="outputs.push({name:'',type:'string',description:''})">+ Add output</button>
      </div>
    </div>

    <!-- ── Analyse button ─────────────────────────────────────────────── -->
    <button
      :class="['w-full flex items-center justify-center gap-1.5 text-xs py-2 px-3 rounded font-medium transition-colors',
        analyseLoading
          ? 'bg-violet-500/10 text-violet-400/50 cursor-not-allowed'
          : 'bg-violet-600 hover:bg-violet-500 text-white']"
      :disabled="analyseLoading || !description.trim()"
      @click="analyse"
    >
      <svg v-if="!analyseLoading" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
      </svg>
      <svg v-else class="animate-spin" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
      </svg>
      {{ analyseLoading ? 'Analysing skill…' : 'Analyse → Generate Workflow' }}
    </button>
    <p v-if="analyseError" class="text-[10px] text-red-400 font-mono">{{ analyseError }}</p>

    <!-- ── Divider ────────────────────────────────────────────────────── -->
    <div class="flex items-center gap-2 pt-1">
      <div class="flex-1 h-px bg-border" />
      <span class="text-[9px] text-slate-700 uppercase tracking-wider">or</span>
      <div class="flex-1 h-px bg-border" />
    </div>

    <!-- ── Render button ──────────────────────────────────────────────── -->
    <div class="space-y-1">
      <button
        :class="['w-full flex items-center justify-center gap-1.5 text-xs py-1.5 px-3 rounded font-medium transition-colors border',
          renderLoading
            ? 'bg-teal-500/5 text-teal-400/40 border-teal-500/10 cursor-not-allowed'
            : 'bg-surface-0 hover:bg-teal-500/10 text-teal-400 border-teal-500/30 hover:border-teal-500/60']"
        :disabled="renderLoading || !yamlSource?.trim()"
        @click="render"
      >
        <svg v-if="!renderLoading" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
        </svg>
        <svg v-else class="animate-spin" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
        </svg>
        {{ renderLoading ? 'Rendering…' : 'Render Description from Current Workflow' }}
      </button>
      <p v-if="renderError" class="text-[10px] text-red-400 font-mono">{{ renderError }}</p>
    </div>

    <!-- ── Rendered output ────────────────────────────────────────────── -->
    <template v-if="rendered">
      <div class="pt-2 border-t border-border space-y-3">
        <h4 class="text-[10px] font-semibold text-teal-400/70 uppercase tracking-wider">Rendered Skill Description</h4>

        <div class="text-[11px] text-slate-300 bg-surface-0 border border-border rounded px-3 py-2.5 leading-relaxed">
          {{ rendered.description }}
        </div>

        <template v-if="rendered.inferred_inputs?.length">
          <div class="space-y-1">
            <div class="text-[9px] text-slate-600 uppercase tracking-wider font-semibold">Inferred Inputs</div>
            <div v-for="p in rendered.inferred_inputs" :key="p.name"
                 class="flex gap-2 text-[10px] px-2 py-1 rounded bg-surface-0 border border-border">
              <span class="font-mono text-slate-300 w-24 shrink-0 truncate">{{ p.name }}</span>
              <span class="text-slate-600 w-14 shrink-0">{{ p.type }}</span>
              <span class="text-slate-500 flex-1">{{ p.description }}</span>
            </div>
          </div>
        </template>

        <template v-if="rendered.inferred_outputs?.length">
          <div class="space-y-1">
            <div class="text-[9px] text-slate-600 uppercase tracking-wider font-semibold">Inferred Outputs</div>
            <div v-for="p in rendered.inferred_outputs" :key="p.name"
                 class="flex gap-2 text-[10px] px-2 py-1 rounded bg-surface-0 border border-border">
              <span class="font-mono text-slate-300 w-24 shrink-0 truncate">{{ p.name }}</span>
              <span class="text-slate-600 w-14 shrink-0">{{ p.type }}</span>
              <span class="text-slate-500 flex-1">{{ p.description }}</span>
            </div>
          </div>
        </template>

        <button class="btn-ghost text-[10px] w-full" @click="copyRenderedToDescription">
          ↑ Copy description to editor
        </button>
      </div>
    </template>

  </div>

  <!-- Fullscreen description editor -->
  <CodeEditorModal
    v-if="showDescModal"
    v-model="description"
    language="text"
    title="Skill Description"
    @close="showDescModal = false"
  />
</template>

<script setup>
import { ref, reactive } from 'vue'
import { api } from '@/stores/auth'
import CodeEditor from './CodeEditor.vue'
import CodeEditorModal from './CodeEditorModal.vue'

const props = defineProps({
  yamlSource: { type: String, default: '' },
})
const emit = defineEmits(['apply'])

// ── State ────────────────────────────────────────────────────────────────────
const description    = ref('')
const inputs         = reactive([])
const outputs        = reactive([])
const showDescModal  = ref(false)

const analyseLoading = ref(false)
const analyseError   = ref('')
const renderLoading  = ref(false)
const renderError    = ref('')
const rendered       = ref(null)

// ── Analyse: description → workflow ──────────────────────────────────────────
async function analyse() {
  if (!description.value.trim()) return
  analyseLoading.value = true
  analyseError.value   = ''
  try {
    const { data } = await api.post('/designer/skill/analyse', {
      description:   description.value,
      inputs:        inputs.filter(p => p.name),
      outputs:       outputs.filter(p => p.name),
      existing_yaml: props.yamlSource || '',
    })
    emit('apply', { yamlStr: data.yaml, definition: data.definition })
  } catch (e) {
    analyseError.value = e.response?.data?.error ?? e.message
  } finally {
    analyseLoading.value = false
  }
}

// ── Render: current workflow → description ────────────────────────────────────
async function render() {
  if (!props.yamlSource?.trim()) return
  renderLoading.value = true
  renderError.value   = ''
  rendered.value      = null
  try {
    const { data } = await api.post('/designer/skill/render', {
      workflow_yaml: props.yamlSource,
    })
    rendered.value = data
  } catch (e) {
    renderError.value = e.response?.data?.error ?? e.message
  } finally {
    renderLoading.value = false
  }
}

function copyRenderedToDescription() {
  if (rendered.value?.description) {
    description.value = rendered.value.description
    if (rendered.value.inferred_inputs?.length)
      inputs.splice(0, inputs.length, ...rendered.value.inferred_inputs.map(p => ({ ...p })))
    if (rendered.value.inferred_outputs?.length)
      outputs.splice(0, outputs.length, ...rendered.value.inferred_outputs.map(p => ({ ...p })))
  }
}
</script>
