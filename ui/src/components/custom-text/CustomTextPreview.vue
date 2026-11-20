<template>
  <div class="text-preview" role="region" aria-label="文案效果预览" aria-live="polite">
    <CustomTextPreviewInfo v-if="info" :info="info">
      <template v-if="previewV2.html !== undefined || previewV2.failed" #v2-markdown>
        <div role="region" aria-label="V2 Markdown 预览">
          <n-text v-if="previewV2.failed" type="warning" role="status">
            Markdown 渲染失败，请查看上方原文。
          </n-text>
          <div v-else class="markdown-output" @click="loadPreviewImage" v-html="previewV2.html" />
        </div>
      </template>
      <template v-if="previewV1.html !== undefined || previewV1.failed" #v1-markdown>
        <div role="region" aria-label="V1 Markdown 预览">
          <n-text v-if="previewV1.failed" type="warning" role="status">
            Markdown 渲染失败，请查看上方原文。
          </n-text>
          <div v-else class="markdown-output" @click="loadPreviewImage" v-html="previewV1.html" />
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
interface MarkdownPreview {
  html?: string;
  failed?: boolean;
}
function markdownPreview(text: string, evaluationError: boolean): MarkdownPreview {
  if (evaluationError) return {};
  try {
    return { html: renderMarkdownIfFormatted(text) };
  } catch {
    return { failed: true };
  }
}
const previewV2 = computed(() =>
  markdownPreview(props.info?.textV2 ?? '', Boolean(props.info?.errV2))
);
const previewV1 = computed(() =>
  markdownPreview(props.info?.textV1 ?? '', Boolean(props.info?.errV1))
);

function loadPreviewImage(event: MouseEvent) {
  if (!(event.target instanceof Element) || !(event.currentTarget instanceof Element)) return;
  const placeholder = event.target.closest<HTMLButtonElement>('button.image-placeholder');
  if (!placeholder || !event.currentTarget.contains(placeholder) || placeholder.disabled) return;
  event.preventDefault();
  event.stopPropagation();
  const source = placeholder.dataset.imageSrc;
  let url: URL;
  try {
    if (!source) throw new Error('Empty image URL');
    url = new URL(source, window.location.href);
    if (url.protocol !== 'https:' && url.protocol !== 'http:')
      throw new Error('Unsupported image URL');
  } catch {
    placeholder.disabled = true;
    placeholder.textContent = '此图片地址不支持预览';
    return;
  }
  const image = document.createElement('img');
  image.alt = placeholder.dataset.imageAlt ?? '';
  image.referrerPolicy = 'no-referrer';
  image.decoding = 'async';
  const width = Number(placeholder.dataset.imageWidth);
  const height = Number(placeholder.dataset.imageHeight);
  if (Number.isSafeInteger(width) && width > 0 && Number.isSafeInteger(height) && height > 0) {
    image.width = width;
    image.height = height;
    image.style.width = `${width}px`;
    image.style.aspectRatio = `${width}/${height}`;
  }
  placeholder.disabled = true;
  placeholder.textContent = '正在加载图片…';
  image.onload = () => placeholder.replaceWith(image);
  image.onerror = () => {
    placeholder.disabled = false;
    placeholder.textContent = '图片加载失败，点击重试';
  };
  image.src = url.href;
}
</script>

<style scoped>
.text-preview {
  min-width: 0;
  overflow-wrap: anywhere;
}
.text-preview :deep(.n-descriptions-table) {
  table-layout: fixed;
}
.text-preview :deep(.n-descriptions-table-content__content) {
  max-width: 100%;
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
.markdown-output :deep(.image-placeholder) {
  display: inline-flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 4px;
  box-sizing: border-box;
  width: 240px;
  min-height: 64px;
  max-width: 100%;
  padding: 8px;
  border: 1px dashed var(--sd-border);
  border-radius: 4px;
  background: var(--sd-bg-control);
  color: var(--sd-text-secondary);
  font: inherit;
  overflow-wrap: anywhere;
  cursor: pointer;
}
.markdown-output :deep(.image-placeholder:focus-visible) {
  outline: 2px solid var(--sd-primary);
  outline-offset: 2px;
}
.markdown-output :deep(.image-placeholder:disabled) {
  cursor: progress;
}
.markdown-output :deep(.katex-display) {
  overflow-x: auto;
  overflow-y: hidden;
  padding: 0.5em 0.3em;
}
.markdown-output :deep(.katex-display > .katex) {
  width: max-content;
  min-width: 100%;
}
</style>
