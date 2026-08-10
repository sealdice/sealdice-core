<script setup lang="ts">
import { nextTick, ref, useTemplateRef, watch } from 'vue';
import ToolTestChatMessage from './ToolTestChatMessage.vue';
import type { ToolTestMessage } from '@/features/toolTest/model';

const props = defineProps<{
  title: string;
  messages: ToolTestMessage[];
}>();

const scrollRef = useTemplateRef<HTMLDivElement>('scrollRef');
const newMessageCount = ref(0);

function isNearBottom() {
  const element = scrollRef.value;
  if (!element) return true;
  return element.scrollHeight - element.scrollTop - element.clientHeight <= 48;
}

async function scrollToBottom() {
  await nextTick();
  const element = scrollRef.value;
  if (!element) return;
  element.scrollTo({
    top: element.scrollHeight,
    behavior: 'smooth',
  });
  newMessageCount.value = 0;
}

watch(
  () => props.messages.length,
  (messageCount, previousMessageCount) => {
    if (messageCount === previousMessageCount || isNearBottom()) {
      void scrollToBottom();
      return;
    }
    newMessageCount.value += Math.max(messageCount - previousMessageCount, 1);
  },
  { immediate: true },
);

function handleScroll() {
  if (isNearBottom()) newMessageCount.value = 0;
}
</script>

<template>
  <section class="tool-test-chat-window">
    <header class="tool-test-chat-window__header">
      <strong class="tool-test-chat-window__title">{{ props.title }}</strong>
      <div class="tool-test-chat-window__actions">
        <n-button
          v-if="newMessageCount > 0"
          size="small"
          secondary
          aria-label="跳转到最新消息"
          @click="scrollToBottom"
        >
          新消息（{{ newMessageCount }}）
        </n-button>
        <slot name="actions" />
      </div>
    </header>
    <div
      ref="scrollRef"
      class="tool-test-chat-window__scroll"
      role="log"
      aria-live="polite"
      @scroll="handleScroll"
    >
      <div class="tool-test-chat-window__messages">
        <ToolTestChatMessage
          v-for="message in props.messages"
          :key="message.id"
          :message="message"
        />
      </div>
    </div>
  </section>
</template>

<style scoped>
.tool-test-chat-window {
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--sd-border-soft);
  border-radius: 16px;
  background: var(--sd-bg-page);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--sd-bg-elevated), transparent 6%) 0%, transparent 100%),
    var(--sd-bg-page);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.08);
}

.tool-test-chat-window__header {
  display: flex;
  min-height: 3.25rem;
  align-items: center;
  gap: 0.75rem;
  padding: 0.55rem 0.75rem;
  border-bottom: 1px solid var(--sd-border-soft);
  background: color-mix(in srgb, var(--sd-bg-elevated), transparent 8%);
}

.tool-test-chat-window__title {
  min-width: 0;
  flex: 0 1 auto;
  overflow: hidden;
  color: var(--sd-text-primary);
  font-size: 0.9rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-test-chat-window__actions {
  min-width: 0;
  flex: 1 1 auto;
}

.tool-test-chat-window__scroll {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
}

.tool-test-chat-window__messages {
  display: grid;
  align-content: start;
  gap: 0.3rem;
  padding: 1rem clamp(0.75rem, 3vw, 1.5rem);
}

@media (max-width: 640px) {
  .tool-test-chat-window__header {
    align-items: stretch;
    flex-direction: column;
  }

  .tool-test-chat-window__actions {
    flex: 0 0 auto;
  }

  .tool-test-chat-window__scroll {
    min-height: 0;
  }
}
</style>
