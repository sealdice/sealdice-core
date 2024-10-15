import type { UploadRawFile } from "element-plus";
import { createRequest } from "..";

const baseUrl = '/deck/';
const request = createRequest(baseUrl);

export function getDeckList() {
    return request<DeckConfig[]>('get', 'list');
}

export function reloadDeck() {
    return request<{testMode:boolean}>('post', 'reload');
}

export function enableDeck(index:number,enable:boolean) {
    return request('post', 'enable', {index,enable});
}

export function deleteDeck(index:number) {
    return request('post', 'delete', {index});
}

export function uploadDeck(file:Blob| UploadRawFile) {
    return request('post', 'upload', {file},'formdata');
}

export function checkDeckUpdate(index:number) {
    return request<{ result: false, err: string } | {
        result: true,
        old: string,
        new: string,
        format: 'json' | 'yaml' | 'toml',
        tempFileName: string,
      }>('post', 'check_update', {index});
}

export function updateDeck(index:number, tempFileName:string) {
    return request<{ result: false, err: string } | {
        result: true,
      }>('post', 'update', {index,tempFileName});
}


type DeckConfig = {
    enable: boolean;  // 是否启用该牌堆
    errText: string;  // 错误信息，如果为空则没有错误
    filename: string;  // 牌堆文件路径
    format: string;  // 牌堆格式
    formatVersion: number;  // 牌堆格式的版本
    fileFormat: string;  // 文件格式（如 "json"）
    name: string;  // 牌堆名称
    version: string;  // 牌堆版本
    author: string;  // 牌堆作者
    license: string;  // 牌堆许可信息（可能为空）
    command: {[key:string]: boolean};  // 牌堆命令的配置
    date: string;  // 牌堆的创建日期
    updateDate: string;  // 牌堆的更新日期（可能为空）
    desc: string;  // 牌堆描述
    updateUrls: string[] | null;  // 更新URL（可能为空）
    etag: string;  // 文件标签（可能为空）
    cloud: boolean;  // 是否为云端存储
};