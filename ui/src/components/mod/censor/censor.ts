import {defineStore} from "pinia";
import { postCensorRestart, postCensorStop } from "~/api/censor";
import { useStore} from "~/store";

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

    const restartCensor = async (): Promise<{ result: false } | {
        result: true,
        enable: boolean,
        isLoading: boolean
    }> => {
        return await postCensorRestart(token)
    }

    const stopCensor = async (): Promise<{ result: true } | {
        result: false,
        err: string
    }> => {
        return await postCensorStop(token);
    }

    return {
        settingsNeedRefresh,
        filesNeedRefresh,
        wordsNeedRefresh,
        logsNeedRefresh,
        needReload,
        markReload,
        reload,
        restartCensor,
        stopCensor,
    }
})