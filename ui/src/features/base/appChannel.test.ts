import { describe, expect, it } from 'vitest';
import {
  formatAppChannel,
  getAppChannelHint,
  getAppChannelTagType,
  normalizeAppChannel,
  shouldShowBuildMetaData,
} from './appChannel';

describe('formatAppChannel', () => {
  it('覆盖四档渠道文案', () => {
    expect(formatAppChannel('stable')).toBe('正式版');
    expect(formatAppChannel('dev')).toBe('开发版');
    expect(formatAppChannel('self-built')).toBe('自编译');
    expect(formatAppChannel('unknown')).toBe('未知');
  });

  it('无法识别的取值落到未知', () => {
    expect(formatAppChannel('nightly')).toBe('未知');
    expect(formatAppChannel(undefined)).toBe('未知');
  });
});

describe('normalizeAppChannel', () => {
  it('保留已知取值', () => {
    expect(normalizeAppChannel('self-built')).toBe('self-built');
  });

  it('未知取值归一到 unknown', () => {
    expect(normalizeAppChannel('rc')).toBe('unknown');
  });
});

describe('getAppChannelTagType', () => {
  it('按语义分配 badge 颜色', () => {
    expect(getAppChannelTagType('stable')).toBe('success');
    expect(getAppChannelTagType('dev')).toBe('primary');
    expect(getAppChannelTagType('self-built')).toBe('warning');
    expect(getAppChannelTagType('unknown')).toBe('default');
  });
});

describe('getAppChannelHint', () => {
  it('正式版不加说明', () => {
    expect(getAppChannelHint('stable')).toBeUndefined();
  });

  it('其余渠道说明来源与可信度', () => {
    expect(getAppChannelHint('dev')).toBeTruthy();
    expect(getAppChannelHint('self-built')).toBeTruthy();
    expect(getAppChannelHint('unknown')).toBeTruthy();
  });
});

describe('shouldShowBuildMetaData', () => {
  it('仅开发版与未知渠道展示构建日期与提交号', () => {
    expect(shouldShowBuildMetaData('dev')).toBe(true);
    expect(shouldShowBuildMetaData('unknown')).toBe(true);
    expect(shouldShowBuildMetaData('stable')).toBe(false);
    expect(shouldShowBuildMetaData('self-built')).toBe(false);
  });
});
