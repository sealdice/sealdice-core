import { expect, it } from 'vitest';
import {
  ApiError,
  isApiErrorHandled,
  markApiErrorHandled,
  needsGlobalApiFeedback,
  isRequestCanceledError,
  shouldClearSessionForRequest,
} from './errors';

it.each([
  [401, false],
  [400, false],
  [403, false],
  [422, false],
  [404, true],
  [500, true],
  [0, true],
])('按状态 %i 分配错误展示责任', (status, global) => {
  const error = new ApiError({ status, statusText: '错误', data: { detail: '具体原因' } });
  expect(needsGlobalApiFeedback(error)).toBe(global);
  expect(error.message).toBe('具体原因');
});

it('401 交给解锁弹窗处理，不再重复弹全局错误框', () => {
  const error = new ApiError({ status: 401, statusText: 'Unauthorized', data: null });
  expect(needsGlobalApiFeedback(error)).toBe(false);
});

it('测试模式错误交给业务层，不按普通 500 弹窗', () => {
  expect(
    needsGlobalApiFeedback(new ApiError({ status: 500, statusText: '', data: { testMode: true } }))
  ).toBe(false);
});

it('仅已反馈的错误和取消操作跳过局部提示', () => {
  const error = new Error('失败');
  expect(isApiErrorHandled(error)).toBe(false);
  markApiErrorHandled(error);
  expect(isApiErrorHandled(error)).toBe(true);
  expect(isApiErrorHandled(new Error('失败'))).toBe(false);
  expect(isRequestCanceledError({ code: 'ERR_CANCELED' })).toBe(true);
  expect(isRequestCanceledError(new DOMException('取消', 'AbortError'))).toBe(true);
});

it('旧请求的令牌不能清除新会话', () => {
  expect(shouldClearSessionForRequest('old', 'new')).toBe(false);
  expect(shouldClearSessionForRequest(undefined, '')).toBe(false);
  expect(shouldClearSessionForRequest('current', 'current')).toBe(true);
});
