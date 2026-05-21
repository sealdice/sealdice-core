<script setup lang="ts">
import { nextTick, useTemplateRef, watch } from 'vue';
import { QHeader, QMain } from 'fake-qq-ui';
import ToolTestChatMessage from './ToolTestChatMessage.vue';
import type { ToolTestMessage } from '@/features/toolTest/model';

const props = defineProps<{
  title: string;
  messages: ToolTestMessage[];
}>();

const scrollRef = useTemplateRef<HTMLDivElement>('scrollRef');

async function scrollToBottom() {
  await nextTick();
  const element = scrollRef.value;
  if (!element) return;
  element.scrollTo({
    top: element.scrollHeight,
    behavior: 'smooth',
  });
}

watch(
  () => props.messages.length,
  () => {
    void scrollToBottom();
  },
  { immediate: true },
);
</script>

<template>
  <section class="tool-test-chat-window">
    <QHeader>{{ props.title }}</QHeader>
    <div ref="scrollRef" class="tool-test-chat-window__scroll">
      <QMain>
        <ToolTestChatMessage
          v-for="message in props.messages"
          :key="message.id"
          :message="message"
        />
      </QMain>
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
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--sd-bg-elevated), transparent 6%) 0%, transparent 100%),
    var(--qq-background-03);
  box-shadow: 0 18px 40px rgba(15, 23, 42, 0.08);
}

.tool-test-chat-window__scroll {
  min-height: 420px;
  flex: 1 1 auto;
  overflow-y: auto;
}

@media (max-width: 640px) {
  .tool-test-chat-window__scroll {
    min-height: 58vh;
  }
}
</style>
