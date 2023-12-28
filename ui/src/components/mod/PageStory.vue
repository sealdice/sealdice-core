<script lang="ts" setup>
import { Back, Delete, Select, Upload } from '@element-plus/icons-vue'
import { Ref, ref, onBeforeMount, computed } from 'vue'
import { useStore, urlPrefix } from '~/store'
import { apiFetch, backend } from '~/backend'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { ElMessage, ElMessageBox } from 'element-plus'

interface Log {
    id: number
    name: string
    groupId: string
    createdAt: number
    updatedAt: number
    size: number
    pitch?: boolean
    current?: number
}

interface Item {
    id: number
    logId: number
    nickname: string
    IMUserId: string
    time: number
    message: string
    isDice: boolean
    isEdit?: boolean
}

const store = useStore()
const token = store.token
const url = (p: string) => urlPrefix + "/story/" + p;

async function getInfo() {
    return apiFetch(url("info"), {
        method: "get"
    })
}

// async function getLogs() {
//     return apiFetch(url("logs"), {
//         method: "get", headers: { token: token }
//     })
// }

const queryLogPage = ref({
    pageNum: 1,
    pageSize: 20,
    name: "",
    groupId: "",
    createdTime: ([undefined, undefined] as unknown) as [Date, Date],
})

async function getLogPage(params: { pageNum: number, pageSize: number, name?: string, groupId?: string, createdTimeBegin?: number, createdTimeEnd?: number }) {
    return backend.get(url("logs/page"), {
        headers: { token: token }, params: params
    })
}

// async function getItems(v: Log) {
//     // ofetch get+params 至少在开发模式有莫名奇妙的 bug ，会丢失 baseURL
//     // 下面的接口就先不更换了
//     return await backend.get(url('items'), { params: v, headers: { token } }) as unknown as Item[]
// }

const logItemPage = ref({
    pageNum: 1,
    pageSize: 100,
    size: 0,
    logName: "",
    groupId: "",
})

async function getItemPage(params: { pageNum: number, pageSize: number, logName: string, groupId: string }) {
    return backend.get(url("items/page"), {
        headers: { token: token }, params: params
    })
}

async function delLog(v: Log) {
    return backend.delete(url('log'), { headers: { token }, data: v }) as unknown as boolean
}

async function uploadLog(v: Log) {
    await ElMessageBox.confirm(
        '将此跑团日志上传至海豹服务器？',
        '警告',
        {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        }
    );
    return apiFetch(url("uploadLog"), {
        method: "post", body: v, headers: { token: token }
    })
}

//

let mode: Ref<'logs' | 'items'> = ref('logs');
let sum_log = ref(0), sum_item = ref(0), cur_log = ref(0), cur_item = ref(0);
dayjs.extend(relativeTime)

let logs: Ref<Log[]> = ref([]);

async function searchLogs() {
    const params = {
        ...queryLogPage.value,
        createdTimeBegin: queryLogPage.value.createdTime?.[0] ? dayjs(queryLogPage.value.createdTime?.[0]).startOf('date').unix() : undefined,
        createdTimeEnd: queryLogPage.value.createdTime?.[1] ? dayjs(queryLogPage.value.createdTime?.[1]).endOf('date').unix() : undefined,
    }
    const page = await getLogPage(params)
    logs.value = page as unknown as Log[]
}

async function refreshLogs() {
    [sum_log.value, sum_item.value, cur_log.value, cur_item.value] = await getInfo()
    logs.value = await getLogPage(queryLogPage.value) as unknown as Log[] || []
    ElMessage({
        message: '刷新日志列表完成',
        type: 'success',
    })
}

const handleLogPageChange = async (val: number) => {
    queryLogPage.value.pageNum = val
    await refreshLogs()
}

