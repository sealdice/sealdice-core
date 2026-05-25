<template>
  <main class="tool-test-page">
    <header class="tool-test-page__header">
      <div>
        <p class="tool-test-page__eyebrow">SealDice UI</p>
        <h1>指令测试</h1>
        <p class="tool-test-page__summary">模拟 UI 私聊/群聊上下文，当前通过 V2 兼容接口轮询拉取海豹回复。</p>
      </div>
      <n-flex align="center" justify="end" wrap class="tool-test-page__status">
        <NTag :type="toolTest.pollingActive.value ? 'success' : 'warning'" size="small">
          {{ toolTest.pollingActive.value ? '轮询中' : '轮询已停止' }}
        </NTag>
        <NButton size="small" secondary @click="toolTest.restartPolling">
          重新连接
        </NButton>
      </n-flex>
    </header>

    <section class="tool-test-page__toolbar">
      <n-flex align="center" justify="space-between" wrap class="tool-test-page__toolbar-inner">
        <NRadioGroup v-model:value="toolTest.mode.value" size="small">
          <NRadioButton value="private">私聊</NRadioButton>
          <NRadioButton value="group">群聊</NRadioButton>
        </NRadioGroup>

        <n-popover placement="bottom-end" trigger="click">
          <template #trigger>
            <NButton secondary>
              <template #icon>
                <NIcon><i-carbon-add-large /></NIcon>
              </template>
              快捷操作
            </NButton>
          </template>
          <n-flex vertical size="small" class="tool-test-page__quick-actions">
            <NButton secondary :loading="toolTest.reloadDeckMutation.isPending.value" @click="toolTest.reloadDeck">
              重载牌堆
            </NButton>
            <NButton secondary :loading="toolTest.reloadJsMutation.isPending.value" @click="toolTest.reloadJs">
              重载 JS
            </NButton>
            <NButton secondary :loading="toolTest.reloadHelpdocMutation.isPending.value" @click="toolTest.reloadHelpdoc">
              重载帮助文档
            </NButton>
          </n-flex>
        </n-popover>
      </n-flex>
    </section>

    <NAlert v-if="toolTest.commandErrorText.value" type="warning" class="tool-test-page__alert">
      {{ toolTest.commandErrorText.value }}
    </NAlert>
    <NAlert v-if="toolTest.pollingErrorText.value" type="error" class="tool-test-page__alert">
      {{ toolTest.pollingErrorText.value }}
    </NAlert>

    <ToolTestChatWindow
      :title="toolTest.modeTitle.value"
      :messages="toolTest.currentMessages.value"
    />

    <footer class="tool-test-page__composer">
      <NAutoComplete
        v-model:value="toolTest.input.value"
        :options="toolTest.commandOptions.value"
        :loading="toolTest.commandLoading.value"
        placeholder="来试一试，回车键发送"
        class="tool-test-page__composer-input"
        @keyup.enter="toolTest.send"
      />
      <NButton
        type="primary"
        class="tool-test-page__composer-send"
        :loading="toolTest.sendMutation.isPending.value"
        @click="toolTest.send"
      >
        发送
      </NButton>
    </footer>
  </main>
</template>

<script setup lang="ts">
import { NAlert, NAutoComplete, NButton, NIcon, NRadioButton, NRadioGroup, NTag } from 'naive-ui';
import ToolTestChatWindow from '@/components/tool-test/ToolTestChatWindow.vue';
import { useToolTest } from '@/features/toolTest/useToolTest';

const toolTest = useToolTest();
</script>

<style scoped>
.tool-test-page {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 1rem;
}

.tool-test-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.tool-test-page__eyebrow {
  margin: 0 0 0.375rem;
  color: var(--sd-accent-strong);
  font-size: 0.75rem;
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
  line-height: 1.7;
}

.tool-test-page__status {
  flex-shrink: 0;
}

.tool-test-page__toolbar-inner {
  width: 100%;
}

.tool-test-page__quick-actions {
  min-width: 13rem;
}

.tool-test-page__alert {
  margin-top: -0.25rem;
}

.tool-test-page__composer {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 0 max(0.75rem, env(safe-area-inset-bottom));
}

.tool-test-page__composer-input {
  min-width: 0;
  flex: 1 1 auto;
}

.tool-test-page__composer-send {
  min-width: 5rem;
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .tool-test-page__header {
    flex-direction: column;
  }

  .tool-test-page__status {
    width: 100%;
    justify-content: flex-start;
  }

  .tool-test-page__composer {
    gap: 0.5rem;
    padding-bottom: max(0.75rem, env(safe-area-inset-bottom));
  }

  .tool-test-page__composer-send {
    min-width: 4.5rem;
  }
}
</style>
