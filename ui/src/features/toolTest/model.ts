export type ToolTestMode = 'private' | 'group';

export type ToolTestMessageKind = 'message' | 'tip';

export type ToolTestMessage = {
  id: string;
  kind: ToolTestMessageKind;
  mode: ToolTestMode;
  self: boolean;
  content: string;
  senderName: string;
  isBot: boolean;
  timestamp: number;
};

export type ToolTestPendingMessage = {
  uid: string;
  message: string;
  messageType: ToolTestMode;
};

export type ToolTestCommandOption = {
  label: string;
  value: string;
};

type AppendSelfInput = {
  text: string;
  mode: ToolTestMode;
  timestamp: number;
};

const READY_MESSAGE_BY_MODE: Record<ToolTestMode, string> = {
  private: '海豹已就绪。此界面可视为私聊窗口。\n设置中添加 Master 名为 UI:1001\n即可在此界面使用 master 命令！',
  group: '海豹已就绪。此界面可视为群聊窗口。\n设置中添加 Master 名为 UI:1002\n即可在此界面使用 master 命令！',
};

const TIP_MESSAGE = '请注意，当前会话记录在刷新页面后会消失。';

function buildMessageId(prefix: string, mode: ToolTestMode, timestamp: number, index: number) {
  return `${prefix}-${mode}-${timestamp}-${index}`;
}

function normalizeCommandPrefix(input: string) {
  const trimmed = input.trim();
  if (!trimmed) return '.';

  const first = trimmed[0];
  if (/[\p{P}\p{S}]/u.test(first)) return first;
  return '.';
}

export function createInitialToolTestMessages(mode: ToolTestMode): ToolTestMessage[] {
  return [
    {
      id: buildMessageId('seed', mode, 0, 0),
      kind: 'message',
      mode,
      self: false,
      content: READY_MESSAGE_BY_MODE[mode],
      senderName: '海豹核心',
      isBot: true,
      timestamp: 0,
    },
    {
      id: buildMessageId('seed', mode, 0, 1),
      kind: 'tip',
      mode,
      self: false,
      content: TIP_MESSAGE,
      senderName: '系统',
      isBot: false,
      timestamp: 0,
    },
  ];
}

export function appendSelfToolTestMessage(
  messages: ToolTestMessage[],
  input: AppendSelfInput,
): ToolTestMessage[] {
  const text = input.text.trim();
  if (!text) return messages;

  return [
    ...messages,
    {
      id: buildMessageId('self', input.mode, input.timestamp, messages.length),
      kind: 'message',
      mode: input.mode,
      self: true,
      content: text,
      senderName: '你',
      isBot: false,
      timestamp: input.timestamp,
    },
  ];
}

export function appendPendingToolTestMessages(
  messages: ToolTestMessage[],
  pending: ToolTestPendingMessage[],
  mode: ToolTestMode,
  timestamp: number,
): ToolTestMessage[] {
  const appended = pending
    .filter(item => item.messageType === mode && item.message.trim() !== '')
    .map((item, index) => ({
      id: buildMessageId('pending', mode, timestamp, messages.length + index),
      kind: 'message' as const,
      mode,
      self: false,
      content: item.message.trim(),
      senderName: '海豹核心',
      isBot: true,
      timestamp,
    }));

  if (!appended.length) return messages;
  return [...messages, ...appended];
}

export function buildToolTestCommandOptions(
  commands: string[],
  input: string,
): ToolTestCommandOption[] {
  const prefix = normalizeCommandPrefix(input);
  const trimmedInput = input.trim();
  const normalizedQuery = trimmedInput.startsWith(prefix)
    ? trimmedInput.slice(prefix.length).toLowerCase()
    : trimmedInput.toLowerCase();

  return commands
    .map(command => command.trim())
    .filter(Boolean)
    .filter(command => normalizedQuery === '' || command.toLowerCase().startsWith(normalizedQuery))
    .map(command => ({
      label: `${prefix}${command}`,
      value: `${prefix}${command}`,
    }));
}
