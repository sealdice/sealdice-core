import { createRequest } from "..";

const baseUrl = '/utils/'
const request = createRequest(baseUrl)

export function getNewUtils() {
    return request<{ result: true, checked: boolean, news: string, newsMark: string } | { result: false, err?: string }>('get','news')
}

export function postUtilsCheckNews(newsMark: string) {
    return request<{ result: true; newsMark: string; } | { result: false }>('post','check_news',{newsMark})
}

export function postUtilsCheckCronExpr(expr: string) {
    return request('post','check_cron_expr',{expr})
}

export function getUtilsCheckNetWorkHealth() {
    return request<{
        result: true,
        total: number,
        ok: string[],
        timestamp: number
      } | { result: false }>('get','check_network_health')
}
