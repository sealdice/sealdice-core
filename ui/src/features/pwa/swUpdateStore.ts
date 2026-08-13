import { defineStore } from 'pinia';
import { shallowRef } from 'vue';
import type { SwUpdateReason } from './swUpdateState';

// 升级提示的全局状态。触发源（SW 换版 / chunk 404）只负责上报，
// 是否弹窗、是否自动刷新由 AppUpdatePrompt 按未保存变更状态决定。
export const useSwUpdateStore = defineStore('sw-update', () => {
  const updateReady = shallowRef(false);
  const updateReason = shallowRef<SwUpdateReason>('sw-update');
  const navigationTarget = shallowRef<string | null>(null);
  const dismissed = shallowRef(false);
  const applying = shallowRef(false);

  function markUpdateReady(reason: SwUpdateReason, target?: string | null): void {
    if (applying.value) return;
    // 用户选过“稍后”后，被动的新版本提示本会话不再打扰；
    // 但 chunk 加载失败意味着页面已部分不可用，需要再次提醒。
    if (dismissed.value && reason === 'sw-update') return;
    updateReason.value = reason;
    navigationTarget.value = target ?? null;
    updateReady.value = true;
  }

  function dismiss(): void {
    dismissed.value = true;
    updateReady.value = false;
  }

  function markApplying(): void {
    applying.value = true;
  }

  return {
    updateReady,
    updateReason,
    navigationTarget,
    dismissed,
    applying,
    markUpdateReady,
    dismiss,
    markApplying,
  };
});
