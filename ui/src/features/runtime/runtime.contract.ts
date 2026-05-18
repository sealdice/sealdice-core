import type { DiceBaseInfo } from './types';
import { useHeartbeat } from './useHeartbeat';

async function assertRuntimeContract(): Promise<void> {
  const heartbeat = useHeartbeat({
    enabled: true,
    immediate: false,
    intervalMs: 5000,
  });

  const offline: boolean = heartbeat.isOffline.value;
  const lastError: unknown = heartbeat.lastError.value;
  const baseInfo: DiceBaseInfo | null = heartbeat.baseInfo.value;
  const refreshed: DiceBaseInfo | null = await heartbeat.refresh();

  heartbeat.pause();
  heartbeat.resume();

  void offline;
  void lastError;
  void baseInfo;
  void refreshed;
}

void assertRuntimeContract;
