export type ListPaginationVisibilityOptions = {
  total?: number;
  page: number;
  pageSize: number;
  hasNext?: boolean;
};

/**
 * Pagination is useful only when the current result is a window into a larger
 * collection. Keep it visible beyond page one so users can always navigate
 * back, including after filters or deletions leave the current page empty.
 */
export function shouldShowListPagination({
  total,
  page,
  pageSize,
  hasNext = false,
}: ListPaginationVisibilityOptions): boolean {
  if (page > 1 || hasNext) return true;
  if (total === undefined || pageSize <= 0) return false;
  return total > pageSize;
}

export function getCursorPaginationItemCount({
  page,
  pageSize,
  itemCount,
  hasNext,
}: {
  page: number;
  pageSize: number;
  itemCount: number;
  hasNext: boolean;
}): number {
  const completedPageCount = Math.max(0, page - 1);
  const currentPageItemCount = Math.max(itemCount, page > 1 ? 1 : 0);
  return completedPageCount * pageSize + currentPageItemCount + (hasNext ? 1 : 0);
}
