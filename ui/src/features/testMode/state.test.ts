import { computed, ref } from 'vue';
import { ApiError } from '@/api';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
  useDerivedTestModeState,
} from './state.ts';
import { it } from 'vitest';

it('passes', async () => {
  const assertEqual = (actual: unknown, expected: unknown) => {
    if (actual !== expected) throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  };

  assertEqual(isTestModeResponse({ testMode: true }), true);
  assertEqual(isTestModeResponse({ success: true }), false);
  assertEqual(isTestModeResponse(null), false);

  const blockedError = new ApiError({
    status: 403,
    statusText: 'Forbidden',
    data: {
      code: 'TEST_MODE_BLOCKED',
      detail: '展示模式不支持该操作',
      testMode: true,
    },
  });
  assertEqual(isTestModeApiError(blockedError), true);
  assertEqual(getTestModeBlockMessage(blockedError), '展示模式不支持该操作');

  const generic403 = new ApiError({
    status: 403,
    statusText: 'Forbidden',
    data: {
      detail: '权限不足',
    },
  });
  assertEqual(isTestModeApiError(generic403), false);
  assertEqual(getTestModeBlockMessage(generic403), '');

  const overview = ref<{ runtime?: { justForTest?: boolean } } | undefined>(undefined);
  const derived = useDerivedTestModeState(computed(() => overview.value));
  assertEqual(derived.isTestMode.value, false);
  assertEqual(derived.bannerText.value, '展示模式，仅用于演示，修改不会生效');
  assertEqual(derived.watermarkText.value, '');

  overview.value = { runtime: { justForTest: true } };
  assertEqual(derived.isTestMode.value, true);
  assertEqual(derived.watermarkText.value, '仅用于展示，修改无效');
});
