<template>
  <div v-if="mode === 'preview'" class="preview-row" v-html="html" />
  <pre v-else class="text-row" :class="{ 'pineapple-row': mode === 'bbspineapple' }">{{
    text
  }}</pre>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type {
  StoryPainterChar,
  StoryPainterForumOptions,
  StoryPainterLogItem,
  StoryPainterOptions,
  StoryPainterPreviewDisplayMode,
} from '@/features/storyPainter/types';
import {
  renderForumText,
  renderPineappleForumText,
  renderPreviewHtml,
  renderTrgText,
} from '@/features/storyPainter/renderers';

const props = defineProps<{
  source: StoryPainterLogItem;
  previous?: StoryPainterLogItem;
  next?: StoryPainterLogItem;
  mode: StoryPainterPreviewDisplayMode;
  chars: StoryPainterChar[];
  options: StoryPainterOptions;
  forumOptions: StoryPainterForumOptions;
  color: string;
  addVoiceMark: boolean;
}>();

const html = computed(() =>
  renderPreviewHtml(props.source, props.chars, props.options, props.color)
);
const text = computed(() => {
  if (props.mode === 'trg')
    return renderTrgText(props.source, props.chars, props.options, props.addVoiceMark);
  if (props.mode === 'bbspineapple')
    return renderPineappleForumText(
      props.source,
      props.chars,
      props.options,
      props.forumOptions,
      props.color,
      props.previous,
      props.next
    );
  return renderForumText(props.source, props.chars, props.options, props.forumOptions, props.color);
});
</script>

<style scoped>
.preview-row {
  padding: 0.32rem 0;
  overflow-wrap: anywhere;
}

.text-row {
  margin: 0;
  padding: 0.28rem 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
}

.pineapple-row {
  padding: 0;
}
</style>
