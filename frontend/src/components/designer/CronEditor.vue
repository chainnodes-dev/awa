<template>
  <div class="space-y-3 bg-surface-1 p-3 rounded-lg border border-border">
    <div class="flex flex-wrap gap-2">
      <button 
        v-for="m in modes" :key="m.id"
        @click.prevent="setMode(m.id)"
        :class="['px-3 py-1.5 text-xs rounded-md transition-colors', mode === m.id ? 'bg-accent text-white font-medium' : 'bg-surface-0 text-text-muted hover:bg-surface-2']"
      >
        {{ m.label }}
      </button>
    </div>

    <!-- Interval Mode -->
    <div v-if="mode === 'interval'" class="flex items-center gap-2">
      <span class="text-xs text-text-muted">Every</span>
      <input type="number" v-model.number="intervalValue" min="1" max="59" class="input w-20 h-8 text-center" @input="updateCron"/>
      <select v-model="intervalUnit" class="input h-8 w-28" @change="updateCron">
        <option value="minutes">Minutes</option>
        <option value="hours">Hours</option>
      </select>
    </div>

    <!-- Daily Mode -->
    <div v-if="mode === 'daily'" class="flex items-center gap-2">
      <span class="text-xs text-text-muted">Every day at</span>
      <input type="time" v-model="timeValue" class="input h-8 w-28 text-center" @input="updateCron"/>
    </div>

    <!-- Weekly Mode -->
    <div v-if="mode === 'weekly'" class="space-y-2">
      <div class="flex gap-1">
        <button 
          v-for="day in weekDays" :key="day.val"
          @click.prevent="toggleDay(day.val)"
          :class="['w-8 h-8 rounded text-xs flex items-center justify-center transition-colors', selectedDays.includes(day.val) ? 'bg-violet-500 text-white' : 'bg-surface-0 text-text-muted hover:bg-surface-2']"
        >
          {{ day.label }}
        </button>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-xs text-text-muted">at</span>
        <input type="time" v-model="timeValue" class="input h-8 w-28 text-center" @input="updateCron"/>
      </div>
    </div>

    <!-- Advanced Mode -->
    <div v-if="mode === 'advanced'" class="space-y-1">
      <input v-model="rawCron" class="input font-mono text-xs h-8" placeholder="0 0 * * * *" @input="updateRawCron"/>
      <p class="text-[10px] text-text-muted">Format: Seconds Minutes Hours Day Month Weekday</p>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue', 'change'])

const modes = [
  { id: 'none', label: 'Off' },
  { id: 'interval', label: 'Interval' },
  { id: 'daily', label: 'Daily' },
  { id: 'weekly', label: 'Weekly' },
  { id: 'advanced', label: 'Advanced' }
]

const weekDays = [
  { label: 'M', val: '1' },
  { label: 'T', val: '2' },
  { label: 'W', val: '3' },
  { label: 'T', val: '4' },
  { label: 'F', val: '5' },
  { label: 'S', val: '6' },
  { label: 'S', val: '0' },
]

const mode = ref('none')

// State for builders
const intervalValue = ref(15)
const intervalUnit = ref('minutes')
const timeValue = ref('09:00')
const selectedDays = ref(['1', '2', '3', '4', '5'])
const rawCron = ref('')

let internalChange = false

function setMode(m) {
  mode.value = m
  if (m === 'none') {
    internalChange = true
    emit('update:modelValue', '')
    emit('change')
  } else {
    updateCron()
  }
}

function toggleDay(val) {
  const idx = selectedDays.value.indexOf(val)
  if (idx === -1) selectedDays.value.push(val)
  else selectedDays.value.splice(idx, 1)
  updateCron()
}

function updateCron() {
  if (mode.value === 'none') return
  
  let newCron = ''
  if (mode.value === 'interval') {
    if (intervalUnit.value === 'minutes') {
      newCron = `0 */${intervalValue.value} * * * *`
    } else {
      newCron = `0 0 */${intervalValue.value} * * *`
    }
  } else if (mode.value === 'daily') {
    let [h, m] = timeValue.value.split(':')
    if (!h) h = '0'
    if (!m) m = '0'
    newCron = `0 ${parseInt(m)} ${parseInt(h)} * * *`
  } else if (mode.value === 'weekly') {
    let [h, m] = timeValue.value.split(':')
    if (!h) h = '0'
    if (!m) m = '0'
    const days = selectedDays.value.length ? selectedDays.value.sort().join(',') : '*'
    newCron = `0 ${parseInt(m)} ${parseInt(h)} * * ${days}`
  } else if (mode.value === 'advanced') {
    newCron = rawCron.value
  }

  if (newCron !== props.modelValue) {
    internalChange = true
    emit('update:modelValue', newCron)
    emit('change')
  }
}

function updateRawCron() {
  updateCron()
}

function parseIncoming(cron) {
  if (!cron) {
    mode.value = 'none'
    return
  }
  
  rawCron.value = cron
  
  const parts = cron.split(' ').filter(Boolean)
  if (parts.length !== 6) {
    mode.value = 'advanced'
    return
  }

  const [s, m, h, d, mon, w] = parts
  
  if (s !== '0') {
    mode.value = 'advanced'
    return
  }

  // Check Interval mode
  if (d === '*' && mon === '*' && w === '*') {
    if (m.startsWith('*/') && h === '*') {
      mode.value = 'interval'
      intervalUnit.value = 'minutes'
      intervalValue.value = parseInt(m.substring(2)) || 15
      return
    }
    if (m === '0' && h.startsWith('*/')) {
      mode.value = 'interval'
      intervalUnit.value = 'hours'
      intervalValue.value = parseInt(h.substring(2)) || 1
      return
    }
    // Daily mode
    if (!m.includes('*') && !m.includes('/') && !h.includes('*') && !h.includes('/')) {
      mode.value = 'daily'
      timeValue.value = `${h.padStart(2, '0')}:${m.padStart(2, '0')}`
      return
    }
  }

  // Check Weekly mode
  if (d === '*' && mon === '*' && w !== '*' && !w.includes('/')) {
    if (!m.includes('*') && !m.includes('/') && !h.includes('*') && !h.includes('/')) {
      mode.value = 'weekly'
      timeValue.value = `${h.padStart(2, '0')}:${m.padStart(2, '0')}`
      selectedDays.value = w.split(',')
      return
    }
  }

  // Fallback
  mode.value = 'advanced'
}

watch(() => props.modelValue, (newVal) => {
  if (internalChange) {
    internalChange = false
    return
  }
  parseIncoming(newVal)
}, { immediate: true })

</script>
