<template>
  <section class="story-painter-preview">
    <div v-if="mode !== 'preview'" class="preview-toolbar">
      <n-flex size="small" align="center" wrap>
        <template v-if="mode === 'bbs' || mode === 'bbspineapple'">
          <n-checkbox
            :checked="forumOptions.bbsUseSpaceWithMultiLine"
            @update:checked="
              value => emit('updateForumOptions', { bbsUseSpaceWithMultiLine: value })
            "
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
        <n-checkbox
          v-if="mode === 'trg'"
          :checked="addVoiceMark"
          @update:checked="value => emit('updateAddVoiceMark', value)"
        >
          添加语音合成标记
        </n-checkbox>
        <n-button size="small" type="primary" secondary @click="copyPreviewText">
          <template #icon>
            <n-icon><i-tabler-copy /></n-icon>
          </template>
          复制
        </n-button>
      </n-flex>
    </div>

    <div v-if="readError" role="alert" class="preview-empty">{{ readError }}</div>
    <div v-if="items.length === 0" class="preview-empty">
      <n-empty description="没有可展示的日志条目" />
    </div>

    <div v-else ref="previewList" class="preview-list">
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
            :previous="items[virtualRow.index - 1]"
            :next="items[virtualRow.index + 1]"
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
  </section>
</template>

<script setup lang="ts">
import { computed, shallowRef, useTemplateRef, watch, type VNodeRef } from 'vue';
import { useVirtualizer } from '@tanstack/vue-virtual';
import type {
  StoryPainterChar,
  StoryPainterForumOptions,
  StoryPainterLogItem,
  StoryPainterOptions,
  StoryPainterPreviewDisplayMode,
} from '@/features/storyPainter/types';
import { packStoryPainterNameId } from '@/features/storyPainter/types';
import { buildStoryPainterPcMap } from '@/features/storyPainter/state';
import StoryPainterPreviewRow from './StoryPainterPreviewRow.vue';

const props = defineProps<{
  mode: StoryPainterPreviewDisplayMode;
  items: StoryPainterLogItem[];
  readItems: (indexes: number[]) => Promise<StoryPainterLogItem[]>;
  chars: StoryPainterChar[];
  options: StoryPainterOptions;
  forumOptions: StoryPainterForumOptions;
  addVoiceMark: boolean;
  copyText: () => Promise<string>;
}>();

const emit = defineEmits<{
  copy: [text: string];
  updateAddVoiceMark: [value: boolean];
  updateForumOptions: [value: Partial<StoryPainterForumOptions>];
}>();

const previewListRef = useTemplateRef<HTMLDivElement>('previewList');
// 只持有当前渲染窗口；正文缓存和并发读取由 useStoryPainter 管理。
const windowItems = shallowRef(new Map<number, StoryPainterLogItem>());
const readError = shallowRef('');
const pcMap = computed(() => buildStoryPainterPcMap(props.chars));
const rowVirtualizer = useVirtualizer<HTMLDivElement, HTMLDivElement>(
  computed(() => ({
    count: props.items.length,
    getScrollElement: () => previewListRef.value,
    estimateSize: () => 28,
    overscan: 20,
    getItemKey: index => props.items[index]?.index ?? index,
    useAnimationFrameWithResizeObserver: true,
  }))
);

const virtualItems = computed(() => rowVirtualizer.value.getVirtualItems());
const virtualTotalSize = computed(() => rowVirtualizer.value.getTotalSize());
const renderedRows = computed(() =>
  virtualItems.value.flatMap(virtualRow => {
    const source = props.items[virtualRow.index];
    return source ? [{ virtualRow, item: windowItems.value.get(source.index!) }] : [];
  })
);
const measureVirtualRow: VNodeRef = node => {
  if (node === null || node instanceof HTMLDivElement) rowVirtualizer.value.measureElement(node);
};

watch(
  [() => props.items, () => virtualItems.value.map(row => row.index).join(',')],
  ([items], [previousItems], onCleanup) => {
    let cancelled = false;
    const indexes = virtualItems.value
      .map(row => props.items[row.index]?.index)
      .filter((index): index is number => index !== undefined);
    const visible = new Set(indexes);
    windowItems.value =
      items === previousItems
        ? new Map([...windowItems.value].filter(([index]) => visible.has(index)))
        : new Map();
    readError.value = '';
    const timer = setTimeout(async () => {
      if (!indexes.length) return;
      try {
        const rows = await props.readItems(indexes);
        if (!cancelled) windowItems.value = new Map(rows.map(item => [item.index!, item]));
      } catch (error) {
        if (!cancelled) readError.value = error instanceof Error ? error.message : '日志读取失败';
      }
    }, 20);
    onCleanup(() => {
      cancelled = true;
      clearTimeout(timer);
    });
  },
  { immediate: true }
);

watch(
  [
    () => props.items,
    () => props.mode,
    () => props.options,
    () => props.forumOptions,
    () => props.addVoiceMark,
  ],
  () => rowVirtualizer.value.measure(),
  { deep: true, flush: 'post' }
);

function colorByItem(item: StoryPainterLogItem): string {
  return pcMap.value.get(packStoryPainterNameId(item))?.color || '#4b5563';
}

async function copyPreviewText(): Promise<void> {
  emit('copy', await props.copyText());
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
</style>