async function DelLog(v: Log, flag = true) {
    await ElMessageBox.confirm(
        '是否删除此跑团日志？',
        '删除',
        {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        }
    ).then(async () => {
        let info = await delLog(v)
        if (info === true) {
            ElMessage({
                message: '删除成功',
                type: 'success',
            })
            if (flag) await refreshLogs();
        } else {
            ElMessage({
                message: '删除失败',
                type: 'error',
            })
        }
    })
}

async function DelLogs() {
    await ElMessageBox.confirm(
        '是否删除所选跑团日志？',
        '删除',
        {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        }
    ).then(async () => {
        let ls = []
        for (const v of logs.value) {
            if (v.pitch == true) {
                ls.push(v)
            }
        }
        for (const v of ls) {
            let info = await delLog(v)
            if (info === true) {
                ElMessage({
                    message: '删除成功',
                    type: 'success',
                })
            } else {
                ElMessage({
                    message: '删除失败',
                    type: 'error',
                })
            }
        }
            await refreshLogs()
        })
}

async function UploadLog(v: Log) {
    let info = await uploadLog(v) as string
    ElMessage({
        showClose: true,
        dangerouslyUseHTMLString: true,
        message: info,
        duration: 0,
    })
    return info
}
//

let item_data: Ref<Item[]> = ref([])

const users = ref({}) as Ref<Record<string, Array<string>>>

async function openItem(log: Log) {
    logItemPage.value.logName = log.name
    logItemPage.value.groupId = log.groupId
    logItemPage.value.size = log.size
    item_data.value = await getItemPage({
        pageNum: logItemPage.value.pageNum,
        pageSize: logItemPage.value.pageSize,
        logName: logItemPage.value.logName,
        groupId: logItemPage.value.groupId,
    }) as unknown as Item[]
    mode.value = 'items'
}

const handleItemPageChange = async (val: number) => {
    logItemPage.value.pageNum = val
    item_data.value = await getItemPage(logItemPage.value) as unknown as Item[]
}

function closeItem() {
    item_data.value = []
    mode.value = 'logs'
    users.value = {}
}

const items = computed(() => {
    let items: Item[] = []
    item_data.value.forEach(v => {
        if (!users.value[v.IMUserId]) {
            users.value[v.IMUserId] = [
                '#' + (Math.random() + 0.01).toString(16).substring(2, 8).toUpperCase(),
                v.nickname
            ]
        }
        items.push(v)
    })
    return items
})

//

onBeforeMount(() => {
    refreshLogs()
})

</script>

