import { getErrorMessage } from './error';

function assertEqual(actual: unknown, expected: unknown): void {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
}

assertEqual(
  getErrorMessage({
    code: 'ERR_CANCELED',
    message: 'canceled',
    name: 'CanceledError',
  }, '读取失败'),
  '',
);
assertEqual(getErrorMessage(new Error('boom'), '读取失败'), 'boom');
