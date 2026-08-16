import { computed, onBeforeUnmount, reactive, ref, shallowRef, watch } from 'vue';
import { useMutation } from '@tanstack/vue-query';
import {
  getSdApiV2ToolTestCommands,
  getSdApiV2ToolTestContext,
  getSdApiV2ToolTestMessagesPending,
  getSdApiV2ToolTestSplitOptions,
  postSdApiV2DeckReload,
  postSdApiV2HelpdocReload,
  postSdApiV2JsReload,
  postSdApiV2ToolTestMessages,
  putSdApiV2ToolTestContext,
  putSdApiV2ToolTestProfile,
} from '@/api';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import { useRealtimeClient, subscribeRealtimeEvent } from '@/features/realtime/client';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
} from '@/features/testMode/state';
import {
  appendPendingToolTestMessages,
  appendRealtimeToolTestMessage,
  appendSelfToolTestMessage,
  buildToolTestCommandOptions,
  createInitialToolTestMessages,
  normalizeToolTestSplitOptions,
  type ToolTestContext,
  type ToolTestCommand,
  type ToolTestMessage,
  type ToolTestMode,
  type ToolTestPendingMessage,
  type ToolTestProfile,
  type ToolTestRealtimeMessage,
  type ToolTestSplitOption,
} from './model';

const POLL_INTERVAL_MS = 1500;

type ToolTestSessions = Record<ToolTestMode, ToolTestMessage[]>;
type ToolTestContexts = Record<ToolTestMode, ToolTestContext | null>;

function fallbackContext(mode: ToolTestMode): ToolTestContext {
  const senderId = mode === 'group' ? 'UI:1002' : 'UI:1001';
  return {
    mode,
    conversationId: mode === 'group' ? 'UI-Group:2001' : `PG-${senderId}`,
    groupId: mode === 'group' ? 'UI-Group:2001' : undefined,
    groupName: mode === 'group' ? 'SealDice 测试群组' : `与 ${senderId} 的私聊`,
    currentSenderId: senderId,
    members: [],
    botName: '海豹核心',
    botAvatarKey: 'seal',
    commandPrefix: ['.'],
  };
}

function normalizeContext(
  value: {
    mode?: string;
    conversationId?: string;
    groupId?: string;
    groupName?: string;
    groupAccess?: string;
    currentSenderId?: string;
    members?: ToolTestProfile[] | null;
    botName?: string;
    botAvatarKey?: string;
    commandPrefix?: string[] | null;
  },
  mode: ToolTestMode
): ToolTestContext {
  const fallback = fallbackContext(mode);
  return {
    mode,
    conversationId: value.conversationId || fallback.conversationId,
    groupId: value.groupId || fallback.groupId,
    groupName: value.groupName || fallback.groupName,
    groupAccess: value.groupAccess || 'normal',
    currentSenderId: value.currentSenderId || fallback.currentSenderId,
    members: value.members ?? [],
    botName: value.botName || fallback.botName,
    botAvatarKey: value.botAvatarKey || fallback.botAvatarKey,
    commandPrefix: value.commandPrefix?.filter(Boolean).length
      ? value.commandPrefix.filter(Boolean)
      : fallback.commandPrefix,
  };
}

