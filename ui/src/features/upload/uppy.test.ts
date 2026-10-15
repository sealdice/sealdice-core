import { describe, expect, it } from 'vitest';
import { getUppy, uploadEndpoint } from './uppy';

describe('upload uppy singleton', () => {
  it('returns the same instance on repeated calls', () => {
    expect(getUppy()).toBe(getUppy());
  });

  it('targets the unified tus upload endpoint', () => {
    expect(uploadEndpoint()).toMatch(/\/sd-api\/v2\/upload\/$/);
  });

  it('allows re-adding the same file without a duplicate restriction deadlock', () => {
    const uppy = getUppy();
    const addOnce = () =>
      uppy.addFile({
        name: 'dup.png',
        type: 'image/png',
        data: new File([new Uint8Array(3)], 'dup.png', {
          type: 'image/png',
          lastModified: 1700000000000,
        }),
        meta: { taskId: 'probe' },
      });
    expect(addOnce).not.toThrow();
    expect(addOnce).not.toThrow();

    // 单例隔离：清理本用例塞进去的文件，避免污染后续用例。
    for (const file of uppy.getFiles()) {
      uppy.removeFile(file.id);
    }
    expect(uppy.getFiles()).toHaveLength(0);
  });
});
