import { storeToRefs } from 'pinia';
import { useBaseLogStreamStore } from './logStreamStore';

export type BaseLogItem = {
  level: string;
  module?: string;
  ts: number;
  caller?: string;
  msg: string;
};

// 日志条目在入库时补一个单调递增的 id。ts + level + msg 并不唯一（同一秒内的重复
// 行会完全相同），而虚拟滚动用行 key 缓存实测行高：key 撞车会让高的那一行把偏移
// 写到别的行上，表现为总高度虚高和滚动回跳。
export type BaseLogEntry = BaseLogItem & { id: number };

export function useBaseLogStream() {
  const store = useBaseLogStreamStore();
  store.ensureInitialized();
  return { ...storeToRefs(store), reconnect: store.reconnect, close: store.close };
}
