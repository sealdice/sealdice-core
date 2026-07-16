import { toggleStoryPainterMode } from './viewMode';
import { it } from 'vitest';

it('passes', async () => {

const assertEqual = (actual: unknown, expected: unknown) => {
  if (actual !== expected) {
    throw new Error(`expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
};

assertEqual(toggleStoryPainterMode('editor', 'preview'), 'preview');
assertEqual(toggleStoryPainterMode('preview', 'preview'), 'editor');
assertEqual(toggleStoryPainterMode('bbs', 'bbspineapple'), 'bbspineapple');
});