<template>
    <header style="margin-bottom: 1rem;">
        <!-- <ElButton type="primary" :icon="Refresh" @click="getLogs(); getInfo()">刷新日志列表</ElButton> -->
    </header>
    <template v-if="mode == 'logs'">
        <header>
            <ElCard>
                <template #header>
                    <strong style="display: block; margin: 10px 0;">跑团日志 / Story</strong>
                </template>
                <el-space direction="vertical" alignment="flex-start">
                    <el-text size="large" style="margin-right: 1rem;">记录过 {{ sum_log }} 份日志，共计 {{ sum_item }} 条消息</el-text>
                    <el-text size="large" style="margin-right: 1rem;">现有 {{ cur_log }} 份日志，共计 {{ cur_item }} 条消息</el-text>
                </el-space>
            </ElCard>
        </header>
        <ElDivider></ElDivider>
        <main>
            <el-form :inline="true" :model="queryLogPage">
                <el-form-item label="日志名">
                    <el-input v-model="queryLogPage.name" clearable />
                </el-form-item>
                <el-form-item label="群号">
                    <el-input v-model="queryLogPage.groupId" clearable />
                </el-form-item>
                <el-form-item label="创建时间">
                    <el-date-picker v-model="queryLogPage.createdTime" type="daterange" range-separator="-" />
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" @click="searchLogs">查询</el-button>
                </el-form-item>
            </el-form>
            <ElButtonGroup style="margin-top: 5px;display: block;">
                <ElButton type="primary" size="small" :icon="Select" @click="logs.forEach(v => v.pitch = !v.pitch)">全选
                </ElButton>
                <ElButton type="danger" size="small" :icon="Delete" @click="DelLogs()"
                    v-show="logs.filter(v => v.pitch).length > 0">删除所选</ElButton>
            </ElButtonGroup>
            <template v-for=" i in logs" :key="i.id">
                <ElCard style="margin-top: 10px;" shadow="hover">
                    <template #header>
                        <div style="display: flex; flex-wrap: wrap; gap: 1rem; justify-content: space-between;">
                            <el-space>
                                <ElCheckbox v-model="i.pitch" style="float: right;" />
                                <el-text size="large" tag="strong">{{ i.name }}</el-text>
                                <el-text>({{ i.groupId }})</el-text>
                            </el-space>
                            <el-space>
                                <ElButton size="small" plain @click="openItem(i)">查看</ElButton>
                                <!--<ElButton>下载到本地</ElButton>-->
                                <ElButton size="small" type="primary" :icon="Upload" plain @click="UploadLog(i)">提取日志
                                </ElButton>
                                <ElButton size="small" type="danger" :icon="Delete" plain
                                    @click="DelLog(i);">删除
                                </ElButton>
                            </el-space>
                        </div>
                    </template>
                    <el-space direction="vertical" alignment="flex-start">
                        <el-space>
                            <el-text>包含 {{ i.size }} 条消息</el-text>
                        </el-space>
                        <el-space>
                            <el-text>创建于：{{ dayjs.unix(i.createdAt).format('YYYY-MM-DD') }}</el-text>
                            <ElTag size="small" disable-transitions>{{ dayjs.unix(i.createdAt).fromNow() }}</ElTag><br />
                        </el-space>
                        <el-space>
                            <el-text>更新于：{{ dayjs.unix(i.updatedAt).format('YYYY-MM-DD') }}</el-text>
                            <ElTag size="small" disable-transitions>{{ dayjs.unix(i.updatedAt).fromNow() }}</ElTag><br />
                        </el-space>
                    </el-space>
                </ElCard>
            </template>
        </main>
        <div style="display: flex; justify-content: center;">
            <el-pagination class="pagination" :page-size="queryLogPage.pageSize" :current-page="queryLogPage.pageNum"
                :pager-count=5 :total="cur_log" @current-change="handleLogPageChange" layout="prev, pager, next" background
                hide-on-single-page />
        </div>
    </template>
    <template v-if="mode == 'items'">
        <ElCard shadow="never">
            <template #header>
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <strong style="margin: 10px 0;">跑团日志 / Story</strong>
                    <ElButton type="primary" :icon="Back" @click="closeItem()">返回列表</ElButton>
                </div>
            </template>
            <ElCollapse>
                <ElCollapseItem title="颜色设置">
                    <template v-for="(_, id) in users" :key="id">
                        <div style="padding: 0.5rem;">
                            <input type="color" v-model="users[id][0]" />
                            <span style="padding-left: 1rem;">{{ users[id][1] }}</span>
                        </div>
                    </template>
                </ElCollapseItem>
            </ElCollapse>
        </ElCard>
        <template v-for="v, i1 in items" :key="i1">
            <p :style="{ color: users[v.IMUserId][0] }">
                <span>{{ v.nickname }}：</span>
                <template v-for="p1, i2 in v.message.split('\n')" :key="i2">
                    <span>{{ p1 }}</span><br>
                </template>
            </p>
        </template>
        <div style="display: flex; justify-content: center;">
            <el-pagination class="pagination" :page-size="logItemPage.pageSize" :current-page="logItemPage.pageNum"
                :pager-count=5 :total="logItemPage.size" @current-change="handleItemPageChange" layout="prev, pager, next"
                background hide-on-single-page />
        </div>
    </template>
</template>

<style lang="scss">
.pagination {
    margin-top: 10px;
    background-color: #f3f5f7;
}
</style>
