<template>
  <g>
    <path
      :id="id"
      :d="path"
      fill="none"
      :stroke="strokeColor"
      stroke-width="2"
      :stroke-dasharray="isTimeout ? '6 3' : undefined"
      marker-end="url(#arrowhead)"
    />
    <!-- Label background + text -->
    <g v-if="label" :transform="`translate(${labelX}, ${labelY})`">
      <rect
        :x="-rectWidth / 2"
        :y="rectY"
        :width="rectWidth"
        :height="rectHeight"
        rx="4"
        :fill="isSelected ? 'var(--color-accent)' : 'var(--color-surface-2)'"
        :stroke="strokeColor"
        stroke-width="1"
        fill-opacity="0.8"
      />
      
      <text
        v-if="!hasGuard"
        text-anchor="middle"
        dominant-baseline="middle"
        :fill="isSelected ? 'white' : 'var(--color-text)'"
        font-size="9"
        font-family="JetBrains Mono, monospace"
      >{{ isTimeout ? '⏱ ' : '' }}{{ label }}</text>

      <g v-else>
        <text
          text-anchor="middle"
          :y="-2"
          :fill="isSelected ? 'white' : 'var(--color-text)'"
          font-size="9"
          font-family="JetBrains Mono, monospace"
        >{{ isTimeout ? '⏱ ' : '' }}{{ label }}</text>
        <text
          text-anchor="middle"
          :y="10"
          fill="var(--color-text-muted)"
          font-size="8"
          font-family="monospace"
          opacity="0.9"
        >{{ guardText }}</text>
      </g>
    </g>
  </g>
</template>

<script setup>
import { computed } from 'vue'
import { getBezierPath, useVueFlow } from '@vue-flow/core'

const props = defineProps({
  id: String,
  sourceX: Number,
  sourceY: Number,
  targetX: Number,
  targetY: Number,
  sourcePosition: String,
  targetPosition: String,
  data: Object,
  label: String,
  selected: Boolean
})

const isSelected = computed(() => props.selected)
const isTimeout  = computed(() => !!props.data?.isTimeout)

// Timeout edges render in amber; normal edges in the default slate palette.
const strokeColor = computed(() => {
  if (isSelected.value) return 'var(--color-accent)'
  return isTimeout.value ? '#92400e' : 'var(--color-border)'
})
const labelColor = computed(() => isTimeout.value ? '#d97706' : 'var(--color-text)')

// Keep as a reactive computed so it redraws when node positions change.
// (Destructuring .value immediately would freeze the values at mount time.)
const edgePath = computed(() =>
  getBezierPath({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    sourcePosition: props.sourcePosition,
    targetX: props.targetX,
    targetY: props.targetY,
    targetPosition: props.targetPosition,
  }) ?? ['', 0, 0]
)

const path   = computed(() => edgePath.value[0])
const labelX = computed(() => edgePath.value[1])
const labelY = computed(() => edgePath.value[2])

// Dynamic layout for label and guard
const hasGuard = computed(() => !!props.data?.guard)
const guardText = computed(() => {
  const g = props.data?.guard || ''
  return g.length > 28 ? g.slice(0, 28) + '…' : g
})

const labelWidth = computed(() => {
  const baseLen = (props.label?.length ?? 0) + (isTimeout.value ? 2 : 0)
  return baseLen * 6.5
})
const guardWidth = computed(() => guardText.value.length * 5.5)

const rectWidth = computed(() => {
  if (!hasGuard.value) return labelWidth.value + 12
  return Math.max(labelWidth.value, guardWidth.value) + 16
})
const rectHeight = computed(() => hasGuard.value ? 32 : 20)
const rectY = computed(() => hasGuard.value ? -14 : -10)
</script>
