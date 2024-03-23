import {defineStore} from "pinia";
import {urlPrefix, useStore} from "~/store";
import {backend} from "~/backend";

export interface Backup {
    name: string,
    fileSize: number,
}

export const useStoryStore = defineStore("story", () => {
    const store = useStore()
    const token = store.token

    const backupList = async () => {
        const info: { result: false, err?: string } | {
            result: true,
            data: Backup[]
        } = await backend.get(urlPrefix + '/story/backup/list')
        return info
    }

    const backupDownload = async (name: string) => {
        const info = await backend.post(urlPrefix + '/story/backup/download', {}, {params: {name}})
        return info as any
    }

    const backupBatchDelete = async (names: string[]) => {
        const info: { result: true } | {
            result: false,
            fails: string[],
        } = await backend.post(urlPrefix + '/story/backup/batch_delete', {names}, {headers: {token}})
        return info
    }

    return {
        token,
        backupList,
        backupDownload,
        backupBatchDelete,
    }
})