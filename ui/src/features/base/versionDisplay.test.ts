import { describe, expect, it } from 'vitest';
import { formatBuildMetaData, formatDisplayVersion } from './versionDisplay';

const devDetail = { major: 1, minor: 6, patch: 1, buildMetaData: '20260810.1a2b3c4' };

describe('formatBuildMetaData', () => {
  it('压缩为 MMDD.4位hash', () => {
    expect(formatBuildMetaData('20260810.1a2b3c4')).toBe('0810.1a2b');
  });

  it('统一小写 hash', () => {
    expect(formatBuildMetaData('20260810.ABCDEF1')).toBe('0810.abcd');
  });

  it('非十六进制的提交号不予解析', () => {
    expect(formatBuildMetaData('20260810.zzzzzzz')).toBeUndefined();
  });

  it('hash 短于 4 位时按原样保留', () => {
    expect(formatBuildMetaData('20260810.ab')).toBe('0810.ab');
  });

  it('格式不符合预期时不猜测', () => {
    expect(formatBuildMetaData('20260810')).toBeUndefined();
    expect(formatBuildMetaData('')).toBeUndefined();
    expect(formatBuildMetaData(undefined)).toBeUndefined();
  });
});

describe('formatDisplayVersion', () => {
  it('开发版显示主版本号与编译信息，且不带 -dev', () => {
    const text = formatDisplayVersion({ simple: '1.6.1-dev', detail: devDetail }, 'dev');
    expect(text).toBe('1.6.1+0810.1a2b');
    expect(text).not.toContain('-dev');
  });

  it('未知渠道同样显示编译信息', () => {
    expect(formatDisplayVersion({ detail: devDetail }, 'unknown')).toBe('1.6.1+0810.1a2b');
  });

  it('正式版只显示主版本号', () => {
    expect(
      formatDisplayVersion(
        { detail: { major: 1, minor: 6, patch: 1, buildMetaData: '20260810' } },
        'stable'
      )
    ).toBe('1.6.1');
  });

  it('自编译即便带编译信息也只显示主版本号', () => {
    expect(formatDisplayVersion({ detail: devDetail }, 'self-built')).toBe('1.6.1');
  });

  it('开发版缺少编译信息时退回主版本号', () => {
    expect(formatDisplayVersion({ detail: { major: 1, minor: 6, patch: 1 } }, 'dev')).toBe('1.6.1');
  });

  it('没有结构化版本号时退回 simple 并去掉先行版本号', () => {
    expect(formatDisplayVersion({ simple: '1.6.1-dev' }, 'dev')).toBe('1.6.1');
  });

  it('完全没有数据时给出占位', () => {
    expect(formatDisplayVersion(undefined, 'dev')).toBe('-');
    expect(formatDisplayVersion({}, 'stable')).toBe('-');
  });
});
