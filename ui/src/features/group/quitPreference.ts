export const defaultGroupQuitText = '因长期不使用等原因，骰主后台操作退出';

const groupQuitDefaultTextStorageKey = 'group-quit-default-text';

type GroupQuitTextStorage = Pick<Storage, 'getItem' | 'setItem'>;

function getBrowserStorage(): GroupQuitTextStorage | undefined {
  if (typeof localStorage === 'undefined') return undefined;
  return localStorage;
}

export function readGroupQuitDefaultText(
  storage: GroupQuitTextStorage | undefined = getBrowserStorage(),
): string {
  try {
    return storage?.getItem(groupQuitDefaultTextStorageKey) || defaultGroupQuitText;
  } catch {
    return defaultGroupQuitText;
  }
}

export function writeGroupQuitDefaultText(
  value: string,
  storage: GroupQuitTextStorage | undefined = getBrowserStorage(),
): void {
  try {
    storage?.setItem(groupQuitDefaultTextStorageKey, value);
  } catch {
    // localStorage may be unavailable in embedded/private contexts.
  }
}
