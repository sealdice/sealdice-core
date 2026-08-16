export type GroupQuitTarget = {
  groupId: string;
  groupName?: string;
  diceId?: string;
};

export function formatGroupQuitTarget(target: GroupQuitTarget): string {
  const groupName = target.groupName?.trim() ? `「${target.groupName.trim()}」` : '';
  const diceId = target.diceId ? `，账号 ${target.diceId}` : '';
  return `${target.groupId}${groupName}${diceId}`;
}
