import dayjs from 'dayjs';
import type { CensorConfigBody, CensorWordItem } from '@/api';

export const CENSOR_HANDLERS = [
  { key: 'SendWarning', name: '发送警告' },
  { key: 'SendNotice', name: '通知 Master' },
  { key: 'BanUser', name: '拉黑用户' },
  { key: 'BanGroup', name: '拉黑群' },
  { key: 'BanInviter', name: '拉黑邀请人' },
  { key: 'AddScore', name: '增加怒气值' },
] as const;

export const CENSOR_MODES = {
  replyOutput: 0,
  commandInput: 1,
  allInput: 2,
} as const;

export type SensitiveTagType = 'default' | 'info' | 'warning' | 'error';

export function createDefaultCensorConfig(): CensorConfigBody {
  return {
    mode: CENSOR_MODES.allInput,
    caseSensitive: false,
    matchPinyin: false,
    filterRegex: '',
    levelConfig: {
      notice: { threshold: 0, handlers: [], score: 0 },
      caution: { threshold: 0, handlers: [], score: 0 },
      warning: { threshold: 0, handlers: [], score: 0 },
      danger: { threshold: 0, handlers: [], score: 0 },
    },
  };
}

export function cloneCensorConfig(config: CensorConfigBody): CensorConfigBody {
  return {
    mode: config.mode,
    caseSensitive: config.caseSensitive,
    matchPinyin: config.matchPinyin,
    filterRegex: config.filterRegex,
    levelConfig: {
      notice: { ...config.levelConfig.notice, handlers: [...(config.levelConfig.notice.handlers ?? [])] },
      caution: { ...config.levelConfig.caution, handlers: [...(config.levelConfig.caution.handlers ?? [])] },
      warning: { ...config.levelConfig.warning, handlers: [...(config.levelConfig.warning.handlers ?? [])] },
      danger: { ...config.levelConfig.danger, handlers: [...(config.levelConfig.danger.handlers ?? [])] },
    },
  };
}

export function isCensorConfigDirty(current: CensorConfigBody, initial: CensorConfigBody) {
  return JSON.stringify(current) !== JSON.stringify(initial);
}

export function getSensitiveTag(level: number | string): { type: SensitiveTagType; label: string } {
  switch (Number(level)) {
    case 1:
      return { type: 'default', label: '提醒' };
    case 2:
      return { type: 'info', label: '注意' };
    case 3:
      return { type: 'warning', label: '警告' };
    case 4:
      return { type: 'error', label: '危险' };
    default:
      return { type: 'default', label: '未知' };
  }
}

export function formatCensorMessageType(msgType: string) {
  if (msgType === 'private') return '私聊';
  if (msgType === 'group') return '群';
  return '未知';
}

export function formatCensorLogTime(timestamp: number) {
  if (!timestamp) return '-';
  return dayjs.unix(timestamp).format('YYYY-MM-DD HH:mm:ss');
}

export function filterCensorWords(words: CensorWordItem[], keyword: string) {
  const value = keyword.trim().toLowerCase();
  if (!value) return words;
  return words.filter(word => {
    if (word.main.toLowerCase().includes(value)) return true;
    return (word.related ?? []).some(item => item.word.toLowerCase().includes(value));
  });
}

export type CensorLogQueryModel = {
  pageNum: number;
  pageSize: number;
};

export function createDefaultCensorLogQuery(): CensorLogQueryModel {
  return {
    pageNum: 1,
    pageSize: 20,
  };
}
