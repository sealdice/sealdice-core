<template>
  <section class="story-painter-preview">
    <div v-if="mode !== 'preview'" class="preview-toolbar">
      <n-flex size="small" align="center" wrap>
        <template v-if="mode === 'bbs' || mode === 'bbspineapple'">
          <n-checkbox
            :checked="forumOptions.bbsUseSpaceWithMultiLine"
            @update:checked="value => emit('updateForumOptions', { bbsUseSpaceWithMultiLine: value })"
          >
            多行空格缩进
          </n-checkbox>
          <n-checkbox
            :checked="forumOptions.bbsUseColorName"
            @update:checked="value => emit('updateForumOptions', { bbsUseColorName: value })"
          >
            使用颜色名
          </n-checkbox>
        </template>
        <n-checkbox v-if="mode === 'trg'" :checked="addVoiceMark" @update:checked="value => emit('updateAddVoiceMark', value)">
          添加语音合成标记
        </n-checkbox>
        <n-button size="small" type="primary" secondary @click="copyPreviewText">
          <template #icon>
            <n-icon><i-carbon-copy /></n-icon>
          </template>
          复制
        </n-button>
      </n-flex>
    </div>

    <div v-if="previewSources.length === 0" class="preview-empty">
      <n-empty description="没有可展示的日志条目" />
    </div>

    <div v-else-if="mode === 'preview' || mode === 'bbs' || mode === 'trg'" ref="previewList" class="preview-list">
      <div class="preview-virtual-spacer" :style="{ height: `${virtualTotalSize}px` }">
        <div
          v-for="{ virtualRow, item } in renderedRows"
          :key="String(virtualRow.key)"
          :ref="measureVirtualRow"
          class="preview-virtual-row"
          :data-index="virtualRow.index"
          :style="{ transform: `translateY(${virtualRow.start}px)` }"
        >
          <StoryPainterPreviewRow
            v-if="item"
            :source="item"
            :mode="mode"
            :chars="chars"
            :options="options"
            :forum-options="forumOptions"
            :color="colorByItem(item)"
            :add-voice-mark="addVoiceMark"
          />
          <div v-else class="preview-row-loading">...</div>
        </div>
      </div>
    </div>

    <div v-else class="preview-list">
      <div v-if="props.itemIndexes.length > 0" class="copy-note">长文本复制/导出会分块读取完整内容。</div>
      <pre v-for="item in textItems" :key="item.index" class="text-row">{{ item.text }}</pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue';
import { useVirtualizer, type VirtualItem, type Virtualizer } from '@tanstack/vue-virtual';
import type { VNodeRef } from 'vue';
import { storyPainterDebug } from '@/features/storyPainter/debug';
import type {
  StoryPainterChar,
  StoryPainterForumOptions,
  StoryPainterLogItem,
  StoryPainterOptions,
  StoryPainterPreviewDisplayMode,
} from '@/features/storyPainter/types';
import { packStoryPainterNameId } from '@/features/storyPainter/types';
import { renderForumText, renderPineappleForumBlocks, renderTrgText } from '@/features/storyPainter/renderers';
import {
  buildStoryPainterPreviewSources,
  collectMissingVisibleIndexes,
  virtualItemsToStoryPainterRange,
  type StoryPainterPreviewSource,
  type StoryPainterVirtualRange,
} from '@/features/storyPainter/virtualRange';
import StoryPainterPreviewRow from './StoryPainterPreviewRow.vue';

const props = defineProps<{
  mode: StoryPainterPreviewDisplayMode;
  items: StoryPainterLogItem[];
  itemIndexes: number[];
  readItem?: (index: number) => Promise<StoryPainterLogItem | undefined>;
  readItems?: (indexes: number[]) => Promise<StoryPainterLogItem[]>;
  chars: StoryPainterChar[];
  options: StoryPainterOptions;
  forumOptions: StoryPainterForumOptions;
  addVoiceMark: boolean;
  copyText?: () => Promise<string>;
}>();

const emit = defineEmits<{
  copy: [text: string];
  updateAddVoiceMark: [value: boolean];
  updateForumOptions: [value: Partial<StoryPainterForumOptions>];
}>();

type PreviewSource = StoryPainterPreviewSource<StoryPainterLogItem>;
type PreviewRenderedRow = {
  virtualRow: VirtualItem;
  item?: StoryPainterLogItem;
};

const previewListRef = useTemplateRef<HTMLDivElement>('previewList');
const loadedItems = shallowRef(new Map<number, StoryPainterLogItem>());
const loadedItemsLimit = 2000;
const rangeOverscan = 20;
const virtualRenderOverscan = 20;
const scrollLoadDelayMs = 20;
const pendingIndexes = new Set<number>();
let scheduledRange: StoryPainterVirtualRange | null = null;
let scrollLoadTimer: ReturnType<typeof setTimeout> | undefined;

const pcMap = computed(() => {
  const map = new Map<string, StoryPainterChar>();
  props.chars.forEach((char) => map.set(packStoryPainterNameId(char), char));
  return map;
});

