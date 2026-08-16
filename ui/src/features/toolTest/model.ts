export type ToolTestMode = 'private' | 'group';

export type ToolTestMessageKind = 'message' | 'tip';

export type ToolTestSegment = {
  type: string;
  text?: string;
  data?: Record<string, string>;
};

export type ToolTestProfile = {
  userId: string;
  name: string;
  role: string;
  avatarKey: string;
  enabled: boolean;
  isBot: boolean;
};

export type ToolTestContext = {
  mode: ToolTestMode;
  conversationId: string;
  groupId?: string;
  groupName: string;
  groupAccess?: string;
  currentSenderId: string;
  members: ToolTestProfile[];
  botName: string;
  botAvatarKey: string;
  commandPrefix: string[];
};

export type ToolTestRealtimeMessage = {
  id: string;
  messageType: ToolTestMode;
  conversationId: string;
  groupId?: string;
  groupName?: string;
  sender: {
    userId: string;
    nickname: string;
    isRobot?: boolean;
  };
  senderRole?: string;
  avatarKey?: string;
  isBot: boolean;
  direction: 'incoming' | 'outgoing';
  rawMessage: string;
  segments: ToolTestSegment[];
  timestamp: number;
  splitIndex?: number;
  splitCount?: number;
};

export type ToolTestMessage = {
  id: string;
  kind: ToolTestMessageKind;
  mode: ToolTestMode;
  self: boolean;
  content: string;
  rawContent: string;
  senderId: string;
  senderName: string;
  senderRole?: string;
  avatarKey: string;
  isBot: boolean;
  direction: 'incoming' | 'outgoing';
  segments: ToolTestSegment[];
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
  description?: string;
  source?: string;
};

export type ToolTestCommand = {
  name: string;
  description?: string;
  source?: string;
};

export type ToolTestSplitOption = {
  key: string;
  label: string;
  messageSplitLen: number;
};

export type ToolTestSplitOptionsState = {
  defaultKey: string;
  options: ToolTestSplitOption[];
};

type AppendSelfInput = {
  text: string;
  mode: ToolTestMode;
  timestamp: number;
  profile?: Pick<ToolTestProfile, 'userId' | 'name' | 'role' | 'avatarKey'>;
};

const READY_MESSAGE_BY_MODE: Record<ToolTestMode, string> = {
  private: '海豹已就绪。此界面可视为私聊窗口。',
  group: '海豹已就绪。此界面可视为群聊窗口。',
};

const TIP_MESSAGE = '测试身份和消息记录由当前服务端上下文提供。';

const DEFAULT_SPLIT_OPTIONS: ToolTestSplitOption[] = [
  { key: 'short', label: '短分段 300', messageSplitLen: 300 },
  { key: 'qq', label: 'QQ 分段 2000', messageSplitLen: 2000 },
  { key: 'unlimited', label: '无限', messageSplitLen: 0 },
];

const AVATAR_COLORS: Record<string, [string, string]> = {
  seal: ['#185c8b', '#58a6c8'],
  owner: ['#943f5b', '#e59d80'],
  admin: ['#4a5d93', '#9ba8da'],
  inviter: ['#9a6a28', '#e6bd75'],
  master: ['#586048', '#b1bf86'],
  member: ['#3e7184', '#81b5c4'],
  'member-2': ['#754e7b', '#b98ac3'],
  blacklisted: ['#5e5962', '#a9a4ad'],
};

function buildMessageId(prefix: string, mode: ToolTestMode, timestamp: number, index: number) {
  return `${prefix}-${mode}-${timestamp}-${index}`;
}

function textSegments(content: string): ToolTestSegment[] {
  return [{ type: 'text', text: content }];
}

