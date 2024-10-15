import { createRequest } from "..";

const baseUrl = '/censor/'
const request = createRequest(baseUrl)

export function getCensorStatus() {
    return request<{ result: false } | {
        result: true,
        enable: boolean,
        isLoading: boolean
    }>('get', 'status')
}

export function postCensorRestart(token: string) {
    return request<{ result: false } | {
        result: true,
        enable: boolean,
        isLoading: boolean
    }>('post', 'restart', { token })
}

export function postCensorStop(token: string) {
    return request<{ result: true } | {
        result: false,
        err: string
    }>('post', 'stop', { token })
}

export function getCensorConfig() {
    return request<{ result: false } | CensorConfig & { result: true }>('get', 'config')
}

export function postCensorConfig(modify: CensorConfig) {
    return request<{ result: true } | { result: false, err: string }>('post', 'config', modify)
}

export function uploadCensorFile(file: Blob) {
    return request<{ result: true } | { result: false, err: string }>('post', 'files/upload', { file }, 'formdata')
}

export function getCensorFiles() {
    return request<{ result: false } | {
        result: true,
        data: SensitiveWordFile[]
    }>('get', 'files')
}
export function deleteCensorFiles(keys: string[]) {
    return request<{ result: true } | { result: false, err: string }>
        ('delete', 'files', { keys })
}

export function getCensorLogs(pageNum: number, pageSize: number) {
    return request<{ result: false } | {
        result: true,
        data: CensorLog[],
        total: number
    }>('get', 'logs/page', { pageNum, pageSize })
}

export function getCensorWords() {
    return request<{ result: false } | {
        result: true,
        data: SensitiveWord[]
    }>('get', 'words')
}


interface CensorConfig {
    mode: number,
    caseSensitive: boolean
    matchPinyin: boolean
    filterRegex: string
    levelConfig: LeverConfig
}

type LeverConfig = {
    [key in 'notice' | 'caution' | 'warning' | 'danger']: {
        threshold: number;
        handlers: string[];
        score: number;
    };
};


interface SensitiveWordFile {
    key: string
    path: string,
    counter: number[]
}

interface CensorLog {
    id: number
    msgType: string
    userId: string
    groupId: string
    content: string
    highestLevel: string
    createAt: number
}
interface SensitiveWord {
    main: string
    level: number
    related: SensitiveRelatedWord[]
}
interface SensitiveRelatedWord {
    word: string
    reason: string
}