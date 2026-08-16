export type JsListParams = {
  page: number;
  pageSize: number;
  keyword?: string;
  sortBy: string;
  sortOrder: 'asc' | 'desc' | string;
};

export type JsDataPage = {
  page: number;
  pageSize: number;
};

export const jsListQueryKey = (params: JsListParams) => ['js-list', params] as const;

export const jsConfigsQueryKey = () => ['js-configs'] as const;

export const jsDeadConfigsQueryKey = () => ['js-dead-configs'] as const;

export const jsListForDataQueryKey = () => ['js-list-for-data'] as const;

export const jsDataListQueryKey = (pluginName: string, page: JsDataPage, keyword: string) =>
  ['js-data-list', pluginName, page, keyword] as const;

export const jsDataInfoQueryKey = (pluginName: string) => ['js-data-info', pluginName] as const;