const previewSources = computed<PreviewSource[]>(() => buildStoryPainterPreviewSources(props.itemIndexes, props.items));

const rowVirtualizer = useVirtualizer<HTMLDivElement, HTMLDivElement>(computed(() => ({
  count: previewSources.value.length,
  getScrollElement: () => previewListRef.value,
  estimateSize: () => 28,
  overscan: virtualRenderOverscan,
  getItemKey: index => previewSources.value[index]?.key ?? index,
  onChange: handleVirtualizerChange,
  useAnimationFrameWithResizeObserver: true,
})));

const virtualItems = computed(() => rowVirtualizer.value.getVirtualItems());
const virtualTotalSize = computed(() => rowVirtualizer.value.getTotalSize());
const renderedRows = computed<PreviewRenderedRow[]>(() => virtualItems.value.flatMap((virtualRow) => {
  const source = previewSources.value[virtualRow.index];
  if (!source) return [];
  return [{
    virtualRow,
    item: source.item ?? loadedItems.value.get(source.index),
  }];
}));
const measureVirtualRow: VNodeRef = (node) => {
  if (node === null || node instanceof HTMLDivElement) {
    rowVirtualizer.value.measureElement(node);
  }
};

const textItems = computed(() => {
  if (props.mode === 'bbspineapple') {
    const items = props.itemIndexes.length > 0 ? props.items : materializedItems.value;
    return renderPineappleForumBlocks(items, props.chars, props.options, props.forumOptions, colorByItem)
      .map((text, index) => ({ index, text }));
  }
  return materializedItems.value.map((item) => ({
    index: item.index ?? item.id,
    text: renderText(item),
  }));
});

const materializedItems = computed(() => {
  if (props.itemIndexes.length > 0) {
    return props.itemIndexes
      .map(index => loadedItems.value.get(index))
      .filter((item): item is StoryPainterLogItem => Boolean(item));
  }
  return previewSources.value
    .map(source => source.item)
    .filter((item): item is StoryPainterLogItem => Boolean(item));
});

watch(
  () => props.items,
  () => {
    if (props.itemIndexes.length === 0) {
      loadedItems.value = new Map(props.items.map((item, index) => [item.index ?? index, item]));
      storyPainterDebug('preview:items-loaded', {
        itemCount: props.items.length,
        loadedCount: loadedItems.value.size,
      });
    }
  },
  { immediate: true },
);

watch(
  () => props.itemIndexes,
  () => {
    clearScheduledScrollLoad();
    pendingIndexes.clear();
    const visibleSet = new Set(props.itemIndexes);
    loadedItems.value = new Map([...loadedItems.value].filter(([index]) => visibleSet.has(index)));
    storyPainterDebug('preview:item-indexes', {
      itemIndexCount: props.itemIndexes.length,
      firstIndex: props.itemIndexes[0],
      lastIndex: props.itemIndexes.at(-1),
      keptLoadedCount: loadedItems.value.size,
    });
    rowVirtualizer.value.measure();
    void loadVisibleItems({ start: 0, end: 79 });
  },
  { immediate: true },
);

onMounted(() => {
  void loadVisibleItems({ start: 0, end: 79 });
  void nextTick(() => {
    scheduleVisibleLoad(virtualItemsToStoryPainterRange(rowVirtualizer.value.getVirtualItems()), {
      source: 'mounted',
      sync: false,
    });
  });
});

onBeforeUnmount(() => {
  clearScheduledScrollLoad();
});

function colorByItem(item: StoryPainterLogItem): string {
  return pcMap.value.get(packStoryPainterNameId(item))?.color || '#4b5563';
}

function renderText(item: StoryPainterLogItem): string {
  if (props.mode === 'trg') {
    return renderTrgText(item, props.chars, props.options, props.addVoiceMark);
  }
  return renderForumText(item, props.chars, props.options, props.forumOptions, colorByItem(item));
}

async function copyPreviewText(): Promise<void> {
  if (props.copyText) {
    emit('copy', await props.copyText());
    return;
  }
  emit('copy', textItems.value.map((item) => item.text).join('\n'));
}