export function avatarDataUrl(avatarKey: string, name: string): string {
  const [start, end] = AVATAR_COLORS[avatarKey] ?? AVATAR_COLORS.member;
  const initials = Array.from(name.trim()).slice(0, 2).join('') || '?';
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 96 96"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop stop-color="${start}"/><stop offset="1" stop-color="${end}"/></linearGradient></defs><circle cx="48" cy="48" r="48" fill="url(#g)"/><circle cx="48" cy="36" r="16" fill="rgba(255,255,255,.9)"/><path d="M20 84c4-18 16-27 28-27s24 9 28 27" fill="rgba(255,255,255,.9)"/><text x="48" y="92" text-anchor="middle" font-family="sans-serif" font-size="12" fill="rgba(0,0,0,.5)">${initials}</text></svg>`;
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`;
}

export function safeMediaSource(value: string | undefined): string | undefined {
  const trimmed = value?.trim();
  if (!trimmed) return undefined;
  if (
    trimmed.startsWith('data:image/') ||
    trimmed.startsWith('data:audio/') ||
    trimmed.startsWith('https://') ||
    trimmed.startsWith('http://')
  ) {
    return trimmed;
  }
  return undefined;
}

export function createInitialToolTestMessages(mode: ToolTestMode): ToolTestMessage[] {
  return [
    {
      id: buildMessageId('seed', mode, 0, 0),
      kind: 'message',
      mode,
      self: false,
      content: READY_MESSAGE_BY_MODE[mode],
      rawContent: READY_MESSAGE_BY_MODE[mode],
      senderId: 'UI:1000',
      senderName: '海豹核心',
      avatarKey: 'seal',
      isBot: true,
      direction: 'outgoing',
      segments: textSegments(READY_MESSAGE_BY_MODE[mode]),
      timestamp: 0,
    },
    {
      id: buildMessageId('seed', mode, 0, 1),
      kind: 'tip',
      mode,
      self: false,
      content: TIP_MESSAGE,
      rawContent: TIP_MESSAGE,
      senderId: 'system',
      senderName: '系统',
      avatarKey: 'seal',
      isBot: false,
      direction: 'incoming',
      segments: [],
      timestamp: 0,
    },
  ];
}

export function appendSelfToolTestMessage(
  messages: ToolTestMessage[],
  input: AppendSelfInput
): ToolTestMessage[] {
  const text = input.text.trim();
  if (!text) return messages;
  const profile = input.profile ?? {
    userId: input.mode === 'group' ? 'UI:1002' : 'UI:1001',
    name: '测试用户',
    role: 'member',
    avatarKey: 'member',
  };

  return [
    ...messages,
    {
      id: buildMessageId('self', input.mode, input.timestamp, messages.length),
      kind: 'message',
      mode: input.mode,
      self: true,
      content: text,
      rawContent: text,
      senderId: profile.userId,
      senderName: profile.name,
      senderRole: profile.role,
      avatarKey: profile.avatarKey,
      isBot: false,
      direction: 'incoming',
      segments: textSegments(text),
      timestamp: input.timestamp,
    },
  ];
}

export function appendRealtimeToolTestMessage(
  messages: ToolTestMessage[],
  event: ToolTestRealtimeMessage,
  context: ToolTestContext
): ToolTestMessage[] {
  if (
    !event.rawMessage.trim() ||
    event.messageType !== context.mode ||
    event.conversationId !== context.conversationId
  ) {
    return messages;
  }

  const existing = messages.find(message => message.id === event.id);
  if (existing) {
    return messages.map(message =>
      message.id === existing.id ? normalizeRealtimeMessage(event, context) : message
    );
  }
  return [...messages, normalizeRealtimeMessage(event, context)];
}

function normalizeRealtimeMessage(
  event: ToolTestRealtimeMessage,
  context: ToolTestContext
): ToolTestMessage {
  const senderName =
    event.sender.nickname.trim() || (event.isBot ? context.botName : event.sender.userId);

  return {
    id: event.id,
    kind: 'message',
    mode: event.messageType,
    self: !event.isBot && event.sender.userId === context.currentSenderId,
    content: event.rawMessage,
    rawContent: event.rawMessage,
    senderId: event.sender.userId,
    senderName,
    senderRole: event.senderRole,
    avatarKey: event.avatarKey || (event.isBot ? context.botAvatarKey : 'member'),
    isBot: event.isBot,
    direction: event.direction,
    segments: event.segments?.length ? event.segments : textSegments(event.rawMessage),
    timestamp: event.timestamp,
  };
}

export function appendPendingToolTestMessages(
  messages: ToolTestMessage[],
  pending: ToolTestPendingMessage[],
  mode: ToolTestMode,
  timestamp: number
): ToolTestMessage[] {
  const appended = pending
    .filter(item => item.messageType === mode && item.message.trim() !== '')
    .map((item, index) => ({
      id: buildMessageId('pending', mode, timestamp, messages.length + index),
      kind: 'message' as const,
      mode,
      self: false,
      content: item.message.trim(),
      rawContent: item.message.trim(),
      senderId: 'UI:1000',
      senderName: '海豹核心',
      avatarKey: 'seal',
      isBot: true,
      direction: 'outgoing' as const,
      segments: textSegments(item.message.trim()),
      timestamp,
    }));

  if (!appended.length) return messages;
  return [...messages, ...appended];
}

export function buildToolTestCommandOptions(
  commands: ToolTestCommand[],
  input: string,
  configuredPrefixes: string[] = ['.']
): ToolTestCommandOption[] {
  const prefixes = configuredPrefixes
    .map(prefix => prefix.trim())
    .filter(Boolean)
    .sort((left, right) => right.length - left.length);
  const trimmedInput = input.trimStart();
  const prefix = prefixes.find(candidate => trimmedInput.startsWith(candidate));
  if (!prefix) return [];

  const normalizedQuery = trimmedInput.slice(prefix.length).toLowerCase();
  if (!normalizedQuery) return [];

  const seen = new Set<string>();
  return commands.flatMap(command => {
    const normalizedCommand = command.name.trim();
    const commandKey = normalizedCommand.toLowerCase();
    if (!normalizedCommand || seen.has(commandKey) || !commandKey.startsWith(normalizedQuery)) {
      return [];
    }
    seen.add(commandKey);
    const option: ToolTestCommandOption = {
      label: `${prefix}${normalizedCommand}`,
      value: `${prefix}${normalizedCommand}`,
    };
    if (command.description?.trim()) option.description = command.description.trim();
    if (command.source?.trim()) option.source = command.source.trim();
    return [option];
  });
}

export function normalizeToolTestSplitOptions(
  value:
    | {
        defaultKey?: string | null;
        options?: Array<Partial<ToolTestSplitOption> | null | undefined> | null;
      }
    | undefined
): ToolTestSplitOptionsState {
  const options = (value?.options ?? []).flatMap(item => {
    if (!item?.key || !item.label || typeof item.messageSplitLen !== 'number') return [];
    return [
      {
        key: item.key,
        label: item.label,
        messageSplitLen: item.messageSplitLen,
      },
    ];
  });

  if (!options.length) {
    return {
      defaultKey: 'qq',
      options: DEFAULT_SPLIT_OPTIONS,
    };
  }

  const defaultKey = options.some(item => item.key === value?.defaultKey)
    ? String(value?.defaultKey)
    : options[0].key;
  return { defaultKey, options };
}
