/** Field controls supported by the self-contained search toolbar. */
export type SearchField = 'input' | 'select' | 'digit' | 'switch' | 'radio-group' | 'date-range';

/**
 * Schema describing one query field of a search toolbar. `fieldProps` are the
 * Naive UI props forwarded to the underlying control (e.g. options, clearable,
 * placeholder, min).
 */
export interface SearchColumn {
  label: string;
  path: string;
  field: SearchField;
  fieldProps?: Record<string, unknown>;
}
