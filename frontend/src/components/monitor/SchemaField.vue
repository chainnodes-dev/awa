<template>
  <div class="space-y-1.5">
    <label v-if="schema.title || name" class="block text-[10px] font-semibold text-text-muted uppercase tracking-wider">
      {{ schema.title || name }}
      <span v-if="required" class="text-red-500/80 ml-0.5">*</span>
    </label>

    <!-- Enum / Select -->
    <select
      v-if="schema.enum"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
      class="input"
      :required="required"
    >
      <option v-if="!required" value="">(Select...)</option>
      <option v-for="opt in schema.enum" :key="opt" :value="opt">{{ opt }}</option>
    </select>

    <!-- Boolean / Checkbox -->
    <div v-else-if="schema.type === 'boolean'" class="flex items-center gap-2 py-1">
      <input
        type="checkbox"
        :checked="modelValue"
        @change="$emit('update:modelValue', $event.target.checked)"
        class="w-4 h-4 rounded border-border bg-surface-1 text-accent focus:ring-accent/30"
      />
      <span class="text-xs text-text-muted">{{ schema.description || 'Enable' }}</span>
    </div>

    <!-- Number -->
    <input
      v-else-if="schema.type === 'number' || schema.type === 'integer'"
      type="number"
      :value="modelValue"
      @input="$emit('update:modelValue', Number($event.target.value))"
      class="input"
      :placeholder="schema.description || ''"
      :required="required"
    />

    <!-- Text / Default -->
    <input
      v-else
      type="text"
      :value="modelValue"
      @input="$emit('update:modelValue', $event.target.value)"
      class="input"
      :placeholder="schema.description || ''"
      :required="required"
    />

    <p v-if="schema.description && schema.type !== 'boolean'" class="text-[10px] text-text-muted italic">
      {{ schema.description }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  name: String,
  schema: { type: Object, required: true },
  modelValue: [String, Number, Boolean],
  required: Boolean
})

defineEmits(['update:modelValue'])
</script>
