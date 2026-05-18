export interface PreInfo {
  testMode: boolean;
}

export interface DiceBaseInfo {
  appChannel: string;
  version: string;
  versionSimple: string;
  versionNew: string;
  versionNewNote: string;
  versionCode: number;
  versionNewCode: number;
  memoryAlloc: number;
  memoryUsedSys: number;
  uptime: number;
  OS: string;
  arch: string;
  justForTest: boolean;
  containerMode: boolean;
  extraTitle?: string;
}

export interface SysLog {
  level: string;
  ts: number;
  caller: string;
  msg: string;
}

export interface SecurityStatus {
  isOk: boolean;
}

export interface ToolOnebotResult {
  ok: boolean;
  ip: string;
  errText: string;
}
