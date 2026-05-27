import { passwordHash } from './crypto';

function assertEqual(actual: unknown, expected: unknown): void {
  if (actual !== expected) {
    throw new Error(`expected ${String(expected)}, got ${String(actual)}`);
  }
}

function assertMatch(actual: string, expected: RegExp): void {
  if (!expected.test(actual)) {
    throw new Error(`expected ${actual} to match ${String(expected)}`);
  }
}

const first = await passwordHash('salt-value', 'password');
const second = await passwordHash('salt-value', 'password');
const differentPassword = await passwordHash('salt-value', 'password-2');

assertEqual(first, second);
assertMatch(first, /^djAx/);
assertEqual(first === differentPassword, false);

const originalCrypto = globalThis.crypto;
Object.defineProperty(globalThis, 'crypto', {
  configurable: true,
  value: undefined,
});

const fallback = await passwordHash('salt-value', 'password');
assertEqual(fallback, first);

Object.defineProperty(globalThis, 'crypto', {
  configurable: true,
  value: originalCrypto,
});
