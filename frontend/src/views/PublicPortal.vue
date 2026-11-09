<template>
  <div class="min-h-screen bg-slate-950 text-slate-100 flex flex-col justify-between selection:bg-indigo-500/30">
    <!-- Header -->
    <header class="border-b border-slate-800/80 bg-slate-900/50 backdrop-blur px-6 py-4 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center font-bold text-white shadow-lg shadow-indigo-500/20">
          ASM
        </div>
        <div>
          <h1 class="text-sm font-semibold tracking-wide text-slate-100">ASM Secure Portal</h1>
          <p class="text-[11px] text-slate-400">Guest Workflow Submission</p>
        </div>
      </div>
      <div v-if="portalData?.expires_at && portalData?.status === 'active'" class="text-xs text-slate-400 flex items-center gap-1.5 bg-slate-800/60 px-2.5 py-1 rounded-full border border-slate-700/50">
        <svg class="w-3.5 h-3.5 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <circle cx="12" cy="12" r="10" stroke-width="2"/>
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6l4 2"/>
        </svg>
        <span>Expires: {{ formatDate(portalData.expires_at) }}</span>
      </div>
    </header>

    <!-- Main Container -->
    <main class="flex-1 flex items-center justify-center p-4 sm:p-6 md:p-8">
      <div class="w-full max-w-2xl bg-slate-900/80 border border-slate-800 rounded-2xl shadow-2xl p-6 sm:p-8 relative overflow-hidden backdrop-blur-xl">
        <!-- Background Gradient Glow -->
        <div class="absolute -top-24 -right-24 w-72 h-72 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none"></div>
        <div class="absolute -bottom-24 -left-24 w-72 h-72 bg-blue-500/10 rounded-full blur-3xl pointer-events-none"></div>

        <!-- 1. Loading State -->
        <div v-if="loading" class="py-16 flex flex-col items-center justify-center gap-4 text-center">
          <svg class="animate-spin h-8 w-8 text-indigo-400" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <p class="text-sm text-slate-400">Loading secure guest form...</p>
        </div>

        <!-- 2. Error / Link Not Found -->
        <div v-else-if="fetchError" class="py-12 text-center space-y-4">
          <div class="w-14 h-14 rounded-full bg-red-500/10 border border-red-500/30 text-red-400 flex items-center justify-center mx-auto shadow-lg shadow-red-500/10">
            <svg class="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
            </svg>
          </div>
          <h2 class="text-xl font-bold text-white">Invalid or Expired Link</h2>
          <p class="text-sm text-slate-400 max-w-md mx-auto">{{ fetchError }}</p>
        </div>

        <!-- 3. Form Already Completed -->
        <div v-else-if="portalData?.status === 'completed' || submitSuccess" class="py-12 text-center space-y-5">
          <div class="w-16 h-16 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 flex items-center justify-center mx-auto shadow-lg shadow-emerald-500/20">
            <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
            </svg>
          </div>
          <div class="space-y-2">
            <h2 class="text-2xl font-bold text-white">Submission Completed</h2>
            <p class="text-sm text-slate-300 max-w-md mx-auto">
              {{ submitSuccess ? 'Thank you! Your information has been securely received and the process has resumed.' : (portalData?.message || 'This form has already been submitted.') }}
            </p>
            <p v-if="portalData?.completed_at" class="text-xs text-slate-500">
              Submitted on {{ formatDate(portalData.completed_at) }}
            </p>
          </div>
          <div class="pt-4 border-t border-slate-800 text-xs text-slate-400">
            You may safely close this window.
          </div>
        </div>

        <!-- 4. Link Expired -->
        <div v-else-if="portalData?.status === 'expired'" class="py-12 text-center space-y-4">
          <div class="w-14 h-14 rounded-full bg-amber-500/10 border border-amber-500/30 text-amber-400 flex items-center justify-center mx-auto">
            <svg class="w-7 h-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <circle cx="12" cy="12" r="10" stroke-width="2"/>
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6l4 2"/>
            </svg>
          </div>
          <h2 class="text-xl font-bold text-white">Link Has Expired</h2>
          <p class="text-sm text-slate-400 max-w-md mx-auto">
            This guest invitation link is no longer valid. Please contact the sender to request a new link.
          </p>
        </div>

        <!-- 5. Active Form Submission -->
        <div v-else-if="portalData?.status === 'active'" class="space-y-6">
          <div class="space-y-1.5 border-b border-slate-800 pb-4">
            <h2 class="text-xl font-bold text-white">{{ portalData.title || 'Required Information' }}</h2>
            <p v-if="portalData.description" class="text-sm text-slate-400 leading-relaxed">{{ portalData.description }}</p>
          </div>

          <form @submit.prevent="handleSubmit" class="space-y-6">
            <!-- Dynamic Fields from JSON Schema -->
            <div class="space-y-5">
              <div v-for="(fieldSchema, fieldKey) in properties" :key="fieldKey" class="space-y-1.5">
                <label class="block text-xs font-semibold text-slate-300">
                  {{ fieldSchema.title || formatFieldName(fieldKey) }}
                  <span v-if="isRequired(fieldKey)" class="text-rose-400 ml-0.5">*</span>
                </label>
                <p v-if="fieldSchema.description" class="text-[11px] text-slate-400">{{ fieldSchema.description }}</p>

                <!-- 1. Textarea -->
                <textarea
                  v-if="fieldSchema.type === 'string' && (fieldSchema.format === 'textarea' || fieldSchema.multiline)"
                  v-model="formData[fieldKey]"
                  rows="3"
                  class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-all"
                  :placeholder="fieldSchema.placeholder || ''"
                  :required="isRequired(fieldKey)"
                ></textarea>

                <!-- 2. Select / Enum -->
                <select
                  v-else-if="fieldSchema.enum"
                  v-model="formData[fieldKey]"
                  class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-all"
                  :required="isRequired(fieldKey)"
                >
                  <option value="" disabled selected>(Select an option...)</option>
                  <option v-for="opt in fieldSchema.enum" :key="opt" :value="opt">{{ opt }}</option>
                </select>

                <!-- 3. Boolean Checkbox -->
                <div v-else-if="fieldSchema.type === 'boolean'" class="flex items-center gap-3 py-1">
                  <input
                    type="checkbox"
                    v-model="formData[fieldKey]"
                    :id="fieldKey"
                    class="w-4 h-4 rounded border-slate-700 bg-slate-800 text-indigo-600 focus:ring-indigo-500/30"
                  />
                  <label :for="fieldKey" class="text-sm text-slate-300 cursor-pointer">
                    {{ fieldSchema.checkboxLabel || 'Confirm / Agree' }}
                  </label>
                </div>

                <!-- 4. File Upload -->
                <div v-else-if="fieldSchema.type === 'file'" class="space-y-2">
                  <div v-if="formData[fieldKey]?.name" class="flex items-center justify-between p-3 rounded-xl bg-slate-800/80 border border-slate-700">
                    <div class="flex items-center gap-2.5 min-w-0">
                      <svg class="w-5 h-5 text-indigo-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                      </svg>
                      <div class="min-w-0">
                        <div class="text-xs font-medium text-white truncate">{{ formData[fieldKey].name }}</div>
                        <div class="text-[10px] text-slate-400">{{ formatSize(formData[fieldKey].size) }}</div>
                      </div>
                    </div>
                    <button
                      type="button"
                      @click="removeFile(fieldKey)"
                      class="p-1 text-slate-400 hover:text-rose-400 transition-colors"
                      title="Remove file"
                    >
                      <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                      </svg>
                    </button>
                  </div>

                  <div v-else class="relative">
                    <input
                      type="file"
                      :id="'file-' + fieldKey"
                      class="hidden"
                      @change="(e) => handleFileUpload(fieldKey, e)"
                    />
                    <label
                      :for="'file-' + fieldKey"
                      class="flex flex-col items-center justify-center border-2 border-dashed border-slate-700 hover:border-indigo-500/60 rounded-xl p-5 cursor-pointer bg-slate-800/30 hover:bg-slate-800/50 transition-all text-center"
                    >
                      <svg v-if="!uploading[fieldKey]" class="w-7 h-7 text-indigo-400 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
                      </svg>
                      <svg v-else class="animate-spin h-7 w-7 text-indigo-400 mb-2" viewBox="0 0 24 24" fill="none">
                        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      <span class="text-xs font-medium text-slate-200">
                        {{ uploading[fieldKey] ? 'Uploading file...' : 'Click or drop document to upload' }}
                      </span>
                      <span class="text-[10px] text-slate-400 mt-1">PDF, DOCX, PNG, JPG up to 25MB</span>
                    </label>
                  </div>
                  <p v-if="uploadErrors[fieldKey]" class="text-[11px] text-rose-400">{{ uploadErrors[fieldKey] }}</p>
                </div>

                <!-- 5. Number / Integer -->
                <input
                  v-else-if="fieldSchema.type === 'number' || fieldSchema.type === 'integer'"
                  type="number"
                  v-model.number="formData[fieldKey]"
                  class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-all"
                  :placeholder="fieldSchema.placeholder || ''"
                  :required="isRequired(fieldKey)"
                />

                <!-- 6. Default String / Input -->
                <input
                  v-else
                  type="text"
                  v-model="formData[fieldKey]"
                  class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-all"
                  :placeholder="fieldSchema.placeholder || ''"
                  :required="isRequired(fieldKey)"
                />
              </div>
            </div>

            <!-- Submitter Name / Email (Optional identification) -->
            <div class="pt-4 border-t border-slate-800 space-y-1.5">
              <label class="block text-xs font-semibold text-slate-300">
                Your Name or Email <span class="text-slate-500 font-normal">(Optional)</span>
              </label>
              <input
                type="text"
                v-model="submitter"
                placeholder="e.g. Alex Smith"
                class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white placeholder-slate-500 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-all"
              />
            </div>

            <!-- Error Banner -->
            <div v-if="submitError" class="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-400 text-xs flex items-center gap-2">
              <svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
              </svg>
              <span>{{ submitError }}</span>
            </div>

            <!-- Submit Button -->
            <button
              type="submit"
              :disabled="submitting || isAnyUploading"
              class="w-full py-3.5 px-6 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-sm transition-all shadow-lg shadow-indigo-600/30 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              <svg v-if="submitting" class="animate-spin h-4 w-4 text-white" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <span>{{ submitting ? 'Submitting Form...' : 'Submit Information' }}</span>
            </button>
          </form>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="py-4 text-center text-xs text-slate-500 border-t border-slate-900">
      Protected by ASM Secure Guest Link. Tokens are single-use and encrypted at rest.
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'

