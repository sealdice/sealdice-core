import {createRequest} from "..";

const baseUrl = '/story/'
const request = createRequest(baseUrl)
export async function getStoryInfo() {
    return await request<[number,number,number,number,]>('get','info')
}

export async function getStoryLogPage(pageParams:PageParams) {
    return await request<{ result: false, err?: string} | {
        result: true
        total: number,
        data: Log[],
        pageNum: number,
        pageSize: number,
    }>('get','logs/page',pageParams)
}

export async function getStoryItemPage(pageParams:PageParams) {
    return await request<Item[]>('get','items/page',pageParams)
}

export async function deleteStoryLog(log: Log) {
    return await request<boolean>('delete','log', log)
}

export async function postStoryLog(log: Log) {
    return await request('post','uploadLog', log)
}

export async function getStoryBackUpList() {
    return await request<{ result: false, err?: string } | {
        result: true,
        data: Backup[]
    }>('get','backup/list')
}

export async function getStoryBackUpDownload(name: string) {
    return await request('get','backup/download',{name})
}

export async function postStoryBatchDel(names: string[]) {
    return await request<{ result: true } | {result: false,fails: string[]}>('post','backup/batch_delete',{names})
}


type PageParams = {
    pageNum: number
    pageSize: number
    name ?: string
    groupId ?: string
    logName? :string
    createdTimeBegin ?: number
    createdTimeEnd ?: number 
}

interface Log {
    id: number
    name: string
    groupId: string
    createdAt: number
    updatedAt: number
    size: number
    pitch?: boolean
    current?: number
}

interface Backup {
    name: string,
    fileSize: number,
}

interface Item {
    id: number
    logId: number
    nickname: string
    IMUserId: string
    time: number
    message: string
    isDice: boolean
    isEdit?: boolean
}