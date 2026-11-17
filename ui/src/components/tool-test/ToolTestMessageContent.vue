<template>
  <div class="tool-test-message-content">
    <template v-if="props.showRaw">
      <pre class="tool-test-message-content__raw">{{ props.message.rawContent }}</pre>
    </template>
    <template v-else>
      <template
        v-for="(segment, index) in props.message.segments"
        :key="`${props.message.id}-segment-${index}`"
      >
        <span v-if="segment.type === 'text'">{{ segment.text }}</span>
        <span v-else-if="segment.type === 'at'" class="tool-test-message-content__token"
          >@{{ segment.data?.target || '未知用户' }}</span
        >
        <img
          v-else-if="segment.type === 'image' && mediaSource(segment)"
          class="tool-test-message-content__image"
          :src="mediaSource(segment)"
          :alt="segment.data?.file || '图片消息'"
          loading="lazy"
        />
        <a
          v-else-if="segment.type === 'file' && mediaSource(segment)"
          class="tool-test-message-content__file"
          :href="mediaSource(segment)"
          target="_blank"
          rel="noreferrer"
        >
          <span class="tool-test-message-content__file-icon">文件</span>
          <span>{{ segment.data?.file || '文件消息' }}</span>
        </a>
        <span v-else-if="segment.type === 'file'" class="tool-test-message-content__file">
          <span class="tool-test-message-content__file-icon">文件</span>
          <span>{{ segment.data?.file || '文件消息' }}</span>
        </span>
        <audio
          v-else-if="segment.type === 'record' && mediaSource(segment)"
          class="tool-test-message-content__audio"
          :src="mediaSource(segment)"
          controls
          preload="none"
        />
        <span v-else-if="segment.type === 'reply'" class="tool-test-message-content__reply">
          <small>回复 {{ segment.data?.id || '未知消息' }}</small>
          <span>回复消息</span>
        </span>
        <span v-else-if="segment.type === 'face'" class="tool-test-message-content__face"
          >表情 {{ segment.data?.id || '?' }}</span
        >
        <span v-else-if="segment.type === 'poke'" class="tool-test-message-content__token"
          >戳一戳 {{ segment.data?.target || '' }}</span
        >
        <span v-else class="tool-test-message-content__unsupported">[{{ segment.type }}]</span>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import {
  safeMediaSource,
  type ToolTestMessage,
  type ToolTestSegment,
} from '@/features/toolTest/model';

const props = defineProps<{
  message: ToolTestMessage;
  showRaw: boolean;
}>();

function mediaSource(segment: ToolTestSegment) {
  return safeMediaSource(segment.data?.url || segment.data?.file);
}
</script>

<style scoped>
.tool-test-message-content {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.tool-test-message-content__raw {
  margin: 0;
  color: inherit;
  font-family: var(--sd-font-code);
  font-size: 0.82rem;
}

.tool-test-message-content__image {
  display: block;
  max-width: min(18rem, 100%);
  max-height: 14rem;
  border-radius: var(--sd-radius-md);
  object-fit: contain;
}

.tool-test-message-content__file,
.tool-test-message-content__reply {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  max-width: 100%;
  padding: 0.5rem 0.65rem;
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-sm);
  color: var(--sd-text-primary);
  text-decoration: none;
}

.tool-test-message-content__file > span:last-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-test-message-content__file-icon {
  color: var(--sd-primary);
  font-size: 0.72rem;
  font-weight: 700;
}

.tool-test-message-content__audio {
  display: block;
  width: min(18rem, 100%);
  height: 2.25rem;
}

.tool-test-message-content__reply {
  flex-direction: column;
  align-items: flex-start;
  gap: 0.1rem;
  border-left: 3px solid var(--sd-primary);
}

.tool-test-message-content__reply small {
  color: var(--sd-text-muted);
}

.tool-test-message-content__token,
.tool-test-message-content__face,
.tool-test-message-content__unsupported {
  display: inline-block;
  margin: 0 0.1rem;
  padding: 0.1rem 0.35rem;
  border-radius: var(--sd-radius-xs);
  background: color-mix(in srgb, var(--sd-accent-strong), transparent 88%);
  color: var(--sd-accent-strong);
  font-size: 0.85em;
}

.tool-test-message-content__unsupported {
  background: color-mix(in srgb, var(--sd-text-muted), transparent 86%);
  color: var(--sd-text-muted);
}
</style>