export function useToolTest() {
  const mode = shallowRef<ToolTestMode>('private');
  const input = shallowRef('');
  const commandList = ref<ToolTestCommand[]>([]);
  const commandLoading = shallowRef(false);
  const commandErrorText = shallowRef('');
  const pollingErrorText = shallowRef('');
  const pollingActive = shallowRef(false);
  const splitOptions = ref<ToolTestSplitOption[]>(normalizeToolTestSplitOptions(undefined).options);
  const splitOptionKey = shallowRef(normalizeToolTestSplitOptions(undefined).defaultKey);
  const selectedSenderIds = reactive<Record<ToolTestMode, string>>({
    private: 'UI:1001',
    group: 'UI:1002',
  });
  const contexts = reactive<ToolTestContexts>({ private: null, group: null });
  const sessions = reactive<ToolTestSessions>({
    private: createInitialToolTestMessages('private'),
    group: createInitialToolTestMessages('group'),
  });

  const realtime = useRealtimeClient();
  const realtimeActive = computed(() => realtime.connected.value);
  const currentContext = computed(() => contexts[mode.value] ?? fallbackContext(mode.value));
  const commandOptions = computed(() =>
    buildToolTestCommandOptions(commandList.value, input.value, currentContext.value.commandPrefix)
  );
  const currentProfile = computed(() =>
    currentContext.value.members.find(item => item.userId === selectedSenderIds[mode.value])
  );
  const currentMessages = computed(() =>
    sessions[mode.value].map(message => ({
      ...message,
      self: !message.isBot && message.senderId === currentContext.value.currentSenderId,
    }))
  );
  const modeTitle = computed(() =>
    mode.value === 'private'
      ? `与 ${currentProfile.value?.name || selectedSenderIds.private} 的私聊`
      : currentContext.value.groupName
  );
  const selectedSplitLen = computed(
    () =>
      splitOptions.value.find(item => item.key === splitOptionKey.value)?.messageSplitLen ?? 2000
  );

  let pollTimer: number | null = null;
  let polling = false;
  let sendInFlight = false;
  let unsubscribeRealtime: (() => void) | null = null;

  const sendMutation = useMutation({
    mutationFn: async (payload: {
      text: string;
      mode: ToolTestMode;
      senderId: string;
      groupId?: string;
      messageSplitLen: number;
    }) => {
      const { data } = await postSdApiV2ToolTestMessages({
        body: payload,
        throwOnError: true,
      });
      return data.item;
    },
  });

  const updateProfileMutation = useMutation({
    mutationFn: async (payload: {
      mode: ToolTestMode;
      groupId?: string;
      userId: string;
      name: string;
      role: string;
      avatarKey: string;
      enabled: boolean;
    }) => {
      const { data } = await putSdApiV2ToolTestProfile({ body: payload, throwOnError: true });
      return normalizeContext(data.item, payload.mode);
    },
  });

  const updateContextMutation = useMutation({
    mutationFn: async (payload: { groupId: string; groupName: string; groupAccess: string }) => {
      const { data } = await putSdApiV2ToolTestContext({ body: payload, throwOnError: true });
      return normalizeContext(data.item, 'group');
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
    const context = contexts[targetMode] ?? fallbackContext(targetMode);
    sessions[targetMode] = [
      ...sessions[targetMode],
      {
        id: `tip-${targetMode}-${Date.now()}-${sessions[targetMode].length}`,
        kind: 'tip',
        mode: targetMode,
        self: false,
        content: text,
        rawContent: text,
        senderId: 'system',
        senderName: '系统',
        avatarKey: context.botAvatarKey,
        isBot: false,
        direction: 'incoming',
        segments: [],
        timestamp: Date.now(),
      },
    ];
  }

  async function loadContext(targetMode: ToolTestMode = mode.value) {
    if (!hasAccessToken.value) return;
    const senderId = selectedSenderIds[targetMode];
    try {
      const { data } = await getSdApiV2ToolTestContext({
        query: {
          mode: targetMode,
          senderId,
          groupId: targetMode === 'group' ? 'UI-Group:2001' : undefined,
        },
        throwOnError: true,
      });
      contexts[targetMode] = normalizeContext(data.item, targetMode);
    } catch (error) {
      appendTip(targetMode, getErrorMessage(error, '测试身份读取失败'));
    }
  }

  async function selectSender(profile: ToolTestProfile) {
    if (!profile.enabled) return;
    selectedSenderIds[mode.value] = profile.userId;
    await loadContext(mode.value);
  }

  async function updateProfile(profile: ToolTestProfile) {
    try {
      const context = await updateProfileMutation.mutateAsync({
        mode: mode.value,
        groupId: mode.value === 'group' ? currentContext.value.groupId : undefined,
        userId: profile.userId,
        name: profile.name,
        role: profile.role,
        avatarKey: profile.avatarKey,
        enabled: profile.enabled,
      });
      contexts[mode.value] = context;
    } catch (error) {
      appendTip(mode.value, getErrorMessage(error, '测试身份保存失败'));
    }
  }

  async function updateGroupAccess(access: string) {
    if (mode.value !== 'group') return;
    try {
      contexts.group = await updateContextMutation.mutateAsync({
        groupId: currentContext.value.groupId || 'UI-Group:2001',
        groupName: currentContext.value.groupName,
        groupAccess: access,
      });
    } catch (error) {
      appendTip('group', getErrorMessage(error, '群组测试状态保存失败'));
    }
  }

  async function loadSplitOptions() {
    if (!hasAccessToken.value) return;
    try {
      const { data } = await getSdApiV2ToolTestSplitOptions({ throwOnError: true });
      const state = normalizeToolTestSplitOptions(data.item);
      splitOptions.value = state.options;
      splitOptionKey.value = state.defaultKey;
    } catch {
      const state = normalizeToolTestSplitOptions(undefined);
      splitOptions.value = state.options;
      splitOptionKey.value = state.defaultKey;
    }
  }

  async function loadCommands() {
    if (!hasAccessToken.value) return;
    commandLoading.value = true;
    commandErrorText.value = '';
    try {
      const { data } = await getSdApiV2ToolTestCommands({
        query: {
          mode: mode.value,
          senderId: selectedSenderIds[mode.value],
          groupId: mode.value === 'group' ? currentContext.value.groupId : undefined,
        },
        throwOnError: true,
      });
      commandList.value = (data.item.items ?? []).map(item => ({
        name: item.name,
        description: item.description,
        source: item.source,
      }));
    } catch (error) {
      commandErrorText.value = getErrorMessage(error, '指令列表读取失败');
    } finally {
      commandLoading.value = false;
    }
  }

  async function pullPendingMessages() {
    if (!hasAccessToken.value || polling || realtimeActive.value) return;
    polling = true;
    try {
      const { data } = await getSdApiV2ToolTestMessagesPending({ throwOnError: true });
      const pending = (data.item.items ?? []).map(
        (item): ToolTestPendingMessage => ({
          uid: item.uid,
          message: item.message,
          messageType: item.messageType === 'group' ? 'group' : 'private',
        })
      );
      const timestamp = Date.now();
      sessions.private = appendPendingToolTestMessages(
        sessions.private,
        pending,
        'private',
        timestamp
      );
      sessions.group = appendPendingToolTestMessages(sessions.group, pending, 'group', timestamp);
      pollingErrorText.value = '';
    } catch (error) {
      pollingErrorText.value = getErrorMessage(error, '兼容消息读取失败');
      stopPolling();
    } finally {
      polling = false;
    }
  }

  function handleRealtimeMessage(event: ToolTestRealtimeMessage) {
    const targetMode = event.messageType === 'group' ? 'group' : 'private';
    const context = contexts[targetMode];
    if (!context) return;
    sessions[targetMode] = appendRealtimeToolTestMessage(sessions[targetMode], event, context);
  }

  function stopPolling() {
    if (pollTimer !== null) {
      window.clearInterval(pollTimer);
      pollTimer = null;
    }
    pollingActive.value = false;
  }

  function startPolling() {
    if (!hasAccessToken.value || realtimeActive.value || pollTimer !== null) return;
    pollTimer = window.setInterval(() => void pullPendingMessages(), POLL_INTERVAL_MS);
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
    if (!text || sendMutation.isPending.value || sendInFlight) return;
    sendInFlight = true;
    const activeMode = mode.value;
    const context = currentContext.value;
    const profile = currentProfile.value;
    if (!realtimeActive.value || !contexts[activeMode]) {
      sessions[activeMode] = appendSelfToolTestMessage(sessions[activeMode], {
        text,
        mode: activeMode,
        timestamp: Date.now(),
        profile: profile ?? {
          userId: selectedSenderIds[activeMode],
          name: activeMode === 'group' ? '群组用户' : '私聊用户',
          role: 'member',
          avatarKey: 'member',
        },
      });
    }
    input.value = '';

    try {
      await sendMutation.mutateAsync({
        text,
        mode: activeMode,
        senderId: selectedSenderIds[activeMode],
        groupId: activeMode === 'group' ? context.groupId : undefined,
        messageSplitLen: selectedSplitLen.value,
      });
      if (!realtimeActive.value) {
        startPolling();
      }
    } catch (error) {
      appendTip(activeMode, getErrorMessage(error, '发送失败'));
    } finally {
      sendInFlight = false;
    }
  }

  async function reloadDeck() {
    try {
      const item = await reloadDeckMutation.mutateAsync();
      if (isTestModeResponse(item)) {
        appendTip(mode.value, '展示模式无法重载牌堆。');
        return;
      }
      appendTip(mode.value, item.success ? '已重载牌堆。' : '牌堆重载失败。');
    } catch (error) {
      if (isTestModeApiError(error)) {
        appendTip(mode.value, getTestModeBlockMessage(error));
        return;
      }
      appendTip(mode.value, getErrorMessage(error, '牌堆重载失败'));
    }
  }

  async function reloadJs() {
    try {
      const item = await reloadJsMutation.mutateAsync();
      if (isTestModeResponse(item)) {
        appendTip(mode.value, '展示模式无法重载 JS。');
        return;
      }
      appendTip(mode.value, item.success ? '已重载 JS。' : 'JS 重载失败。');
    } catch (error) {
      if (isTestModeApiError(error)) {
        appendTip(mode.value, getTestModeBlockMessage(error));
        return;
      }
      appendTip(mode.value, getErrorMessage(error, 'JS 重载失败'));
    }
  }

  async function reloadHelpdoc() {
    try {
      const item = await reloadHelpdocMutation.mutateAsync();
      appendTip(mode.value, item.success ? '已重载帮助文档。' : item.err || '帮助文档重载失败');
    } catch (error) {
      appendTip(mode.value, getErrorMessage(error, '帮助文档重载失败'));
    }
  }

  unsubscribeRealtime = subscribeRealtimeEvent<ToolTestRealtimeMessage>(
    'tooltest/message',
    handleRealtimeMessage
  );

  watch(
    [hasAccessToken, realtimeActive],
    ([canAccess, connected]) => {
      if (!canAccess) {
        stopPolling();
        commandList.value = [];
        commandErrorText.value = '';
        pollingErrorText.value = '';
        return;
      }
      void loadCommands();
      void loadSplitOptions();
      void loadContext('private');
      void loadContext('group');
      if (connected) {
        stopPolling();
      } else {
        startPolling();
      }
    },
    { immediate: true }
  );

  watch(mode, () => {
    void loadContext(mode.value);
    void loadCommands();
  });

  onBeforeUnmount(() => {
    stopPolling();
    unsubscribeRealtime?.();
  });

  return {
    commandErrorText,
    commandLoading,
    commandOptions,
    contexts,
    currentContext,
    currentMessages,
    currentProfile,
    input,
    loadContext,
    mode,
    modeTitle,
    pollingActive,
    pollingErrorText,
    realtimeActive,
    selectedSenderIds,
    splitOptionKey,
    splitOptions,
    updateContextMutation,
    updateProfileMutation,
    reloadDeck,
    reloadDeckMutation,
    reloadHelpdoc,
    reloadHelpdocMutation,
    reloadJs,
    reloadJsMutation,
    restartPolling,
    selectSender,
    send,
    sendMutation,
    updateGroupAccess,
    updateProfile,
  };
}
