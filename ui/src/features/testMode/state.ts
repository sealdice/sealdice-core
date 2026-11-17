import { computed } from 'vue';
import { ApiError, TEST_MODE_ERROR_CODE } from '@/api/errors';
import { useBaseOverview } from '@/features/base/useBaseOverview';

export { TEST_MODE_ERROR_CODE };
export const TEST_MODE_BANNER_TEXT = '展示模式，仅用于演示，修改不会生效';
export const TEST_MODE_WATERMARK_TEXT = '仅用于展示，修改无效';
export const TEST_MODE_DEFAULT_REASON = '展示模式不支持该操作';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function pickTestModeData(value: unknown): Record<string, unknown> | null {
  return isRecord(value) ? value : null;
}

export function isTestModeResponse(value: unknown): boolean {
  const data = pickTestModeData(value);
  return data?.testMode === true;
}

export function isTestModeApiError(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false;
  const data = pickTestModeData(error.data);
  if (data?.testMode === true) return true;
  if (error.code === TEST_MODE_ERROR_CODE) return true;
  if (error.status !== 403) return false;
  return (
    error.detail === TEST_MODE_DEFAULT_REASON ||
    error.message === TEST_MODE_DEFAULT_REASON ||
    error.title === TEST_MODE_DEFAULT_REASON
  );
}

export function getTestModeBlockMessage(
  error: unknown,
  fallback = TEST_MODE_DEFAULT_REASON
): string {
  if (!isTestModeApiError(error) && !isTestModeResponse(error)) {
    return '';
  }
  if (error instanceof ApiError) {
    const data = pickTestModeData(error.data);
    if (typeof data?.detail === 'string' && data.detail.trim() !== '') {
      return data.detail;
    }
    if (typeof data?.message === 'string' && data.message.trim() !== '') {
      return data.message;
    }
    if (error.message.trim() !== '') {
      return error.message;
    }
  }
  if (isTestModeResponse(error)) {
    return fallback;
  }
  return '';
}

export function useTestMode() {
  const { overview } = useBaseOverview();
  const isTestMode = computed(() => overview.value?.runtime?.justForTest === true);
  const bannerText = computed(() => TEST_MODE_BANNER_TEXT);
  const watermarkText = computed(() => (isTestMode.value ? TEST_MODE_WATERMARK_TEXT : ''));

  return {
    isTestMode,
    bannerText,
    watermarkText,
  };
}