const route = useRoute()
const runID = computed(() => route.params.run_id)
const token = computed(() => route.params.token)

const loading = ref(true)
const fetchError = ref('')
const portalData = ref(null)
const submitter = ref('')

const formData = reactive({})
const uploading = reactive({})
const uploadErrors = reactive({})
const submitting = ref(false)
const submitSuccess = ref(false)
const submitError = ref('')

const properties = computed(() => portalData.value?.form_schema?.properties || {})

const isAnyUploading = computed(() => {
  return Object.values(uploading).some(Boolean)
})

function isRequired(key) {
  const req = portalData.value?.form_schema?.required
  return Array.isArray(req) && req.includes(key)
}

function formatFieldName(key) {
  return key
    .replace(/([A-Z])/g, ' $1')
    .replace(/[_-]/g, ' ')
    .replace(/^./, (s) => s.toUpperCase())
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  try {
    return new Date(dateStr).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return dateStr
  }
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

onMounted(async () => {
  if (!runID.value || !token.value) {
    fetchError.value = 'Incomplete portal URL.'
    loading.value = false
    return
  }

  try {
    const res = await axios.get(`/api/v1/portal/${runID.value}/${token.value}`)
    portalData.value = res.data

    // Initialize defaults from schema
    if (portalData.value.form_schema?.properties) {
      for (const [key, prop] of Object.entries(portalData.value.form_schema.properties)) {
        if (prop.default !== undefined) {
          formData[key] = prop.default
        } else if (prop.type === 'boolean') {
          formData[key] = false
        }
      }
    }
  } catch (err) {
    fetchError.value = err.response?.data?.error || err.message || 'Failed to load form.'
  } finally {
    loading.value = false
  }
})

async function handleFileUpload(fieldKey, event) {
  const file = event.target.files?.[0]
  if (!file) return

  uploading[fieldKey] = true
  uploadErrors[fieldKey] = ''

  try {
    const data = new FormData()
    data.append('file', file)

    const res = await axios.post(
      `/api/v1/portal/${runID.value}/${token.value}/upload`,
      data,
      { headers: { 'Content-Type': 'multipart/form-data' } }
    )

    formData[fieldKey] = res.data
  } catch (err) {
    uploadErrors[fieldKey] = err.response?.data?.error || err.message || 'File upload failed.'
  } finally {
    uploading[fieldKey] = false
    event.target.value = ''
  }
}

function removeFile(fieldKey) {
  formData[fieldKey] = null
}

async function handleSubmit() {
  submitError.value = ''
  submitting.value = true

  try {
    const payload = {
      submitter: submitter.value,
      data: { ...formData }
    }

    const res = await axios.post(
      `/api/v1/portal/${runID.value}/${token.value}/submit`,
      payload
    )

    if (res.data?.status === 'completed') {
      submitSuccess.value = true
    }
  } catch (err) {
    submitError.value = err.response?.data?.error || err.message || 'Submission failed.'
  } finally {
    submitting.value = false
  }
}
</script>

