<template>
  <div class="relative" ref="rootEl">
    <div class="relative">
      <input
        ref="inputEl"
        class="input font-mono pr-8"
        :value="modelValue"
        :placeholder="placeholder"
        autocomplete="off"
        @input="onInput"
        @focus="onFocus"
        @keydown="onKeydown"
      />
      <button
        v-if="options.length || modelValue"
        type="button"
        tabindex="-1"
        class="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text transition-colors"
        @mousedown.prevent="toggleDropdown"
      >
        <svg
          class="w-3.5 h-3.5 transition-transform duration-150"
          :class="open ? 'rotate-180' : ''"
          viewBox="0 0 20 20" fill="currentColor"
        >
          <path fill-rule="evenodd" d="M5.23 7.21a.75.75 0 011.06.02L10 11.168l3.71-3.938a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z" clip-rule="evenodd" />
        </svg>
      </button>
    </div>

    <Transition
      enter-active-class="transition ease-out duration-100"
      enter-from-class="opacity-0 -translate-y-1"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-75"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-1"
    >
      <ul
        v-if="open && (filteredOptions.length || modelValue)"
        class="absolute z-50 mt-1 w-full rounded-lg border border-border bg-surface-2 shadow-xl shadow-black/40 overflow-hidden"
      >
        <li
          v-for="(opt, i) in filteredOptions"
          :key="opt"
          :class="[
            'flex items-center gap-2 px-3 py-2 cursor-pointer text-sm font-mono transition-colors',
            i === cursor
              ? 'bg-accent/20 text-accent'
              : 'text-text hover:bg-white/5 hover:text-slate-100',
          ]"
          @mousedown.prevent="select(opt)"
        >
          <span class="w-1.5 h-1.5 rounded-full bg-indigo-400/60 shrink-0" />
          {{ opt }}
        </li>
        <li v-if="!filteredOptions.length && modelValue" class="px-3 py-2 text-[10px] text-text-muted italic border-t border-border/50 bg-surface-1/50">
          Press enter to use "{{ modelValue }}" as a new trigger
        </li>
      </ul>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps({
  modelValue: { type: String,  default: '' },
  options:    { type: Array,   default: () => [] }, // Array of strings
  placeholder:{ type: String,  default: 'trigger_name' },
})
const emit = defineEmits(['update:modelValue'])

const rootEl  = ref(null)
const inputEl = ref(null)
const open    = ref(false)
const cursor  = ref(-1)

const filteredOptions = computed(() => {
  const q = props.modelValue?.trim().toLowerCase()
  if (!q) return props.options
  return props.options.filter(o => o.toLowerCase().includes(q))
})

function onInput(e) {
  emit('update:modelValue', e.target.value)
  open.value   = true
  cursor.value = -1
}

function onFocus() {
  if (props.options.length) open.value = true
}

function toggleDropdown() {
  open.value = !open.value
  if (open.value) inputEl.value?.focus()
}

function select(val) {
  emit('update:modelValue', val)
  open.value   = false
  cursor.value = -1
}

function onKeydown(e) {
  if (e.key === 'ArrowDown') {
    if (!open.value) { open.value = true; return }
    e.preventDefault()
    cursor.value = Math.min(cursor.value + 1, filteredOptions.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    if (!open.value) return
    e.preventDefault()
    cursor.value = Math.max(cursor.value - 1, -1)
  } else if (e.key === 'Enter') {
    if (open.value && cursor.value >= 0) {
      e.preventDefault()
      select(filteredOptions.value[cursor.value])
    } else {
      open.value = false
    }
  } else if (e.key === 'Escape') {
    open.value   = false
    cursor.value = -1
  }
}

function onOutsideClick(e) {
  if (rootEl.value && !rootEl.value.contains(e.target)) {
    open.value   = false
    cursor.value = -1
  }
}
onMounted(()        => document.addEventListener('mousedown', onOutsideClick))
onBeforeUnmount(()  => document.removeEventListener('mousedown', onOutsideClick))
</script>
