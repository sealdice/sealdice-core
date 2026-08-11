<template>
  <main class="tool-test-page">
    <header class="tool-test-page__header">
      <div class="tool-test-page__heading">
        <p class="tool-test-page__eyebrow">SealDice UI</p>
        <h1>指令测试</h1>
        <p class="tool-test-page__summary">在私聊或群聊上下文中验证指令、权限和消息段表现。</p>
      </div>

      <div class="tool-test-page__controls">
        <NRadioGroup v-model:value="toolTest.mode.value" size="small">
          <NRadioButton value="private">私聊</NRadioButton>
          <NRadioButton value="group">群聊</NRadioButton>
        </NRadioGroup>
        <NSelect
          v-if="toolTest.mode.value === 'private'"
          v-model:value="toolTest.selectedSenderIds.private"
          size="small"
          class="tool-test-page__private-sender"
          :options="privateSenderOptions"
          @update:value="toolTest.loadContext('private')"
        />
        <NRadioGroup v-model:value="toolTest.splitOptionKey.value" size="small">
          <NRadioButton
            v-for="option in toolTest.splitOptions.value"
            :key="option.key"
            :value="option.key"
          >
            {{ option.label }}
          </NRadioButton>
        </NRadioGroup>
        <NTag :type="toolTest.realtimeActive.value ? 'success' : 'warning'" size="small">
          {{ toolTest.realtimeActive.value ? '实时连接' : '兼容轮询' }}
        </NTag>
        <NPopover placement="bottom-end" trigger="click">
          <template #trigger>
            <NButton secondary size="small" title="快捷操作">
              <template #icon>
                <NIcon><i-ep-more /></NIcon>
              </template>
              操作
            </NButton>
          </template>
          <div class="tool-test-page__quick-actions">
            <NButton
              secondary
              :loading="toolTest.reloadDeckMutation.isPending.value"
              @click="toolTest.reloadDeck"
            >
              重载牌堆
            </NButton>
            <NButton
              secondary
              :loading="toolTest.reloadJsMutation.isPending.value"
              @click="toolTest.reloadJs"
            >
              重载 JS
            </NButton>
            <NButton
              secondary
              :loading="toolTest.reloadHelpdocMutation.isPending.value"
              @click="toolTest.reloadHelpdoc"
            >
              重载帮助文档
            </NButton>
          </div>
        </NPopover>
      </div>
    </header>

    <NAlert v-if="toolTest.commandErrorText.value" type="warning" class="tool-test-page__alert">
      {{ toolTest.commandErrorText.value }}
    </NAlert>
    <NAlert v-if="toolTest.pollingErrorText.value" type="error" class="tool-test-page__alert">
      {{ toolTest.pollingErrorText.value }}
    </NAlert>

    <section class="tool-test-page__workspace" :class="workspaceClass">
      <div class="tool-test-page__chat-column">
        <ToolTestChatWindow
          :title="toolTest.modeTitle.value"
          :messages="toolTest.currentMessages.value"
        >
          <template #actions>
            <NButton
              v-if="toolTest.mode.value === 'group'"
              secondary
              size="small"
              class="tool-test-page__mobile-members-button"
              title="群成员"
              @click="memberRailOpen = true"
            >
              <template #icon
                ><NIcon><i-ep-user /></NIcon
              ></template>
              群成员
            </NButton>
          </template>
        </ToolTestChatWindow>

        <footer class="tool-test-page__composer">
          <ToolTestCommandComposer
            :model-value="toolTest.input.value"
            :options="toolTest.commandOptions.value"
            :prefixes="toolTest.currentContext.value.commandPrefix"
            :loading="toolTest.sendMutation.isPending.value || toolTest.commandLoading.value"
            @update:model-value="toolTest.input.value = $event"
            @submit="toolTest.send"
          />
        </footer>
      </div>

      <div
        v-if="memberRailOpen && toolTest.mode.value === 'group'"
        class="tool-test-page__mobile-scrim"
        @click="memberRailOpen = false"
      />
      <ToolTestMemberRail
        v-if="toolTest.mode.value === 'group'"
        :mode="toolTest.mode.value"
        :profiles="toolTest.currentContext.value.members"
        :current-user-id="toolTest.currentContext.value.currentSenderId"
        :group-access="toolTest.currentContext.value.groupAccess"
        :mobile-open="memberRailOpen"
        :saving="toolTest.updateProfileMutation.isPending.value"
        @select="handleSelectProfile"
        @save-profile="toolTest.updateProfile"
        @update-group-access="toolTest.updateGroupAccess"
        @close-mobile="memberRailOpen = false"
      />
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, shallowRef } from 'vue';
import {
  NAlert,
  NButton,
  NIcon,
  NPopover,
  NRadioButton,
  NRadioGroup,
  NSelect,
  NTag,
} from 'naive-ui';
import ToolTestChatWindow from '@/components/tool-test/ToolTestChatWindow.vue';
import ToolTestCommandComposer from '@/components/tool-test/ToolTestCommandComposer.vue';
import ToolTestMemberRail from '@/components/tool-test/ToolTestMemberRail.vue';
import { useToolTest } from '@/features/toolTest/useToolTest';
import type { ToolTestProfile } from '@/features/toolTest/model';

