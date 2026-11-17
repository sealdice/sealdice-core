<template>
  <div
    v-if="props.message.kind === 'tip'"
    class="tool-test-chat-message tool-test-chat-message--tip"
  >
    <McBubble variant="none">
      <span class="tool-test-chat-message__tip">{{ props.message.content }}</span>
    </McBubble>
  </div>
  <article
    v-else
    class="tool-test-chat-message"
    :class="{
      'tool-test-chat-message--self': props.message.self,
      'tool-test-chat-message--guest': !props.message.self,
    }"
  >
    <div class="tool-test-chat-message__avatar">
      <NAvatar
        round
        :size="36"
        :src="avatarDataUrl(props.message.avatarKey, props.message.senderName)"
        :alt="props.message.senderName"
      />
    </div>
    <div class="tool-test-chat-message__meta">
      <strong>{{ props.message.senderName }}</strong>
      <span
        v-if="roleMeta"
        class="tool-test-chat-message__role"
        :class="`tool-test-chat-message__role--${roleMeta.color}`"
      >
        {{ roleMeta.label }}
      </span>
    </div>
    <div class="tool-test-chat-message__body">
      <McBubble variant="filled" class="tool-test-chat-message__bubble">
        <ToolTestMessageContent :message="props.message" :show-raw="showRaw" />
      </McBubble>
      <div class="tool-test-chat-message__actions">
        <NTooltip>
          <template #trigger>
            <NButton
              quaternary
              circle
              size="tiny"
              class="tool-test-chat-message__toggle"
              @click.stop="showRaw = !showRaw"
            >
              <template #icon>
                <NIcon><i-tabler-file-text v-if="showRaw" /><i-tabler-eye v-else /></NIcon>
              </template>
            </NButton>
          </template>
          {{ showRaw ? '显示渲染效果' : '显示原始内容' }}
        </NTooltip>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, shallowRef } from 'vue';
import { McBubble } from '@matechat/core/Bubble';
import { NAvatar, NButton, NIcon, NTooltip } from 'naive-ui';
import ToolTestMessageContent from './ToolTestMessageContent.vue';
import { avatarDataUrl, type ToolTestMessage } from '@/features/toolTest/model';

const props = defineProps<{
  message: ToolTestMessage;
}>();

const showRaw = shallowRef(false);
const roleMeta = computed(() => {
  if (props.message.isBot) return { label: '骰子', color: 'primary' };
  switch (props.message.senderRole) {
    case 'owner':
      return { label: '群主', color: 'neutral' };
    case 'admin':
      return { label: '管理员', color: 'neutral' };
    case 'inviter':
      return { label: '邀请人', color: 'neutral' };
    case 'master':
      return { label: '骰主', color: 'neutral' };
    case 'blacklisted':
      return { label: '黑名单', color: 'error' };
    default:
      return undefined;
  }
});
</script>

<style scoped>
.tool-test-chat-message {
  display: grid;
  grid-template-areas:
    'avatar . meta'
    'avatar . body';
  grid-template-columns: 2.25rem 0.5rem minmax(0, 1fr);
  align-items: start;
  width: 100%;
  padding-top: 0.8rem;
}

.tool-test-chat-message--self {
  grid-template-areas:
    'meta . avatar'
    'body . avatar';
  grid-template-columns: minmax(0, 1fr) 0.5rem 2.25rem;
  justify-items: end;
}

.tool-test-chat-message__avatar {
  grid-area: avatar;
  align-self: start;
}

.tool-test-chat-message__meta {
  display: inline-flex;
  width: fit-content;
  min-width: 0;
  grid-area: meta;
  justify-self: start;
  align-items: center;
  gap: 0.35rem;
  min-height: 1.2rem;
  color: var(--sd-text-secondary);
  font-size: 0.78rem;
  line-height: 1.2;
}

.tool-test-chat-message--self .tool-test-chat-message__meta {
  justify-self: end;
}

.tool-test-chat-message__meta strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-test-chat-message__role {
  flex: 0 0 auto;
  padding: 0.08rem 0.3rem;
  border-radius: var(--sd-radius-xs);
  font-size: 0.68rem;
  line-height: 1.2;
  color: var(--sd-text-secondary);
  background: var(--sd-bg-control);
}

.tool-test-chat-message__role--primary {
  color: var(--sd-primary);
  background: var(--sd-primary-soft);
}

.tool-test-chat-message__role--error {
  color: var(--sd-error);
  background: var(--sd-error-soft);
}

.tool-test-chat-message__body {
  display: flex;
  width: fit-content;
  min-width: 0;
  grid-area: body;
  flex-direction: column;
  align-items: flex-start;
  max-width: min(42rem, 100%);
}

.tool-test-chat-message--self .tool-test-chat-message__body {
  justify-self: end;
  align-items: flex-end;
}

.tool-test-chat-message__bubble {
  max-width: 100%;
}

.tool-test-chat-message__bubble :deep(.mc-bubble-content) {
  min-width: 2rem;
  max-width: 100%;
  color: var(--sd-text-primary);
  background: var(--sd-bg-elevated);
}

.tool-test-chat-message--self .tool-test-chat-message__bubble :deep(.mc-bubble-content) {
  color: var(--sd-text-inverse);
  background: var(--sd-primary);
}

.tool-test-chat-message__actions {
  display: flex;
  min-height: 1.15rem;
  align-items: center;
  justify-content: flex-start;
}

.tool-test-chat-message--self .tool-test-chat-message__actions {
  justify-content: flex-end;
}

.tool-test-chat-message__toggle {
  width: 1.15rem;
  height: 1.15rem;
  color: var(--sd-text-muted);
  opacity: 0.75;
}

.tool-test-chat-message__toggle:hover {
  color: var(--sd-text-primary);
  opacity: 1;
}

.tool-test-chat-message--tip {
  display: flex;
  justify-content: center;
  padding-top: 0.7rem;
}

.tool-test-chat-message__tip {
  padding: 0.3rem 0.7rem;
  border-radius: var(--sd-radius-pill);
  color: var(--sd-text-muted);
  background: var(--sd-bg-control);
  font-size: 0.75rem;
}

@media (max-width: 640px) {
  .tool-test-chat-message {
    grid-template-columns: 2rem 0.4rem minmax(0, 1fr);
  }

  .tool-test-chat-message--self {
    grid-template-columns: minmax(0, 1fr) 0.4rem 2rem;
  }
}
</style>
