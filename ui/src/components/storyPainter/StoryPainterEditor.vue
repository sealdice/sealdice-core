<template>
  <CodeMirror
    v-if="!lazy"
    v-model="code"
    class="story-painter-editor"
    :extensions="editorExtensions as never[]"
    :dark="false"
    :wrap="true"
  />
  <div v-else class="story-painter-editor story-painter-editor-lazy">
    <n-empty description="编辑器需要载入完整文本后才能修改">
      <template #extra>
        <n-button type="primary" secondary @click="emit('loadFull')">
          <template #icon>
            <n-icon><i-carbon-document /></n-icon>
          </template>
          载入完整文本
        </n-button>
      </template>
    </n-empty>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, shallowRef, watch } from 'vue';
import { basicSetup } from 'codemirror';
import { Decoration, EditorView, ViewPlugin, type DecorationSet, type ViewUpdate } from '@codemirror/view';
import type { Extension } from '@codemirror/state';
import type { StoryPainterChar, StoryPainterLogItem, StoryPainterOptions } from '@/features/storyPainter/types';
import { storyPainterTime } from '@/features/storyPainter/formatters';

const CodeMirror = defineAsyncComponent(() => import('vue-codemirror6'));

const props = defineProps<{
  items: StoryPainterLogItem[];
  chars: StoryPainterChar[];
  options: StoryPainterOptions;
  highlight: boolean;
  lazy?: boolean;
}>();

const emit = defineEmits<{
  updateItems: [items: StoryPainterLogItem[]];
  loadFull: [];
}>();

const code = shallowRef('');
const internalUpdate = shallowRef(false);

const editorExtensions = computed<Extension[]>(() => [
  basicSetup,
  EditorView.lineWrapping,
  props.highlight ? storyLogHighlightExtension(props.chars, props.options) : [],
]);

watch(
  () => props.items,
  () => {
    if (props.lazy) {
      internalUpdate.value = true;
      code.value = '';
      queueMicrotask(() => {
        internalUpdate.value = false;
      });
      return;
    }
    internalUpdate.value = true;
    code.value = exportEditableText(props.items);
    queueMicrotask(() => {
      internalUpdate.value = false;
    });
  },
  { immediate: true },
);

watch(code, (value) => {
  if (internalUpdate.value) return;
  emit('updateItems', parseEditableText(value, props.items));
});

function exportEditableText(items: StoryPainterLogItem[]): string {
  return items.map((item) => `${item.nickname}(${item.IMUserId}) ${storyPainterTime(item, { ...props.options, timeHide: false, yearHide: false })}\n${item.message.trimEnd()}\n`).join('\n');
}

function parseEditableText(text: string, originalItems: StoryPainterLogItem[]): StoryPainterLogItem[] {
  const blocks = text.split(/\n{2,}/).filter((block) => block.trim() !== '');
  return blocks.map((block, index) => {
    const [head = '', ...messageLines] = block.split(/\r?\n/);
    const matched = head.match(/^(.+?)\((.*?)\)\s+(.+)$/);
    const original = originalItems[index];
    if (!matched) {
      return {
        ...(original ?? blankItem(index)),
        message: block,
      };
    }
    return {
      ...(original ?? blankItem(index)),
      nickname: matched[1] ?? original?.nickname ?? '',
      IMUserId: matched[2] ?? original?.IMUserId ?? '',
      message: messageLines.join('\n'),
    };
  });
}

function blankItem(index: number): StoryPainterLogItem {
  return {
    id: index + 1,
    nickname: '未知',
    IMUserId: '',
    time: 0,
    message: '',
    isDice: false,
    commandId: 0,
    version: 105,
  };
}

function storyLogHighlightExtension(chars: StoryPainterChar[], options: StoryPainterOptions): Extension {
  const nameColorMap = new Map(chars.map((char) => [char.name, char.role === '隐藏' ? '#6b7280' : char.color]));
  const buildDecorations = (view: EditorView) => {
    const decorations = [];
    for (const range of view.visibleRanges) {
      let pos = range.from;
      while (pos <= range.to) {
        const line = view.state.doc.lineAt(pos);
        const matched = line.text.match(/^(.+?)\(.+?\)\s+\d{4}\//);
        const color = matched?.[1] ? nameColorMap.get(matched[1]) : undefined;
        if (color) {
          decorations.push(Decoration.line({ attributes: { style: `color: ${color}` } }).range(line.from));
        }
        if (line.to + 1 <= pos) break;
        pos = line.to + 1;
      }
    }
    void options;
    return Decoration.set(decorations, true);
  };

  type HighlightPlugin = { decorations: DecorationSet };
  return ViewPlugin.fromClass(class implements HighlightPlugin {
    decorations: DecorationSet;

    constructor(view: EditorView) {
      this.decorations = buildDecorations(view);
    }

    update(update: ViewUpdate) {
      if (update.docChanged || update.viewportChanged) {
        this.decorations = buildDecorations(update.view);
      }
    }
  }, {
    decorations: (plugin: HighlightPlugin) => plugin.decorations,
  });
}
</script>

<style scoped>
.story-painter-editor {
  border: 1px solid var(--sd-border);
  min-height: 36rem;
}

.story-painter-editor :deep(.cm-editor) {
  height: 70vh;
}

.story-painter-editor-lazy {
  min-height: 36rem;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
