export type GroupQuitTarget = {
  groupId: string;
  groupName?: string;
  diceId?: string;
};

export function formatGroupQuitTarget(target: GroupQuitTarget): string {
  const groupName = target.groupName?.trim() ? '「' + target.groupName.trim() + '」' : '';
  const diceId = target.diceId ? '，账号 ' + target.diceId : '';
  return target.groupId + groupName + diceId;
}

export const defaultGroupQuitText = '因长期不使用等原因，骰主后台操作退出';

const groupQuitDefaultTextStorageKey = 'group-quit-default-text';
type GroupQuitTextStorage = Pick<Storage, 'getItem' | 'setItem'>;

function getBrowserStorage(): GroupQuitTextStorage | undefined {
  if (typeof localStorage === 'undefined') return undefined;
  return localStorage;
}

export function readGroupQuitDefaultText(
  storage: GroupQuitTextStorage | undefined = getBrowserStorage()
): string {
  try {
    return storage?.getItem(groupQuitDefaultTextStorageKey) || defaultGroupQuitText;
  } catch {
    return defaultGroupQuitText;
  }
}

export function writeGroupQuitDefaultText(
  value: string,
  storage: GroupQuitTextStorage | undefined = getBrowserStorage()
): void {
  try {
    storage?.setItem(groupQuitDefaultTextStorageKey, value);
  } catch {
    // 本地存储不可用不影响本次操作。
  }
}
