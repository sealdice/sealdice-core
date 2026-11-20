<template>
  <div
    ref="root"
    class="text-code-editor"
    :style="{ height: `${height}px` }"
    @focusout="onFocusOut"
  >
    <div class="editor-body">
      <div v-show="!active" class="passive-editor">
        <pre
          ref="highlightLayer"
          class="passive-highlight"
          aria-hidden="true"
        ><template v-for="(part, index) in parts" :key="index"><span :class="part.classes">{{ part.text }}</span></template>{{ '\n' }}</pre>
        <textarea
          v-show="!active"
          ref="input"
          :value="modelValue"
          aria-label="文案代码编辑器"
          spellcheck="false"
          wrap="soft"
          @focus="activate"
          @input="onInput"
          @scroll="syncScroll"
          @compositionstart="composing = true"
          @compositionend="onCompositionEnd"
        />
      </div>
      <div v-show="active" ref="host" class="active-editor" />
    </div>
    <div v-if="loadFailed" class="editor-notice" role="status">
      编辑增强功能加载失败，可继续输入；重新聚焦可重试。
    </div>
    <div
      class="resize-handle"
      role="separator"
      tabindex="0"
      aria-label="调整文案框高度"
      aria-orientation="horizontal"
      :aria-valuenow="height"
      :aria-valuemin="56"
      @pointerdown="startResize"
      @pointermove="moveResize"
      @pointerup="endResize"
      @pointercancel="endResize"
      @keydown="resizeByKey"
    >
      <span />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
import { passiveHighlights } from '@/features/customText/passiveHighlight';
import type { ActiveTextEditor, EditorSnapshot } from '@/features/customText/codeEditor';

const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ 'update:modelValue': [text: string] }>();
const root = ref<HTMLDivElement>();
const host = ref<HTMLDivElement>();
const input = ref<HTMLTextAreaElement>();
const highlightLayer = ref<HTMLPreElement>();
const height = ref(96);
const active = ref(false);
const loadFailed = ref(false);
const composing = ref(false);
const parts = computed(() => (active.value ? [] : passiveHighlights(props.modelValue)));
let editor: ActiveTextEditor | undefined;
let generation = 0;
let disposed = false;
let switching = false;
let blurTimer: ReturnType<typeof setTimeout> | undefined;
let resize: { pointerId: number; y: number; height: number } | undefined;
let savedSelection: EditorSnapshot = { anchor: 0, head: 0, scrollTop: 0 };

function syncScroll() {
  if (!input.value || !highlightLayer.value) return;
  highlightLayer.value.scrollTop = input.value.scrollTop;
  highlightLayer.value.scrollLeft = input.value.scrollLeft;
}
function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value);
}
function onCompositionEnd() {
  composing.value = false;
  void activate();
}
async function activate() {
  clearTimeout(blurTimer);
  if (editor || disposed || switching) return;
  const id = ++generation;
  try {
    const { createTextEditor } = await import('@/features/customText/codeEditor');
    if (disposed || id !== generation || composing.value || document.activeElement !== input.value)
      return;
    const field = input.value!;
    const start = field.selectionStart;
    const end = field.selectionEnd;
    savedSelection = {
      anchor: field.selectionDirection === 'backward' ? end : start,
      head: field.selectionDirection === 'backward' ? start : end,
      scrollTop: field.scrollTop,
    };
    switching = true;
    active.value = true;
    await nextTick();
    if (disposed || id !== generation || !host.value) {
      active.value = false;
      return;
    }
    editor = createTextEditor(host.value, field.value, savedSelection, value => {
      if (value !== props.modelValue) emit('update:modelValue', value);
    });
    loadFailed.value = false;
  } catch {
    if (disposed || id !== generation) return;
    const restoreFocus =
      switching &&
      (document.activeElement === document.body || root.value?.contains(document.activeElement));
    host.value?.replaceChildren();
    active.value = false;
    loadFailed.value = true;
    await nextTick();
    if (restoreFocus && !disposed && id === generation && input.value) {
      input.value.focus();
      restoreNativeSelection();
    }
  } finally {
    switching = false;
  }
}
function restoreNativeSelection() {
  if (disposed || !input.value) return;
  input.value.setSelectionRange(
    Math.min(savedSelection.anchor, savedSelection.head),
    Math.max(savedSelection.anchor, savedSelection.head),
    savedSelection.anchor > savedSelection.head ? 'backward' : 'forward'
  );
  input.value.scrollTop = savedSelection.scrollTop;
  syncScroll();
}
function onFocusOut(event: FocusEvent) {
  if (switching) return;
  if (event.relatedTarget instanceof Node && root.value?.contains(event.relatedTarget)) return;
  generation++;
  clearTimeout(blurTimer);
  blurTimer = setTimeout(() => {
    if (disposed || root.value?.contains(document.activeElement)) return;
    if (editor) savedSelection = editor.snapshot();
    editor?.destroy();
    editor = undefined;
    active.value = false;
    void nextTick(restoreNativeSelection);
  }, 0);
}
function startResize(event: PointerEvent) {
  if (event.button !== 0) return;
  event.preventDefault();
  const handle = event.currentTarget as HTMLElement;
  handle.setPointerCapture(event.pointerId);
  resize = { pointerId: event.pointerId, y: event.clientY, height: height.value };
}
function moveResize(event: PointerEvent) {
  if (resize?.pointerId === event.pointerId)
    height.value = Math.max(56, resize.height + event.clientY - resize.y);
}
function endResize(event: PointerEvent) {
  if (resize?.pointerId !== event.pointerId) return;
  const handle = event.currentTarget as HTMLElement;
  if (handle.hasPointerCapture(event.pointerId)) handle.releasePointerCapture(event.pointerId);
  resize = undefined;
}
function resizeByKey(event: KeyboardEvent) {
  if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return;
  event.preventDefault();
  height.value = Math.max(56, height.value + (event.key === 'ArrowUp' ? -20 : 20));
}
watch(
  () => props.modelValue,
  value => {
    editor?.setText(value);
    void nextTick(syncScroll);
  }
);
onBeforeUnmount(() => {
  disposed = true;
  generation++;
  clearTimeout(blurTimer);
  editor?.destroy();
});
</script>

