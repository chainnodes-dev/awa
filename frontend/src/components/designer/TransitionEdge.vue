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
        :x="-labelWidth / 2 - 6"
        :y="-10"
        :width="labelWidth + 12"
        :height="20"
        rx="4"
        :fill="isSelected ? '#1e1b4b' : '#1a1d27'"
        :stroke="strokeColor"
        stroke-width="1"
      />
      <text
        text-anchor="middle"
        dominant-baseline="middle"
        :fill="isSelected ? '#a5b4fc' : labelColor"
        font-size="11"
        font-family="JetBrains Mono, monospace"
      >{{ isTimeout ? '⏱ ' : '' }}{{ label }}</text>
      <!-- Guard chip -->
      <g v-if="data?.guard" :transform="`translate(0, 14)`">
        <text text-anchor="middle" :fill="'#f59e0b'" font-size="9" font-family="monospace" opacity="0.8">
          {{ data.guard.slice(0, 28) }}{{ data.guard.length > 28 ? '…' : '' }}
        </text>
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
  if (isSelected.value) return '#6366f1'
  return isTimeout.value ? '#92400e' : '#3a3d4a'
})
const labelColor = computed(() => isTimeout.value ? '#d97706' : '#64748b')

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

// Approximate label width
const labelWidth = computed(() => (props.label?.length ?? 0) * 7)
</script>
