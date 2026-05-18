import { ApiError } from './error';
import { createHttpErrorFeedback, createNetworkErrorFeedback } from './httpStatusFeedback';

function assertEqual(actual: unknown, expected: unknown): void {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
}

function makeApiError(status: number, data: unknown = { detail: 'boom' }): ApiError {
  return new ApiError({
    status,
    statusText: `status ${status}`,
    data,
  });
}

const unauthorized = createHttpErrorFeedback(makeApiError(401, { detail: 'unauthorized' }));
assertEqual(unauthorized.kind, 'dialog');
if (unauthorized.kind === 'dialog') {
  assertEqual(unauthorized.title, '身份信息');
  assertEqual(unauthorized.clearSession, true);
}

const notFound = createHttpErrorFeedback(makeApiError(404));
assertEqual(notFound.kind, 'dialog');
if (notFound.kind === 'dialog') {
  assertEqual(notFound.title, '接口报错');
  assertEqual(notFound.clearSession, undefined);
}

const serverError = createHttpErrorFeedback(makeApiError(500));
assertEqual(serverError.kind, 'dialog');
if (serverError.kind === 'dialog') {
  assertEqual(serverError.title, '接口报错');
  assertEqual(serverError.clearSession, true);
}

const forbidden = createHttpErrorFeedback(makeApiError(403, { detail: '展示模式不支持该操作' }));
assertEqual(forbidden.kind, 'message');
if (forbidden.kind === 'message') {
  assertEqual(forbidden.content, '展示模式不支持该操作');
}

const networkError = createNetworkErrorFeedback(new Error('Failed to fetch'));
assertEqual(networkError.kind, 'dialog');
if (networkError.kind === 'dialog') {
  assertEqual(networkError.title, '请求报错');
}
