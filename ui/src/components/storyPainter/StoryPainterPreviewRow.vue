<script setup lang="ts">
import { computed } from 'vue';
import type {
  StoryPainterChar,
  StoryPainterForumOptions,
  StoryPainterLogItem,
  StoryPainterOptions,
  StoryPainterPreviewDisplayMode,
} from '@/features/storyPainter/types';
import { renderForumText, renderPreviewHtml, renderTrgText } from '@/features/storyPainter/renderers';

const props = defineProps<{
  source: StoryPainterLogItem;
  mode: StoryPainterPreviewDisplayMode;
  chars: StoryPainterChar[];
  options: StoryPainterOptions;
  forumOptions: StoryPainterForumOptions;
  color: string;
  addVoiceMark: boolean;
}>();

const html = computed(() => renderPreviewHtml(props.source, props.chars, props.options, props.color));
const text = computed(() => props.mode === 'trg'
  ? renderTrgText(props.source, props.chars, props.options, props.addVoiceMark)
  : renderForumText(props.source, props.chars, props.options, props.forumOptions, props.color));
</script>

<template>
  <div v-if="mode === 'preview'" class="preview-row" v-html="html" />
  <pre v-else class="text-row">{{ text }}</pre>
</template>

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
</style>
