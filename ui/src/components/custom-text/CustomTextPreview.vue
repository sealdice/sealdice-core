<template>
  <div class="text-preview" role="region" aria-label="文案效果预览" aria-live="polite">
    <CustomTextPreviewInfo v-if="info" :info="info">
      <template v-if="htmlV2 !== undefined" #v2-markdown>
        <div role="region" aria-label="V2 Markdown 预览">
          <div class="markdown-output" v-html="htmlV2" />
        </div>
      </template>
      <template v-if="htmlV1 !== undefined" #v1-markdown>
        <div role="region" aria-label="V1 Markdown 预览">
          <div class="markdown-output" v-html="htmlV1" />
        </div>
      </template>
    </CustomTextPreviewInfo>
    <n-text v-else depth="3">此文案暂无预览，保存文案后即可查看效果。</n-text>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import CustomTextPreviewInfo from './CustomTextPreviewInfo.vue';
import type { TextItemCompatibleInfo } from '@/api';
import { renderMarkdownIfFormatted } from '@/features/customText/markdown';
import 'katex/dist/katex.min.css';
const props = defineProps<{ info?: TextItemCompatibleInfo }>();
const htmlV2 = computed(() =>
  props.info?.errV2 ? undefined : renderMarkdownIfFormatted(props.info?.textV2 ?? '')
);
const htmlV1 = computed(() =>
  props.info?.errV1 ? undefined : renderMarkdownIfFormatted(props.info?.textV1 ?? '')
);
</script>

<style scoped>
.text-preview {
  min-width: 0;
  overflow-wrap: anywhere;
}
.markdown-output {
  white-space: normal;
}
.markdown-output :deep(pre) {
  padding: 12px;
  background: var(--sd-bg-control);
  overflow: auto;
}
.markdown-output :deep(code) {
  font-family: monospace;
  background: var(--sd-bg-control);
}
.markdown-output :deep(blockquote) {
  border-left: 3px solid var(--sd-border);
  margin-left: 0;
  padding-left: 12px;
}
.markdown-output :deep(.qq-underline-bold) {
  text-decoration: underline;
}
.markdown-output :deep(img) {
  max-width: 100%;
  height: auto;
}
.markdown-output :deep(.katex-display) {
  overflow-x: auto;
  overflow-y: hidden;
}
</style>
