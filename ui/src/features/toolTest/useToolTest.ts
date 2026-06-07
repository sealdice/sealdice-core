import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue';
import { useMutation } from '@tanstack/vue-query';
import {
  getSdApiV2ToolTestCommands,
  getSdApiV2ToolTestMessagesPending,
  postSdApiV2DeckReload,
  postSdApiV2HelpdocReload,
  postSdApiV2JsReload,
  postSdApiV2ToolTestMessages,
} from '@/api';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import {
  appendPendingToolTestMessages,
  appendSelfToolTestMessage,
  buildToolTestCommandOptions,
  createInitialToolTestMessages,
  type ToolTestMessage,
  type ToolTestMode,
} from './model';

const POLL_INTERVAL_MS = 1000;

type ToolTestSessions = Record<ToolTestMode, ToolTestMessage[]>;

export function useToolTest() {
  const mode = ref<ToolTestMode>('private');
  const input = ref('');
  const commandList = ref<string[]>([]);
  const commandLoading = ref(false);
  const commandErrorText = ref('');
  const pollingErrorText = ref('');
  const pollingActive = ref(false);
  const sessions = reactive<ToolTestSessions>({
    private: createInitialToolTestMessages('private'),
    group: createInitialToolTestMessages('group'),
  });

  const commandOptions = computed(() => buildToolTestCommandOptions(commandList.value, input.value));
  const currentMessages = computed(() => sessions[mode.value]);
  const modeTitle = computed(() => (mode.value === 'private' ? '私聊测试窗口' : '群聊测试窗口'));

  let pollTimer: number | null = null;
  let polling = false;

  const sendMutation = useMutation({
    mutationFn: async (payload: { text: string; mode: ToolTestMode }) => {
      const { data } = await postSdApiV2ToolTestMessages({
        body: payload,
        throwOnError: true,
      });
      return data.item;
    },
  });

  const reloadDeckMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2DeckReload({ throwOnError: true });
      return data.item;
    },
  });

  const reloadJsMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2JsReload({ throwOnError: true });
      return data.item;
    },
  });

  const reloadHelpdocMutation = useMutation({
    mutationFn: async () => {
      const { data } = await postSdApiV2HelpdocReload({ throwOnError: true });
      return data.item;
    },
  });

  function appendTip(targetMode: ToolTestMode, content: string) {
    const text = content.trim();
    if (!text) return;
    sessions[targetMode] = [
      ...sessions[targetMode],
      {
        id: `tip-${targetMode}-${Date.now()}-${sessions[targetMode].length}`,
        kind: 'tip',
        mode: targetMode,
        self: false,
        content: text,
        senderName: '系统',
        isBot: false,
        timestamp: Date.now(),
      },
    ];
  }

  async function loadCommands() {
    if (!hasAccessToken.value) return;
    commandLoading.value = true;
    commandErrorText.value = '';
    try {
      const { data } = await getSdApiV2ToolTestCommands({ throwOnError: true });
      commandList.value = data.item.items ?? [];
    } catch (error) {
      commandErrorText.value = getErrorMessage(error, '指令列表读取失败');
    } finally {
      commandLoading.value = false;
    }
  }

  async function pullPendingMessages() {
    if (!hasAccessToken.value || polling) return;
    polling = true;

    try {
      const { data } = await getSdApiV2ToolTestMessagesPending({ throwOnError: true });
      const pending = (data.item.items ?? []).map((item): {
        uid: string;
        message: string;
        messageType: ToolTestMode;
      } => ({
        uid: item.uid,
        message: item.message,
        messageType: item.messageType === 'group' ? 'group' : 'private',
      }));
      const timestamp = Date.now();
      sessions.private = appendPendingToolTestMessages(sessions.private, pending, 'private', timestamp);
      sessions.group = appendPendingToolTestMessages(sessions.group, pending, 'group', timestamp);
      pollingErrorText.value = '';
    } catch (error) {
      pollingErrorText.value = getErrorMessage(error, '指令测试消息读取失败');
      stopPolling();
    } finally {
      polling = false;
    }
  }

  function stopPolling() {
    if (pollTimer !== null) {
      window.clearInterval(pollTimer);
      pollTimer = null;
    }
    pollingActive.value = false;
  }

  function startPolling() {
    if (!hasAccessToken.value || pollTimer !== null) return;
    pollTimer = window.setInterval(() => {
      void pullPendingMessages();
    }, POLL_INTERVAL_MS);
    pollingActive.value = true;
    pollingErrorText.value = '';
    void pullPendingMessages();
  }

  function restartPolling() {
    stopPolling();
    startPolling();
  }

  async function send() {
    const text = input.value.trim();
    if (!text || sendMutation.isPending.value) return;

    const activeMode = mode.value;
    sessions[activeMode] = appendSelfToolTestMessage(sessions[activeMode], {
      text,
      mode: activeMode,
      timestamp: Date.now(),
    });
    input.value = '';

    try {
      await sendMutation.mutateAsync({
        text,
        mode: activeMode,
      });
      if (!pollingActive.value) {
        startPolling();
      } else {
        await pullPendingMessages();
      }
    } catch (error) {
      appendTip(activeMode, getErrorMessage(error, '发送失败'));
    }
  }

  async function reloadDeck() {
    try {
      const item = await reloadDeckMutation.mutateAsync();
      if (item.testMode) {
        appendTip(mode.value, '展示模式无法重载牌堆。');
        return;
      }
      appendTip(mode.value, item.success ? '已重载牌堆。' : '牌堆重载失败。');
    } catch (error) {
      appendTip(mode.value, getErrorMessage(error, '牌堆重载失败'));
    }
  }

  async function reloadJs() {
    try {
      const item = await reloadJsMutation.mutateAsync();
      if (item.testMode) {
        appendTip(mode.value, '展示模式无法重载 JS。');
        return;
      }
      appendTip(mode.value, item.success ? '已重载 JS。' : 'JS 重载失败。');
    } catch (error) {
      appendTip(mode.value, getErrorMessage(error, 'JS 重载失败'));
    }
  }

  async function reloadHelpdoc() {
    try {
      const item = await reloadHelpdocMutation.mutateAsync();
      appendTip(mode.value, item.success ? '已重载帮助文档。' : item.err || '帮助文档重载失败。');
    } catch (error) {
      appendTip(mode.value, getErrorMessage(error, '帮助文档重载失败'));
    }
  }

  watch(
    hasAccessToken,
    (canAccess) => {
      if (canAccess) {
        void loadCommands();
        startPolling();
        return;
      }
      stopPolling();
      commandList.value = [];
      commandErrorText.value = '';
      pollingErrorText.value = '';
    },
    { immediate: true },
  );

  onBeforeUnmount(() => {
    stopPolling();
  });

  return {
    commandErrorText,
    commandLoading,
    commandOptions,
    currentMessages,
    input,
    mode,
    modeTitle,
    pollingActive,
    pollingErrorText,
    reloadDeck,
    reloadDeckMutation,
    reloadHelpdoc,
    reloadHelpdocMutation,
    reloadJs,
    reloadJsMutation,
    restartPolling,
    send,
    sendMutation,
  };
}
