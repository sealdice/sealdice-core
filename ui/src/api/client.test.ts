import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { AxiosError, AxiosHeaders, type InternalAxiosRequestConfig } from 'axios';
import { appPinia } from '@/pinia';
import { useAuthStore } from '@/features/auth/store';
import { client } from './generated/client.gen';
import { setupApiClient } from './client';
import { isApiErrorHandled } from './errors';

const auth = useAuthStore(appPinia);
let dispose: () => void;
const globalError = vi.fn();
beforeEach(() => {
  auth.setAccessToken('token-a');
  auth.finishInitialization();
  globalError.mockClear();
  dispose = setupApiClient(globalError);
});
afterEach(() => {
  dispose();
  auth.clearAccessToken();
});

function rejectResponse(config: InternalAxiosRequestConfig, status: number) {
  return Promise.reject(
    new AxiosError('request failed', 'ERR_BAD_RESPONSE', config, undefined, {
      status,
      statusText: 'failed',
      data: { detail: '保存失败' },
      headers: {},
      config,
    })
  );
}

it('实际请求注入认证头，全局错误只交给一个展示入口', async () => {
  const error = await client.instance
    .get('/sd-api/v2/test', {
      adapter: config => {
        expect(config.headers.get('Authorization')).toBe('Bearer token-a');
        return rejectResponse(config, 500);
      },
    })
    .catch(error => error);
  expect(error.status).toBe(500);
  expect(globalError).toHaveBeenCalledOnce();
  expect(isApiErrorHandled(error)).toBe(true);
});

it('业务校验错误交给调用方，不弹全局错误', async () => {
  const error = await client.instance
    .post(
      '/sd-api/v2/test',
      {},
      {
        adapter: config => rejectResponse(config, 422),
      }
    )
    .catch(error => error);
  expect(globalError).not.toHaveBeenCalled();
  expect(isApiErrorHandled(error)).toBe(false);
});

it('旧请求的 401 不清空刚建立的新会话', async () => {
  const error = await client.instance
    .get('/sd-api/v2/test', {
      adapter: config => {
        auth.setAccessToken('token-b');
        return rejectResponse(config, 401);
      },
    })
    .catch(error => error);
  expect(auth.token).toBe('token-b');
  expect(globalError).not.toHaveBeenCalled();
  expect(isApiErrorHandled(error)).toBe(true);
});

it('当前会话的 401 清理会话后交由解锁弹窗处理，不再重复弹全局错误', async () => {
  const error = await client.instance
    .get('/sd-api/v2/test', {
      adapter: config => rejectResponse(config, 401),
    })
    .catch(error => error);
  expect(auth.token).toBe('');
  expect(globalError).not.toHaveBeenCalled();
  expect(isApiErrorHandled(error)).toBe(true);
});

it('旧请求的续期响应不覆盖新会话', async () => {
  await client.instance.get('/sd-api/v2/test', {
    adapter: async config => {
      auth.setAccessToken('token-b');
      return {
        status: 200,
        statusText: 'OK',
        data: {},
        config,
        headers: new AxiosHeaders({ 'new-token': 'old-renewed' }),
      };
    },
  });
  expect(auth.token).toBe('token-b');
});
