<template>
  <div class="code-editor-wrapper relative rounded-lg overflow-hidden border border-border bg-[#1a1b26]">
    <div ref="editorEl" class="text-xs" />
    <!-- Language badge -->
    <div class="absolute top-2 right-2 text-[9px] font-mono text-slate-600 select-none pointer-events-none uppercase tracking-widest">
      {{ language }}
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, drawSelection, highlightActiveLineGutter, highlightActiveLine } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { javascript } from '@codemirror/lang-javascript'
import { oneDark } from '@codemirror/theme-one-dark'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { syntaxHighlighting, defaultHighlightStyle, bracketMatching } from '@codemirror/language'
import { closeBrackets, closeBracketsKeymap, autocompletion, completionKeymap } from '@codemirror/autocomplete'

const props = defineProps({
  modelValue: { type: String, default: '' },
  language:   { type: String, default: 'javascript' },
  // bbSchema: { fieldName: { type: 'string'|'number'|'bool'|'object' } }
  bbSchema:   { type: Object, default: () => ({}) },
  height:     { type: String, default: '260px' },
  readonly:   { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const editorEl = ref(null)
const view = shallowRef(null)

// ── bb autocomplete ──────────────────────────────────────────────────────────
function bbCompletions(context) {
  // Trigger on "bb." to suggest blackboard field names
  const before = context.matchBefore(/bb\.\w*/)
  if (!before && !context.explicit) return null
  const prefix = before ? before.text.slice(3) : '' // strip "bb."
  const options = Object.entries(props.bbSchema).map(([name, field]) => ({
    label: name,
    type:  field.type === 'number' ? 'variable' : 'property',
    detail: field.type ?? '',
    apply: name,
    boost: 10,
  }))
  const filtered = prefix
    ? options.filter(o => o.label.startsWith(prefix))
    : options
  return { from: before ? before.from + 3 : context.pos, options: filtered }
}

// ── global snippets ──────────────────────────────────────────────────────────
const globalSnippets = [
  { label: 'trigger', type: 'function', detail: 'fire a trigger (early exit)', apply: "trigger('name');" },
  { label: 'console.log', type: 'function', detail: 'log output', apply: "console.log();" },
  { label: 'return trigger', type: 'keyword', detail: 'return trigger + optional updates', apply: "return {\n  blackboard_updates: {},\n  trigger: 'name',\n  reasoning: ''\n};" },
]

function globalCompletions(context) {
  const word = context.matchBefore(/\w+/)
  if (!word && !context.explicit) return null
  const prefix = word?.text ?? ''
  const filtered = globalSnippets.filter(s => s.label.startsWith(prefix))
  if (!filtered.length) return null
  return { from: word ? word.from : context.pos, options: filtered }
}

// ── Build editor ─────────────────────────────────────────────────────────────
function buildExtensions() {
  const isText = props.language === 'text'

  const base = [
    drawSelection(),
    highlightActiveLine(),
    history(),
    oneDark,
    syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
    keymap.of([
      ...defaultKeymap,
      ...historyKeymap,
    ]),
    EditorView.updateListener.of(update => {
      if (update.docChanged) {
        emit('update:modelValue', update.state.doc.toString())
      }
    }),
    EditorView.theme({
      '&': { height: props.height, fontSize: '12px' },
      '.cm-scroller': { overflow: 'auto', fontFamily: isText ? "'Inter', 'system-ui', sans-serif" : "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace" },
      '.cm-content': { padding: '8px 0' },
    }),
    props.readonly ? EditorState.readOnly.of(true) : [],
  ]

  if (isText) {
    // Plain-text mode: word wrap, no line numbers, no bracket/autocomplete
    return [...base, EditorView.lineWrapping]
  }

  // JavaScript mode: full featured
  return [
    ...base,
    lineNumbers(),
    highlightActiveLineGutter(),
    bracketMatching(),
    closeBrackets(),
    javascript(),
    autocompletion({ override: [bbCompletions, globalCompletions] }),
    keymap.of([...closeBracketsKeymap, ...completionKeymap]),
  ]
}

onMounted(() => {
  view.value = new EditorView({
    state: EditorState.create({
      doc: props.modelValue,
      extensions: buildExtensions(),
    }),
    parent: editorEl.value,
  })
})

// Sync external model changes into the editor (e.g. when switching states)
watch(() => props.modelValue, (newVal) => {
  if (!view.value) return
  const current = view.value.state.doc.toString()
  if (current === newVal) return
  view.value.dispatch({
    changes: { from: 0, to: current.length, insert: newVal ?? '' },
  })
})

onBeforeUnmount(() => {
  view.value?.destroy()
})
</script>

<style>
/* Override oneDark background to match ASM surface */
.code-editor-wrapper .cm-editor {
  background: #1a1b26;
}
.code-editor-wrapper .cm-gutters {
  background: #1a1b26;
  border-right: 1px solid #2a2d3a;
}
.code-editor-wrapper .cm-activeLineGutter,
.code-editor-wrapper .cm-activeLine {
  background: rgba(255,255,255,0.04);
}
</style>
