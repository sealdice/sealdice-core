import { passwordHash } from './crypto';
import { expect, it } from 'vitest';

it('认证摘要与密码派生结果符合协议', async () => {
  function assertMatch(actual: string, expected: RegExp): void {
    if (!expected.test(actual)) {
      throw new Error(`expected ${actual} to match ${String(expected)}`);
    }
  }

  const first = await passwordHash('salt-value', 'password');
  const second = await passwordHash('salt-value', 'password');
  const differentPassword = await passwordHash('salt-value', 'password-2');

  expect(first).toBe(second);
  assertMatch(first, /^djAx/);
  expect(first === differentPassword).toBe(false);

  const originalCrypto = globalThis.crypto;
  Object.defineProperty(globalThis, 'crypto', {
    configurable: true,
    value: undefined,
  });

  const fallback = await passwordHash('salt-value', 'password');
  expect(fallback).toBe(first);

  Object.defineProperty(globalThis, 'crypto', {
    configurable: true,
    value: originalCrypto,
  });
});
