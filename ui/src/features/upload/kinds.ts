// 上传种类注册表：把"哪个页面传什么、对应后端 module、允许什么类型"集中在一处，
// 供全局 store 与各页面共用。后端已统一到 tus 单一端点，这里只管 module 与元数据。

export type UploadKindID =
  | 'js'
  | 'deck'
  | 'helpdoc'
  | 'resource'
  | 'censor'
  | 'custom-reply'
  | 'extension-install';

export type UploadKindContext = {
  group?: string;
};

export type UploadKind = {
  id: UploadKindID;
  label: string;
  /** tus Upload-Metadata 的 module 字段（后端 Destination 据此分发）。 */
  module: string;
  /** 允许的扩展名；空数组表示不限制。 */
  allowedExtensions: readonly string[];
  /** 追加到 tus metadata 的额外字段（如 helpdoc 的 group）。 */
  extraMeta?: (ctx: UploadKindContext) => Record<string, string>;
};

export const UPLOAD_KINDS: Record<UploadKindID, UploadKind> = {
  js: { id: 'js', label: 'JS 脚本', module: 'js', allowedExtensions: [] },
  deck: { id: 'deck', label: '牌堆', module: 'deck', allowedExtensions: [] },
  helpdoc: {
    id: 'helpdoc',
    label: '帮助文档',
    module: 'helpdoc',
    allowedExtensions: ['.json', '.xlsx'],
    extraMeta: (ctx): Record<string, string> => (ctx.group ? { group: ctx.group } : {}),
  },
  resource: {
    id: 'resource',
    label: '图片资源',
    module: 'resource',
    allowedExtensions: ['.png', '.jpg', '.jpeg', '.gif'],
  },
  censor: {
    id: 'censor',
    label: '敏感词库',
    module: 'censor',
    allowedExtensions: ['.txt', '.toml'],
  },
  'custom-reply': {
    id: 'custom-reply',
    label: '自定义回复',
    module: 'custom-reply',
    allowedExtensions: [],
  },
  'extension-install': {
    id: 'extension-install',
    label: '扩展包',
    module: 'extension-install',
    allowedExtensions: [],
  },
};

function extensionOf(filename: string): string {
  const index = filename.lastIndexOf('.');
  return index >= 0 ? filename.slice(index).toLowerCase() : '';
}

export function isFileAllowed(kind: UploadKindID, filename: string): boolean {
  const allowed = UPLOAD_KINDS[kind].allowedExtensions;
  if (allowed.length === 0) return true;
  return allowed.includes(extensionOf(filename));
}
