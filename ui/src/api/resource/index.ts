import type { Resource, ResourceType } from "~/store";
import { createRequest } from "..";
import type { UploadRawFile } from "element-plus";

const baseUrl = '/resource';
const request = createRequest(baseUrl);

export function getResourcePage(type: ResourceType) {
    return request<{ result: false, err: string } | {
        result: true,
        total?: number,
        data: Resource[]
      }>('get', '/page',{type});
}

export function createResource(files:UploadRawFile|Blob) {
    return request<{ result: false, err: string } | { result: true }>('post', '', {files},'formdata');
}

export function deleteResource(path: string) {
    return request<{ result: false, err: string } | { result: true }>('delete', '',{path});
}

export function getResourceData(path: string, thumbnail: boolean = false) {
    return request<Blob>('get', '/data',{ path, thumbnail });
}
