import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue';
import { useIntervalFn } from '@vueuse/core';
import { getBaseInfo } from './legacyApi';
import type { DiceBaseInfo } from './types';

export interface UseHeartbeatOptions {
  enabled?: MaybeRefOrGetter<boolean>;
  immediate?: boolean;
  intervalMs?: number;
}

const defaultHeartbeatIntervalMs = 5000;

export function useHeartbeat(options: UseHeartbeatOptions = {}) {
  const baseInfo = ref<DiceBaseInfo | null>(null);
  const lastError = ref<unknown>(null);
  const isOffline = ref(false);
  const enabled = computed(() => toValue(options.enabled ?? true));
  const shouldRefreshImmediately = options.immediate ?? true;

  async function refresh(): Promise<DiceBaseInfo | null> {
    if (!enabled.value) {
      return baseInfo.value;
    }

    try {
      const result = await getBaseInfo();
      baseInfo.value = result;
      lastError.value = null;
      isOffline.value = false;
      return result;
    } catch (error) {
      lastError.value = error;
      isOffline.value = true;
      throw error;
    }
  }

  const controls = useIntervalFn(
    () => {
      void refresh().catch(() => undefined);
    },
    options.intervalMs ?? defaultHeartbeatIntervalMs,
    { immediate: false },
  );

  watch(
    enabled,
    value => {
      if (value) {
        controls.resume();
        if (shouldRefreshImmediately) {
          void refresh().catch(() => undefined);
        }
      } else {
        controls.pause();
      }
    },
    { immediate: true },
  );

  return {
    baseInfo,
    isOffline,
    lastError,
    isActive: controls.isActive,
    refresh,
    pause: controls.pause,
    resume: controls.resume,
  };
}
