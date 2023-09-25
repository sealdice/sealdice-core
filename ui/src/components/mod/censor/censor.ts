import {defineStore} from "pinia";
import {ref} from "vue";
import {backend} from "~/backend";
import {urlPrefix, useStore} from "~/store";

export const useCensorStore = defineStore("censor", () => {
    const store = useStore()
    const token = store.token

    const settingsNeedRefresh = ref<boolean>(false)
    const filesNeedRefresh = ref<boolean>(false)
    const wordsNeedRefresh = ref<boolean>(false)
    const logsNeedRefresh = ref<boolean>(false)

    const needReload = ref<boolean>(false)

    const markReload = () => {
        needReload.value = true
    }

    const reload = () => {
        needReload.value = false
        settingsNeedRefresh.value = true
        filesNeedRefresh.value = true
        wordsNeedRefresh.value = true
        logsNeedRefresh.value = true
    }

    const url = (p: string) => urlPrefix + "/censor/" + p;

    const getConfig = async (): Promise<{ result: false } | {
        result: true
        mode: number,
        caseSensitive: boolean
        matchPinyin: boolean
        filterRegex: string
        levelConfig: unknown
    }> => {
        return await backend.get(url("config"));
    }

    const saveConfig = async (modify: unknown): Promise<{ result: true } | { result: false, err: string }> => {
        return await backend.post(url("config"), modify, {headers: {token}});
    }

    const fileUpload = async ({form}: { form: FormData }): Promise<{ result: true } | {
        result: false,
        err: string
    }> => {
        return await backend.post(url("files/upload"), form, {headers: {token, "Content-Type": "multipart/form-data"}})
    }

    return {
        settingsNeedRefresh,
        filesNeedRefresh,
        wordsNeedRefresh,
        logsNeedRefresh,
        needReload,
        markReload,
        reload,
        getConfig,
        saveConfig,
        fileUpload,
    }
})