import request from "..";

export function getPreInfo() {
    return request<{
        testMode: boolean
      }>('get','preInfo',null,'form',{timeout:5000})
}

export function getBaseInfo() {
    return request<DiceBaseInfo>('get','baseInfo',null,'form',{timeout:5000})
}

export function getLogFetchAndClear() {
    return request<SysLog[]>('get','log/fetchAndClear')
}

export function getHello() {
    return request('get','hello')
}

export function checkSecurity() {
    return request<{isOk:boolean}>('get','checkSecurity')
}

export function postToolOnebot() {
    return request<{
        ok: boolean,
        ip: string,
        errText: string
      }>('post','/tool/onebot')
}
interface DiceBaseInfo {
    appChannel: string
    version: string
    versionSimple: string
    versionNew: string
    versionNewNote: string
    versionCode: number
    versionNewCode: number
    memoryAlloc: number
    memoryUsedSys: number
    uptime: number
    OS: string
    arch: string
    justForTest: boolean
    containerMode: boolean
    extraTitle?: string
  }

type SysLog={
    level: string;
    ts: number;
    caller: string;
    msg: string;
}