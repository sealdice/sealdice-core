import { createRequest } from "..";

const baseUrl = '/signin';
const request = createRequest(baseUrl);

export function getSalt() {
    return request<{salt:string}>('get', '/salt');
}

export function signin(password:string) {
    return request<{token:string}>('post', '', {password});
}
