import { createRequest } from "..";

const baseUrl = '/configs/'
const request = createRequest(baseUrl)

export function saveCustomText(category: string, data: { [k: string]: string[][]; }) {
    return request('post', 'customText/save', { data, category })
}

export function getCustomText() {
    return request<CustomTexts>('get', 'customText')
}

export function saveCustomReply(data: CustomReply) {
    return request('post', 'custom_reply/save', data)
}

export function getCustomReply(filename: string) {
    return request<CustomReply>('get', 'custom_reply', {filename})
}

export function postCustomReplyNew(filename: string) {
    return request<{success: boolean}>('post', 'custom_reply/file_new', {filename})
}

export function postCustomReplyDel(filename: string) {
    return request<{success: boolean}>('post', 'custom_reply/file_delete', {filename})
}

export function getCustomReplyDownload(filename: string) {
    return request('get', 'custom_reply/file_download', {filename})
}

export function uploadCustomReply(file: Blob) {
    return request('post', 'custom_reply/file_upload', {file}, 'formdata')
}

export function getCustomReplyFileList() {
    return request<{items:ReplyFileInfo[]}>('get', 'custom_reply/file_list')
}

export function getCustomReplyDebug() {
    return request<{value: boolean}>('get', 'custom_reply/debug_mode')
}

export function postCustomReplyDebug(value: boolean) {
    return request<{value: boolean}>('post', 'custom_reply/debug_mode', {value})
}

type CustomTexts = {
    texts: {
        [k: string]: {
            [k: string]: string[][];
        };
    }
    helpInfo: {
        [k: string]: {
            [k: string]: {
                filename: string[];
                origin: (string[])[];
                vars: string[];
                modified: boolean;
                notBuiltin: boolean;
                topOrder: number;
                subType: string;
                extraText: string;
            }
        }
    }
    previewInfo: {
        [key: string]: {
            version: string;
            textV2: string;
            textV1: string;
            presetExists: boolean;
            errV1: string;
            errV2: string;
        }
    }
}

type CustomReply = {
    author: []
    conditions:ReplyCondition[]
    createTimestamp: number
    desc:string
    enable:false
    filename:string
    interval:number
    items:CustomReplyItem[]
    name:string
    updateTimestamp:number
    version:string
}
type CustomReplyItem = {
    conditions:ReplyCondition[]
    enable: boolean
    results: ReplyResult[]
}

type ReplyCondition = {
    condType: string
    matchType: string
    value: string
}

type ReplyResult= {
    resultType: string
    delay: number
    message: [string,number][]
}

type ReplyFileInfo = {
    enable: boolean
    filename: string
}