async function loadVisibleItems(range: StoryPainterVirtualRange): Promise<void> {
  if ((!props.readItem && !props.readItems) || props.itemIndexes.length === 0) {
    storyPainterDebug('preview:load-skip', {
      reason: props.itemIndexes.length === 0 ? 'empty-item-indexes' : 'missing-reader',
      range: compactRange(range),
      itemIndexCount: props.itemIndexes.length,
    });
    return;
  }
  const indexes = collectMissingVisibleIndexes({
    visibleIndexes: props.itemIndexes,
    loadedIndexes: new Set(loadedItems.value.keys()),
    pendingIndexes,
    range,
    overscan: rangeOverscan,
  });
  if (indexes.length === 0) {
    storyPainterDebug('preview:load-skip', {
      reason: 'no-missing-indexes',
      range: compactRange(range),
      loadedCount: loadedItems.value.size,
      pendingCount: pendingIndexes.size,
    });
    return;
  }
  storyPainterDebug('preview:load-start', {
    range: compactRange(range),
    missingCount: indexes.length,
    firstMissingIndex: indexes[0],
    lastMissingIndex: indexes.at(-1),
    loadedCount: loadedItems.value.size,
    pendingCount: pendingIndexes.size,
  });
  indexes.forEach(index => pendingIndexes.add(index));
  try {
    const rows = props.readItems
      ? await props.readItems(indexes)
      : (await Promise.all(indexes.map(index => props.readItem?.(index)))).filter((item): item is StoryPainterLogItem => Boolean(item));
    const visibleSet = new Set(props.itemIndexes);
    const next = new Map(loadedItems.value);
    let insertedCount = 0;
    rows.forEach((item) => {
      if (item.index === undefined) return;
      if (!visibleSet.has(item.index)) return;
      if (next.has(item.index)) next.delete(item.index);
      next.set(item.index, item);
      insertedCount += 1;
    });
    loadedItems.value = trimLoadedItems(next, indexes);
    storyPainterDebug('preview:load-done', {
      range: compactRange(range),
      requestedCount: indexes.length,
      rowCount: rows.length,
      insertedCount,
      loadedCount: loadedItems.value.size,
      firstRowIndex: rows[0]?.index,
      lastRowIndex: rows.at(-1)?.index,
    });
  } finally {
    indexes.forEach(index => pendingIndexes.delete(index));
  }
}

function handleVirtualizerChange(instance: Virtualizer<HTMLDivElement, HTMLDivElement>, sync: boolean): void {
  const range = virtualItemsToStoryPainterRange(instance.getVirtualItems());
  scheduleVisibleLoad(range, {
    source: 'tanstack',
    sync,
    totalSize: Math.round(instance.getTotalSize()),
    virtualCount: instance.getVirtualItems().length,
  });
}

function scheduleVisibleLoad(
  range: StoryPainterVirtualRange | undefined,
  context: Record<string, number | boolean | string | undefined>,
): void {
  if (!range) {
    storyPainterDebug('preview:scroll-skip', {
      reason: 'empty-virtual-range',
      ...context,
      itemIndexCount: props.itemIndexes.length,
      loadedCount: loadedItems.value.size,
      pendingCount: pendingIndexes.size,
    });
    return;
  }
  clearScheduledScrollLoad();
  scheduledRange = range;
  const target = previewListRef.value;
  storyPainterDebug('preview:scroll', {
    ...context,
    range: compactRange(range),
    itemIndexCount: props.itemIndexes.length,
    loadedCount: loadedItems.value.size,
    pendingCount: pendingIndexes.size,
    scrollTop: target?.scrollTop,
    clientHeight: target?.clientHeight,
    scrollHeight: target?.scrollHeight,
  });
  storyPainterDebug('preview:schedule-load', {
    delayMs: scrollLoadDelayMs,
    range: compactRange(range),
  });
  scrollLoadTimer = setTimeout(() => {
    const rangeToLoad = scheduledRange;
    scheduledRange = null;
    scrollLoadTimer = undefined;
    if (rangeToLoad) void loadVisibleItems(rangeToLoad);
  }, scrollLoadDelayMs);
}

function clearScheduledScrollLoad(): void {
  if (scrollLoadTimer) clearTimeout(scrollLoadTimer);
  scrollLoadTimer = undefined;
  scheduledRange = null;
}

function compactRange(range: StoryPainterVirtualRange): Record<string, number | undefined> {
  return {
    start: range.start,
    end: range.end,
    padFront: range.padFront,
    padBehind: range.padBehind,
  };
}

function trimLoadedItems(
  source: Map<number, StoryPainterLogItem>,
  protectedIndexes: number[],
): Map<number, StoryPainterLogItem> {
  const protectedSet = new Set(protectedIndexes);
  const next = new Map(source);
  for (const index of source.keys()) {
    if (next.size <= loadedItemsLimit) break;
    if (protectedSet.has(index)) continue;
    next.delete(index);
  }
  return next;
}
</script>

<style scoped>
.story-painter-preview {
  border: 1px solid var(--sd-border);
  background: var(--sd-bg-elevated);
  min-height: 30rem;
}

.preview-toolbar {
  border-bottom: 1px solid var(--sd-border);
  padding: 0.65rem 0.8rem;
}

.preview-empty {
  padding: 2rem;
}

.preview-list {
  height: 70vh;
  overflow: auto;
  padding: 0.8rem;
}

.preview-virtual-spacer {
  position: relative;
  width: 100%;
}

.preview-virtual-row {
  left: 0;
  position: absolute;
  top: 0;
  width: 100%;
}

.preview-row-loading {
  padding: 0.32rem 0;
  color: var(--sd-text-muted);
}

.copy-note {
  color: var(--sd-text-muted);
  font-size: 0.82rem;
  padding-bottom: 0.5rem;
}

.text-row {
  margin: 0;
  padding: 0.28rem 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
}
</style>
