import type { TextItemCompatibleInfo, TextResp, Value } from '@/api';

export type TextTemplateItem = [string, number];
export type TextTemplateWithWeight = Record<string, TextTemplateItem[]>;
export type TextTemplateWithWeightDict = Record<string, TextTemplateWithWeight>;
export type TextTemplateHelpGroup = Record<string, Value>;
export type TextTemplateHelpDict = Record<string, TextTemplateHelpGroup>;
export type TextTemplatePreviewDict = Record<string, Record<string, TextItemCompatibleInfo>>;

export function normalizeCustomTextData(data?: TextResp) {
  return {
    texts: normalizeTextDict(data?.texts),
    helpInfo: (data?.helpInfo ?? {}) as TextTemplateHelpDict,
    previewInfo: (data?.previewInfo ?? {}) as TextTemplatePreviewDict,
  };
}

export function normalizeTextDict(
  raw?: TextResp['texts'] | TextTemplateWithWeightDict
): TextTemplateWithWeightDict {
  const out: TextTemplateWithWeightDict = {};
  for (const [category, entries] of Object.entries(raw ?? {})) {
    out[category] = {};
    for (const [key, items] of Object.entries(entries ?? {})) {
      out[category][key] = (items ?? []).map(item => {
        const [text = '', weight = 1] = item ?? [];
        return [String(text ?? ''), Number(weight ?? 1)];
      });
    }
  }
  return out;
}
