export const UI_BETA_NOTICE_STORAGE_KEY = 'sd-ui-beta-notice-dismissed-v1';
export const UI_BETA_FEEDBACK_URL = 'https://github.com/sealdice/sealdice-core/issues';

// 采用窄接口（只依赖 getItem/setItem），便于在非浏览器环境与测试中复用。
type ReadableStorage = Pick<Storage, 'getItem'>;
type WritableStorage = Pick<Storage, 'setItem'>;

export function readUiBetaNoticeDismissed(storage: ReadableStorage): boolean {
  try {
    return storage.getItem(UI_BETA_NOTICE_STORAGE_KEY) === '1';
  } catch {
    return false;
  }
}

export function persistUiBetaNoticeDismissed(storage: WritableStorage): void {
  try {
    storage.setItem(UI_BETA_NOTICE_STORAGE_KEY, '1');
  } catch {
    // 隐私模式等场景下存储不可用，忽略即可。
  }
}
