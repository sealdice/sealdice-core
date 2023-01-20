
export interface CharItem {
  name: string,
  IMUserId: number | string,
  role: '主持人' | '角色' | '骰子' | '隐藏',
  color: string
}

export interface LogItem {
  id: number;
  nickname: string;
  IMUserId: number | string;
  time: number;
  timeText?: string;
  message: string;
  isDice: boolean;
  commandId: number;
  color?: string;
  role?: string;
  commandInfo?: any;

  // 如果为真，那么只有message有意义，且当作纯文本处理
  isRaw?: boolean;
  index?: number;
}

export function packNameId(i: CharItem | LogItem) {
  return `${(i as any).name || (i as any).nickname}-${i.IMUserId}`
}