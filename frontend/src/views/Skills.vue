<template>
  <div class="flex flex-col h-full overflow-hidden">

    <!-- Top bar -->
    <header class="flex items-center gap-4 px-6 py-4 border-b border-border shrink-0">
      <svg class="text-violet-400 shrink-0" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
        <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
      </svg>
      <div>
        <h1 class="text-base font-semibold text-slate-200">Skills</h1>
        <p class="text-xs text-slate-500">Reusable capabilities with typed I/O contracts backed by workflows</p>
      </div>
      <div class="flex-1"/>
      <!-- Search -->
      <div class="relative">
        <svg class="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-600" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input
          v-model="search"
          class="input pl-7 pr-3 py-1.5 text-xs w-52"
          placeholder="Search skills…"
        />
      </div>
      <button v-if="canWrite" @click="openModal(null)" class="btn-primary text-xs gap-1.5">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        New Skill
      </button>
    </header>

    <!-- Body -->
    <div class="flex-1 overflow-y-auto p-6">

      <div v-if="loading" class="text-slate-600 text-sm py-12 text-center">Loading…</div>

      <!-- Empty state -->
      <div v-else-if="!filtered.length" class="card p-12 text-center">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" class="mx-auto mb-4 text-slate-700">
          <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
        </svg>
        <p class="text-slate-500 text-sm mb-1">{{ search ? 'No skills match your search' : 'No skills registered yet' }}</p>
        <p class="text-slate-600 text-xs mb-4">{{ search ? 'Try a different search term.' : 'Create your first skill to make it available as a reusable building block.' }}</p>
        <button v-if="canWrite && !search" @click="openModal(null)" class="btn-primary text-xs">New Skill</button>
      </div>

      <!-- Skills table -->
      <div v-else class="card overflow-hidden">
        <table class="w-full text-xs">
          <thead>
            <tr class="border-b border-border text-slate-500 font-medium">
              <th class="text-left px-4 py-3 w-48">Name</th>
              <th class="text-left px-4 py-3">Description</th>
              <th class="text-center px-3 py-3 w-16">Inputs</th>
              <th class="text-center px-3 py-3 w-16">Outputs</th>
              <th class="text-center px-3 py-3 w-20">Used by</th>
              <th class="text-left px-4 py-3 w-44">Workflow</th>
              <th class="px-4 py-3 w-28"/>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="sk in filtered"
              :key="sk.name"
              class="border-b border-border/50 hover:bg-white/2 transition-colors group"
            >
              <!-- Name -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <svg class="text-violet-400/60 shrink-0" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z"/>
                  </svg>
                  <span class="font-mono text-slate-200 font-medium">{{ sk.name }}</span>
                </div>
                <!-- Capability badges -->
                <div v-if="sk.capabilities?.length" class="flex flex-wrap gap-1 mt-1 pl-5">
                  <span v-for="cap in sk.capabilities" :key="cap.mcp_server"
                        class="text-[9px] font-mono px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                    {{ cap.mcp_server }}
                  </span>
                </div>
              </td>

              <!-- Description -->
              <td class="px-4 py-3 text-slate-400 leading-relaxed max-w-xs">
                <span class="line-clamp-2">{{ sk.description || '—' }}</span>
              </td>

              <!-- Inputs count -->
              <td class="px-3 py-3 text-center">
                <span v-if="sk.inputs?.length"
                      class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-indigo-500/10 text-indigo-400 font-medium">
                  {{ sk.inputs.length }}
                </span>
                <span v-else class="text-slate-700">—</span>
              </td>

              <!-- Outputs count -->
              <td class="px-3 py-3 text-center">
                <span v-if="sk.outputs?.length"
                      class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-teal-500/10 text-teal-400 font-medium">
                  {{ sk.outputs.length }}
                </span>
                <span v-else class="text-slate-700">—</span>
              </td>

              <!-- Dependents count -->
              <td class="px-3 py-3 text-center">
                <span v-if="sk.dependents_count > 0"
                      class="inline-flex items-center justify-center w-6 h-6 rounded-full bg-violet-500/10 text-violet-400 font-medium"
                      :title="`Used by ${sk.dependents_count} skill${sk.dependents_count === 1 ? '' : 's'}`">
                  {{ sk.dependents_count }}
                </span>
                <span v-else class="text-slate-700">—</span>
              </td>

              <!-- Workflow ref -->
              <td class="px-4 py-3">
                <RouterLink
                  v-if="sk.workflow_ref"
                  :to="`/designer/${sk.workflow_ref}/latest`"
                  class="font-mono text-indigo-400 hover:text-indigo-300 transition-colors flex items-center gap-1 group/link"
                >
                  <span class="truncate max-w-[140px]">{{ sk.workflow_ref }}</span>
                  <svg class="opacity-0 group-hover/link:opacity-100 shrink-0 transition-opacity" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                    <polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/>
                  </svg>
                </RouterLink>
                <span v-else class="text-slate-600 italic text-[10px]">not linked</span>
              </td>

              <!-- Actions -->
              <td class="px-4 py-3">
                <div class="flex items-center gap-1 justify-end opacity-0 group-hover:opacity-100 transition-opacity">
                  <button v-if="canWrite" @click="openModal(sk)" class="btn-ghost text-xs py-1 px-2">Edit</button>
                  <button v-if="canWrite" @click="confirmDelete(sk)" class="btn-danger text-xs py-1 px-2">Delete</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

    </div>

    <!-- Create / Edit Modal -->
    <SkillModal
      v-if="modalSkill !== undefined"
      :skill="modalSkill"
      @close="modalSkill = undefined"
      @saved="onSaved"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useAuthStore, api } from '@/stores/auth'
import SkillModal from '@/components/designer/SkillModal.vue'

const authStore = useAuthStore()
const canWrite  = computed(() => authStore.hasRole('admin') || authStore.hasRole('operator'))

const skills     = ref([])
const loading    = ref(true)
const search     = ref('')
// undefined = closed | null = create | object = edit
const modalSkill = ref(undefined)

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  if (!q) return skills.value
  return skills.value.filter(s =>
    s.name.toLowerCase().includes(q) ||
    s.description?.toLowerCase().includes(q) ||
    s.workflow_ref?.toLowerCase().includes(q)
  )
})

async function fetchSkills() {
  loading.value = true
  try {
    const { data } = await api.get('/skills')
    skills.value = data ?? []
  } catch { /* non-fatal */ } finally {
    loading.value = false
  }
}

function openModal(sk) {
  modalSkill.value = sk  // null → create, object → edit
}

async function confirmDelete(sk) {
  const warning = sk.dependents_count > 0
    ? `\n\n⚠ This skill is used by ${sk.dependents_count} other skill${sk.dependents_count === 1 ? '' : 's'}. Deleting it will break those callers.`
    : ''
  if (!confirm(`Delete skill "${sk.name}"? This cannot be undone.${warning}`)) return
  try {
    await api.delete(`/skills/${sk.name}?force=true`)
    skills.value = skills.value.filter(s => s.name !== sk.name)
  } catch (e) {
    alert(e.response?.data?.error ?? e.message)
  }
}

function onSaved(sk) {
  modalSkill.value = undefined
  const idx = skills.value.findIndex(s => s.name === sk.name)
  if (idx !== -1) skills.value[idx] = sk
  else skills.value.push(sk)
  // Re-sort by name
  skills.value.sort((a, b) => a.name.localeCompare(b.name))
}

onMounted(fetchSkills)
</script>
