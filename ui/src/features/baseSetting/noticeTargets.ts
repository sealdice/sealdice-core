export type NoticeType = 'group' | 'invite' | 'ban' | 'censor' | 'inactive' | 'send' | 'system';

export const noticeTypeOptions: Array<{ label: string; value: NoticeType }> = [
  { label: '群事件', value: 'group' },
  { label: '入群邀请', value: 'invite' },
  { label: '黑名单', value: 'ban' },
  { label: '敏感词', value: 'censor' },
  { label: '自动退群', value: 'inactive' },
  { label: '消息发送', value: 'send' },
  { label: '系统通知', value: 'system' },
];

export type NoticeTargetModel = {
  id: string;
  enabled: boolean;
  noticeTypes: NoticeType[];
};

const noticeTypeOrder = noticeTypeOptions.map(option => option.value);
const validNoticeTypes = new Set<NoticeType>(noticeTypeOrder);

export function parseNoticeTarget(raw: string): NoticeTargetModel {
  const parts = raw.trim().split(':');
  let end = parts.length;
  let enabled = true;
  let noticeTypes: NoticeType[] | null = null;

  while (end > 1) {
    const suffix = parts[end - 1]?.trim() ?? '';
    if (suffix === 'disable') {
      enabled = false;
      end -= 1;
      continue;
    }
    if (suffix.startsWith('only=')) {
      if (noticeTypes === null) {
        noticeTypes = suffix
          .slice('only='.length)
          .split(',')
          .map(item => item.trim() as NoticeType)
          .filter(item => validNoticeTypes.has(item));
      }
      end -= 1;
      continue;
    }
    break;
  }

  const selected = new Set(noticeTypes ?? noticeTypeOrder);
  return {
    id: parts.slice(0, end).join(':').trim(),
    enabled,
    noticeTypes: noticeTypeOrder.filter(type => selected.has(type)),
  };
}

export function serializeNoticeTarget(target: NoticeTargetModel): string {
  const id = target.id.trim();
  if (!id) return '';

  const selected = new Set(target.noticeTypes.filter(type => validNoticeTypes.has(type)));
  const orderedTypes = noticeTypeOrder.filter(type => selected.has(type));
  let result = id;
  if (!target.enabled) result += ':disable';
  if (orderedTypes.length !== noticeTypeOrder.length) result += `:only=${orderedTypes.join(',')}`;
  return result;
}
