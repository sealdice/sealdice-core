import { describe, expect, it } from 'vitest';
import { getCursorPaginationItemCount, shouldShowListPagination } from './listPagination';

describe('list pagination visibility', () => {
  it('hides pagination for empty and single-page first-page results', () => {
    expect(shouldShowListPagination({ total: 0, page: 1, pageSize: 20 })).toBe(false);
    expect(shouldShowListPagination({ total: 20, page: 1, pageSize: 20 })).toBe(false);
  });

  it('shows pagination when another page can be reached', () => {
    expect(shouldShowListPagination({ total: 21, page: 1, pageSize: 20 })).toBe(true);
    expect(shouldShowListPagination({ page: 1, pageSize: 20, hasNext: true })).toBe(true);
  });

  it('keeps pagination beyond page one so users can navigate back', () => {
    expect(shouldShowListPagination({ total: 0, page: 2, pageSize: 20 })).toBe(true);
    expect(shouldShowListPagination({ page: 3, pageSize: 20 })).toBe(true);
  });
});

describe('cursor pagination item count', () => {
  it('creates only the sentinel item needed to expose a next page', () => {
    expect(
      getCursorPaginationItemCount({ page: 1, pageSize: 20, itemCount: 20, hasNext: true })
    ).toBe(21);
    expect(
      getCursorPaginationItemCount({ page: 2, pageSize: 20, itemCount: 4, hasNext: false })
    ).toBe(24);
    expect(
      getCursorPaginationItemCount({ page: 2, pageSize: 20, itemCount: 0, hasNext: false })
    ).toBe(21);
  });
});
