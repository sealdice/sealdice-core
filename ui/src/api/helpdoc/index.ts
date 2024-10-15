import type { HelpDoc, HelpTextItem, HelpTextItemQuery } from "~/type";
import { createRequest } from "..";
import type { UploadUserFile } from "element-plus";

const baseUrl = '/helpdoc/';
const request = createRequest(baseUrl);

export function getHelpDocTree() {
    return request<{ result: true, data: HelpDoc[] } | { result: false, err?: string }>('get', 'tree');
}

export function reloadHelpDoc() {
    return request<{ result: true } | { result: false, err?: string }>('post', 'reload');
}

export function uploadHelpDoc({files,group}:{files:UploadUserFile[],group:string}) {
    return request<{ result: true } | { result: false, err?: string }>('post', 'upload', {files:files.map(v =>v.raw),group},'formdata');
}

export function deleteHelpDoc(keys:string[]) {
    return request<{ result: true } | { result: false, err?: string }>('post', 'delete', {keys});
}

export function getHelpDocPage(param: HelpTextItemQuery) {
    return request<{ result: true; total: number; data: HelpTextItem[] } | { result: false; err?: string }>('post', 'textitem/get_page', 
        param
    );
}

export function getHelpDocConfig() {
    return request<{ aliases: { [key: string]: string[] } }>('get', 'config', );
}

export function postHelpDocConfig(param: { aliases: { [key: string]: string[] } }) {
    return request<{ result: true } | { result: false, err?: string }>('post', 'config', param);
}
