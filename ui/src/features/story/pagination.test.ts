import { describe, expect, it } from 'vitest';
import { getStoryPageSizeChange } from './pagination';

describe('getStoryPageSizeChange', () => {
  it('resets the raw log page when the page size changes', () => {
    expect(getStoryPageSizeChange(200)).toEqual({ pageNum: 1, pageSize: 200 });
  });
});
