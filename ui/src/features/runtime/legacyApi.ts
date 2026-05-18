import { legacyRequest } from '@/api/legacy';
import type {
  DiceBaseInfo,
  PreInfo,
  SecurityStatus,
  SysLog,
  ToolOnebotResult,
} from './types';

export type { DiceBaseInfo, PreInfo, SecurityStatus, SysLog, ToolOnebotResult };

export function getPreInfo() {
  return legacyRequest<PreInfo>('get', 'preInfo', null, 'form', { timeout: 5000 });
}

export function getBaseInfo() {
  return legacyRequest<DiceBaseInfo>('get', 'baseInfo', null, 'form', { timeout: 5000 });
}

export function getLogFetchAndClear() {
  return legacyRequest<SysLog[]>('get', 'log/fetchAndClear');
}

export function getHello() {
  return legacyRequest<unknown>('get', 'hello');
}

export function checkSecurity() {
  return legacyRequest<SecurityStatus>('get', 'checkSecurity');
}

export function postToolOnebot() {
  return legacyRequest<ToolOnebotResult>('post', '/tool/onebot');
}
