import { getErrorMessage } from './error';
import { expect, it } from 'vitest';

it('提取业务错误信息并忽略取消请求', async () => {
  expect(
    getErrorMessage(
      {
        code: 'ERR_CANCELED',
        message: 'canceled',
        name: 'CanceledError',
      },
      '读取失败'
    )
  ).toBe('');
  expect(getErrorMessage(new Error('boom'), '读取失败')).toBe('boom');
});