<style scoped>
.text-code-editor {
  display: flex;
  flex-direction: column;
  min-height: 56px;
  border: 1px solid var(--sd-border);
  border-radius: 6px;
  overflow: hidden;
  color: var(--sd-text-primary);
  background: var(--sd-bg-elevated);
}
.text-code-editor:focus-within {
  border-color: var(--sd-primary);
  box-shadow: 0 0 0 1px var(--sd-primary-border);
}
.editor-body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.editor-notice {
  flex: none;
  padding: 2px 10px;
  font-size: 12px;
  color: var(--sd-text-secondary);
  background: var(--sd-bg-control);
}
.passive-editor,
.active-editor {
  height: 100%;
}
.passive-highlight,
textarea {
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 8px 30px 8px 10px;
  border: 0;
  font-family: var(--sd-font-code);
  font-size: 14px;
  font-weight: 400;
  line-height: 1.7;
  letter-spacing: normal;
  tab-size: 4;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: normal;
}
.passive-highlight {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}
textarea {
  position: relative;
  display: block;
  resize: none;
  color: transparent;
  -webkit-text-fill-color: transparent;
  caret-color: var(--sd-text-primary);
  background: transparent;
  outline: none;
  overflow-y: scroll;
}
textarea::selection {
  background: var(--sd-primary-soft-strong);
}
.passive-highlight {
  padding: 8px 30px 8px 10px;
  overflow-y: scroll;
  scrollbar-color: transparent transparent;
}
.resize-handle {
  display: flex;
  flex: 0 0 12px;
  align-items: center;
  justify-content: center;
  cursor: ns-resize;
  touch-action: none;
  background: var(--sd-bg-control);
  border-top: 1px solid var(--sd-border-soft);
}
.resize-handle span {
  width: 28px;
  height: 2px;
  background: var(--sd-text-muted);
  border-radius: 2px;
}
.resize-handle:focus-visible {
  outline: 2px solid var(--sd-primary);
  outline-offset: -2px;
}
.text-code-editor :deep(.cm-editor) {
  height: 100%;
  color: var(--sd-text-primary);
  background: var(--sd-bg-elevated);
  outline: none;
}
.text-code-editor :deep(.cm-scroller) {
  overflow: auto;
  font-family: var(--sd-font-code);
  font-size: 14px;
  line-height: 1.7;
}
.text-code-editor :deep(.cm-content) {
  padding: 8px 30px 8px 10px;
}
.text-code-editor :deep(.cm-gutters) {
  display: none;
  background: var(--sd-bg-control);
  color: var(--sd-text-muted);
  border-color: var(--sd-border-soft);
}
.text-code-editor :deep(.cm-line) {
  padding: 0;
}
.text-code-editor :deep(.cm-cursor) {
  border-left-color: var(--sd-text-primary);
}
.text-code-editor :deep(.cm-activeLine) {
  background: var(--sd-primary-soft);
}
.text-code-editor :deep(.seal-expression) {
  background: var(--sd-primary-soft);
}
.text-code-editor :deep(.seal-interpolation) {
  background: var(--sd-primary-soft-strong);
}
.text-code-editor :deep(.seal-variable),
.text-code-editor :deep(.seal-keyword) {
  color: var(--sd-primary);
}
.text-code-editor :deep(.seal-delimiter),
.text-code-editor :deep(.seal-escape) {
  color: var(--sd-accent-strong);
  font-weight: 600;
}
.text-code-editor :deep(.seal-string) {
  color: var(--sd-success);
}
.text-code-editor :deep(.seal-number),
.text-code-editor :deep(.seal-dice),
.text-code-editor :deep(.seal-operator) {
  color: var(--sd-text-secondary);
}
.text-code-editor :deep(.seal-comment) {
  color: var(--sd-text-muted);
  font-style: italic;
}
</style>
