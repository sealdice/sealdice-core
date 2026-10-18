import type { MessageApi } from 'naive-ui';
import { isApiErrorHandled } from '@/api/errors';
import { getErrorMessage } from '@/features/auth/error';

export function notifyError(
  error: unknown,
  messages: Pick<MessageApi, 'error'>,
  fallback: string
): void {
  if (!isApiErrorHandled(error)) messages.error(getErrorMessage(error, fallback));
}
