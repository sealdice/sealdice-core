import { createRequest } from "..";

const baseUrl = '/backup/'
const request = createRequest(baseUrl)

export function getBackupList() {
    return request<{items:BackupInfo[]}>('get', 'list')
}

export function getBackupConfig() {
    return request<BackupConfig>('get', 'config_get')
}

export function setBackupConfig(data: BackupConfig) {
    return request('post', 'config_set', data)
}

export function postDoBackup(selection: number) {
    return request('post', 'do_backup', { selection })
}

export function postBackupDel(name: string) {
    return request<{ success: boolean }>('post', 'delete?name=' + name, { name })
}

export function postBackupBatchDel(names: string[]) {
    return request<{ result: true } | {
        result: false,
        fails: string[],
    }>('post', 'batch_delete', { names })
}

type BackupConfig = {
    autoBackupEnable: boolean,
    autoBackupTime: string,
    autoBackupSelection: number,
    backupCleanStrategy: number,
    backupCleanKeepCount: number,
    backupCleanKeepDur: string,
    backupCleanTrigger: number,
    backupCleanCron: string,
    autoBackupSelectionList: string[]
}
type BackupInfo = {
    name: string,
    fileSize: number,
    selection: number
}