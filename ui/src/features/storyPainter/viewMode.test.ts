import { toggleStoryPainterMode } from './viewMode';
import { expect, it } from 'vitest';

it('日志视图模式保留合法偏好', async () => {
  expect(toggleStoryPainterMode('editor', 'preview')).toBe('preview');
  expect(toggleStoryPainterMode('preview', 'preview')).toBe('editor');
  expect(toggleStoryPainterMode('bbs', 'bbspineapple')).toBe('bbspineapple');
});