const toolTest = useToolTest();
const memberRailOpen = shallowRef(false);

const privateSenderOptions = computed(
  () =>
    toolTest.contexts.private?.members
      .filter(profile => profile.enabled)
      .map(profile => ({ label: `${profile.name} · ${profile.userId}`, value: profile.userId })) ??
    []
);

const workspaceClass = computed(() => ({
  'tool-test-page__workspace--private': toolTest.mode.value === 'private',
  'tool-test-page__workspace--group': toolTest.mode.value === 'group',
}));

async function handleSelectProfile(profile: ToolTestProfile) {
  await toolTest.selectSender(profile);
  memberRailOpen.value = false;
}
</script>

<style scoped>
.tool-test-page {
  --devui-primary: var(--sd-primary);
  --devui-primary-hover: color-mix(in srgb, var(--sd-primary), white 12%);
  --devui-primary-active: color-mix(in srgb, var(--sd-primary), black 12%);
  --devui-base-bg: var(--sd-bg-elevated);
  --devui-global-bg: var(--sd-bg-control);
  --devui-form-control-bg: var(--sd-bg-elevated);
  --devui-form-control-line: var(--sd-border-soft);
  --devui-disabled-bg: var(--sd-bg-control);
  --devui-disabled-text: var(--sd-text-muted);
  --devui-text: var(--sd-text-primary);
  --devui-light-text: var(--sd-text-inverse);
  --devui-placeholder: var(--sd-text-muted);
  --devui-dividing-line: var(--sd-border-soft);
  --devui-list-item-hover-bg: var(--sd-bg-hover);
  --devui-list-item-hover-text: var(--sd-text-primary);
  --devui-list-item-active-bg: var(--sd-bg-selected);
  --devui-list-item-active-text: var(--sd-text-primary);
  --devui-gray-form-control-bg: var(--sd-bg-control);
  --devui-connected-overlay-bg: var(--sd-bg-elevated);
  --devui-font-size: 0.875rem;
  display: flex;
  flex: 1 1 auto;
  min-height: 0;
  flex-direction: column;
  gap: 1rem;
  overflow: hidden;
}

.tool-test-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1.5rem;
}

.tool-test-page__heading {
  min-width: 0;
}

.tool-test-page__eyebrow {
  margin: 0 0 0.375rem;
  color: var(--sd-accent-strong);
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.tool-test-page h1 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 1.75rem;
  line-height: 1.2;
}

.tool-test-page__summary {
  margin: 0.5rem 0 0;
  color: var(--sd-text-secondary);
  line-height: 1.6;
}

.tool-test-page__controls {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.tool-test-page__private-sender {
  width: min(14rem, 100%);
}

.tool-test-page__quick-actions {
  display: grid;
  min-width: 12rem;
  gap: 0.4rem;
}

.tool-test-page__alert {
  margin-top: -0.25rem;
}

.tool-test-page__workspace {
  display: flex;
  flex: 1 1 0;
  height: 0;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--sd-border-soft);
  border-radius: 16px;
  background: var(--sd-bg-elevated);
}

.tool-test-page__chat-column {
  display: flex;
  height: 100%;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
}

.tool-test-page__workspace :deep(.tool-test-chat-window) {
  min-height: 0;
  flex: 1 1 auto;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.tool-test-page__workspace--private .tool-test-page__chat-column {
  width: 100%;
}

.tool-test-page__mobile-members-button {
  display: none;
}

.tool-test-page__mobile-scrim {
  display: none;
  position: fixed;
  z-index: 19;
  inset: 0;
  background: rgba(15, 23, 42, 0.34);
}

.tool-test-page__composer {
  flex-shrink: 0;
  padding: 0.65rem 0.75rem max(0.65rem, env(safe-area-inset-bottom));
  border-top: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated);
}

@media (max-width: 1100px) {
  .tool-test-page__header {
    flex-direction: column;
  }

  .tool-test-page__controls {
    justify-content: flex-start;
  }
}

@media (max-width: 860px) {
  .tool-test-page__mobile-members-button {
    display: inline-flex;
  }

  .tool-test-page__mobile-scrim {
    display: block;
  }

  .tool-test-page__workspace--group :deep(.tool-test-member-rail) {
    position: fixed;
    z-index: 20;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(19rem, 88vw);
    transform: translateX(100%);
    transition: transform 180ms ease;
  }

  .tool-test-page__workspace--group :deep(.tool-test-member-rail--open) {
    transform: translateX(0);
  }
}

@media (max-width: 640px) {
  .tool-test-page__controls {
    width: 100%;
    align-items: stretch;
  }

  .tool-test-page__private-sender {
    flex: 1 1 12rem;
  }

  .tool-test-page__workspace {
    margin-inline: -0.25rem;
    border-radius: 12px;
  }

  .tool-test-page__summary {
    font-size: 0.875rem;
  }
}
</style>
