<template>
  <div class="flex flex-col h-full overflow-y-auto bg-surface-0">
    <header class="px-6 py-4 border-b border-border shrink-0 bg-surface-1 flex items-center justify-between">
      <div>
        <h1 class="text-base font-semibold text-text">System Settings</h1>
        <p class="text-xs text-text-muted mt-0.5">Manage platform-wide security and infrastructure configuration</p>
      </div>
      <div class="flex items-center gap-2">
        <span class="badge bg-purple-500/20 text-purple-400">Super Admin</span>
      </div>
    </header>

    <div class="flex-1 px-6 py-6 max-w-3xl w-full mx-auto space-y-8">
      <!-- Licensing Keys -->
      <section class="space-y-4">
        <div class="flex items-center gap-2 px-1">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" class="text-indigo-400">
            <path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/>
          </svg>
          <h2 class="text-sm font-bold text-text uppercase tracking-wider">Platform Licensing Keys</h2>
        </div>

        <div class="card p-6 space-y-6">
          <div class="space-y-4">
            <p class="text-xs text-text-muted">
              These keys are used to sign and verify enterprise licenses. 
              <span class="text-amber-400 font-medium">Rotating these keys will invalidate all existing license tokens.</span>
            </p>

            <div>
              <label class="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1.5">Public Key (Verification)</label>
              <textarea 
                v-model="keys.public_key" 
                readonly 
                rows="6" 
                class="input w-full font-mono text-[10px] bg-black/20 border-border/40 focus:border-border/40 cursor-default"
              />
              <p class="text-[10px] text-text-muted mt-1">This key is used by workers to verify licenses locally.</p>
            </div>

            <div>
              <label class="text-[10px] font-bold text-text-muted uppercase tracking-widest block mb-1.5">Private Key (Signing)</label>
              <textarea 
                v-model="keys.private_key" 
                :readonly="!isEditing"
                rows="10" 
                :class="[
                  'input w-full font-mono text-[10px] transition-all',
                  isEditing ? 'bg-black/10 border-accent' : 'bg-black/20 border-border/40 cursor-default'
                ]"
                placeholder="-----BEGIN RSA PRIVATE KEY-----"
              />
              <p class="text-[10px] text-text-muted mt-1">This key must remain secret. It is used to issue new licenses.</p>
            </div>
          </div>

          <div class="flex items-center justify-between pt-4 border-t border-border/40">
            <div class="flex gap-3">
              <button 
                v-if="!isEditing"
                @click="startEditing" 
                class="btn-ghost text-xs px-4"
              >
                Edit Keys
              </button>
              <button 
                v-else
                @click="cancelEditing" 
                class="btn-ghost text-xs px-4"
              >
                Cancel
              </button>
              
              <button 
                v-if="!isEditing"
                @click="generateNewKeys" 
                :disabled="busy"
                class="btn-ghost text-violet-400 text-xs px-4 font-bold uppercase"
              >
                {{ busy ? 'Generating...' : 'Generate New Pair' }}
              </button>
            </div>

            <button 
              v-if="isEditing"
              @click="saveKeys" 
              :disabled="busy"
              class="btn-primary text-xs px-6"
            >
              {{ busy ? 'Saving...' : 'Save & Reload' }}
            </button>
          </div>
        </div>
      </section>

      <!-- Advanced Info -->
      <section class="p-4 rounded-lg bg-amber-500/5 border border-amber-500/20">
        <div class="flex gap-3">
          <svg class="text-amber-500 shrink-0 mt-0.5" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          <div class="space-y-1">
            <h4 class="text-xs font-bold text-amber-500 uppercase tracking-wider">Critical Infrastructure</h4>
            <p class="text-xs text-text-muted leading-relaxed">
              Modifying these keys directly impacts the platform's ability to enforce resource limits. Ensure you have a backup of the current keys before making any changes. If you rotate keys, you must re-issue licenses for all active tenants.
            </p>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { api } from '@/stores/auth'

const keys = reactive({
  public_key: '',
  private_key: ''
})

const isEditing = ref(false)
const busy = ref(false)
let originalKeys = { public_key: '', private_key: '' }

async function fetchKeys() {
  try {
    const { data } = await api.get('/system/keys')
    keys.public_key = data.public_key
    keys.private_key = data.private_key
    originalKeys = { ...data }
  } catch (e) {
    console.error('Failed to fetch keys', e)
  }
}

function startEditing() {
  isEditing.value = true
  // When editing, we might want to clear the masked private key 
  // so the user can paste a new one.
  if (keys.private_key.includes('**** MASKED ****')) {
    keys.private_key = ''
  }
}

function cancelEditing() {
  isEditing.value = false
  keys.public_key = originalKeys.public_key
  keys.private_key = originalKeys.private_key
}

async function generateNewKeys() {
  if (!confirm('This will generate a new RSA-2048 key pair. Existing licenses will be invalidated if you save these. Continue?')) {
    return
  }
  
  busy.value = true
  try {
    const { data } = await api.post('/system/keys/generate')
    keys.public_key = data.public_key
    keys.private_key = data.private_key
    isEditing.value = true
  } catch (e) {
    alert('Generation failed: ' + (e.response?.data?.error ?? e.message))
  } finally {
    busy.value = false
  }
}

async function saveKeys() {
  if (!keys.private_key || !keys.public_key) {
    alert('Both keys are required.')
    return
  }

  if (keys.private_key.includes('**** MASKED ****')) {
     alert('Please provide a full private key or use the existing one.')
     return
  }

  busy.value = true
  try {
    await api.post('/system/keys', {
      public_key: keys.public_key,
      private_key: keys.private_key
    })
    isEditing.value = false
    await fetchKeys()
    alert('System keys updated and reloaded successfully.')
  } catch (e) {
    alert('Update failed: ' + (e.response?.data?.error ?? e.message))
  } finally {
    busy.value = false
  }
}

onMounted(fetchKeys)
</script>
