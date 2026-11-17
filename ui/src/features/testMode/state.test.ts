import { ref } from 'vue';
import { ApiError } from '@/api';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
  useTestMode,
} from './state.ts';
import { expect, it, vi } from 'vitest';
const overview = ref<{ runtime?: { justForTest?: boolean } } | undefined>(undefined);
vi.mock('@/features/base/useBaseOverview', () => ({ useBaseOverview: () => ({ overview }) }));

it('展示模式限制与错误标识识别', async () => {
  expect(isTestModeResponse({ testMode: true })).toBe(true);
  expect(isTestModeResponse({ success: true })).toBe(false);
  expect(isTestModeResponse(null)).toBe(false);

  const blockedError = new ApiError({
    status: 403,
    statusText: 'Forbidden',
    data: {
      code: 'TEST_MODE_BLOCKED',
      detail: '展示模式不支持该操作',
      testMode: true,
    },
  });
  expect(isTestModeApiError(blockedError)).toBe(true);
  expect(getTestModeBlockMessage(blockedError)).toBe('展示模式不支持该操作');

  const generic403 = new ApiError({
    status: 403,
    statusText: 'Forbidden',
    data: {
      detail: '权限不足',
    },
  });
  expect(isTestModeApiError(generic403)).toBe(false);
  expect(getTestModeBlockMessage(generic403)).toBe('');

  const derived = useTestMode();
  expect(derived.isTestMode.value).toBe(false);
  expect(derived.bannerText.value).toBe('展示模式，仅用于演示，修改不会生效');
  expect(derived.watermarkText.value).toBe('');

  overview.value = { runtime: { justForTest: true } };
  expect(derived.isTestMode.value).toBe(true);
  expect(derived.watermarkText.value).toBe('仅用于展示，修改无效');
});
