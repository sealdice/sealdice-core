/**
 * 发布渠道。后端由「构建声明的渠道」与「官方构建校验结果」共同得出，
 * 前端只做展示映射，不自行推断可信度。
 */
export type AppChannel = 'stable' | 'dev' | 'self-built' | 'unknown';

/** badge 语义色：正式版是已确认的正常状态，自编译需要警惕，未知不作断言。 */
export type AppChannelTagType = 'success' | 'primary' | 'warning' | 'default';

const CHANNEL_TEXT: Record<AppChannel, string> = {
  stable: '正式版',
  dev: '开发版',
  'self-built': '自编译',
  unknown: '未知',
};

const CHANNEL_TAG_TYPE: Record<AppChannel, AppChannelTagType> = {
  stable: 'success',
  dev: 'primary',
  'self-built': 'warning',
  unknown: 'default',
};

/**
 * 渠道说明。仅补充 badge 文案无法直接得知的来源与可信度信息，
 * 正式版不加说明：它是默认预期状态，重复一句「这是正式版」没有信息量。
 */
const CHANNEL_HINT: Partial<Record<AppChannel, string>> = {
  dev: '包含尚未发布的改动',
  'self-built': '由本地源码编译，未经官方构建校验',
  unknown: '未能通过官方构建校验，无法确认来源',
};

export function normalizeAppChannel(channel: string | undefined): AppChannel {
  if (channel === 'stable' || channel === 'dev' || channel === 'self-built') return channel;
  return 'unknown';
}

export function formatAppChannel(channel: string | undefined): string {
  return CHANNEL_TEXT[normalizeAppChannel(channel)];
}

export function getAppChannelTagType(channel: string | undefined): AppChannelTagType {
  return CHANNEL_TAG_TYPE[normalizeAppChannel(channel)];
}

export function getAppChannelHint(channel: string | undefined): string | undefined {
  return CHANNEL_HINT[normalizeAppChannel(channel)];
}

/**
 * 是否展示构建日期与提交号。开发版与未知渠道的版本号本身不足以定位具体构建，
 * 需要日期与 hash 才能对应到源码；正式版与自编译即使带有这些信息也不展示，
 * 前者主版本号已经唯一，后者的构建信息不代表官方发布。
 */
export function shouldShowBuildMetaData(channel: string | undefined): boolean {
  const normalized = normalizeAppChannel(channel);
  return normalized === 'dev' || normalized === 'unknown';
}
