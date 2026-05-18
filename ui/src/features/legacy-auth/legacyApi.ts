import { legacyRequest } from '@/api/legacy';
import type { LegacySaltResult, LegacySigninResult } from './types';

export type { LegacySaltResult, LegacySigninResult };

export function getLegacySalt() {
  return legacyRequest<LegacySaltResult>('get', 'signin/salt');
}

export function legacySignin(password: string) {
  return legacyRequest<LegacySigninResult>('post', 'signin', { password });
}
