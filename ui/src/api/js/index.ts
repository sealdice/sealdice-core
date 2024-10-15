import type { JsPluginConfig, JsScriptInfo } from "~/type";
import { createRequest } from "..";
import type { UploadRawFile } from "element-plus";

const baseUrl = '/js/';
const request = createRequest(baseUrl);

export function getJsStatus() {
    return request<{ result: true, status: boolean } | {
        result: false, err: string
      }>('get', 'status');
}

export function getJsList() {
    return request<JsScriptInfo[]>('get', 'list');
}

export function getJsConfigs() {
    return request<{[key:string]: JsPluginConfig}>('get', 'get_configs');
}

export function setJsConfigs(config:{[key:string]: JsPluginConfig}) {
    return request('post', 'set_configs', config);
}

export function resetJsConfig(pluginName: string, key: string) {
    return request('post', 'reset_config', {pluginName,key});
}

export function deleteUnusedJsConfig(pluginName: string, key: string) {
    return request('post', 'delete_unused_config', {pluginName,key});
}

export function getJsRecord() {
    return request<{
        outputs: string[]
      }>('get', 'get_record');
}

export function uploadJs(file:UploadRawFile|Blob) {
    return request('post', 'upload', {file}, 'formdata');
}

export function deleteJs(index:number) {
    return request('post', 'delete', {index});
}

export function reloadJS() {
    return request('post', 'reload');
}

export function shutDownJS() {
    return request('post', 'shutdown');
}

export function executeJS(value: string) {
    return request<{
        ret: unknown,
        outputs: string[],
        err: string,
      }>('post', 'execute',{value});
}

export function enableJS(name:string) {
    return request<{result:boolean}>('post', 'enable',{name});
}

export function disableJS(name:string) {
    return request('post', 'disable',{name});
}

export function checkJsUpdate(index:number) {
    return request<{ result: false, err: string } | {
        result: true,
        old: string,
        new: string,
        tempFileName: string,
      }>('post', 'check_update', {index});
}

export function updateJs(tempFileName: string, index: number) {
    return request<{ result: false, err: string } | {
        result: true,
      }>('post', 'update', {tempFileName,index});
}



