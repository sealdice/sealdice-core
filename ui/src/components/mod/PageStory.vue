<script lang="ts" setup>
import { Delete, Select } from '@element-plus/icons-vue'
import { Ref, ref, onBeforeMount, computed } from 'vue'
import { useStore, urlPrefix } from '~/store'
import { apiFetch, backend } from '~/backend'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { ElMessage } from 'element-plus'

interface Log {
    id: number
    name: string
    groupId: string
    createdAt: number
    updatedAt: number
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

async function getLogs() {
    return apiFetch(url("logs"), {
        method: "get", headers: { token: token }
    })
}

async function getItems(v: Log) {
    // ofetch get+params 至少在开发模式有莫名奇妙的 bug ，会丢失 baseURL 
    // 下面的接口就先不更换了
    return await backend.get(url('items'), { params: v, headers: { token } }) as unknown as Item[]
}

async function delLog(v: Log) {
    return backend.delete(url('log'), { headers: { token }, data: v }) as unknown as boolean
}

async function uploadLog(v: Log) {
    return apiFetch(url("uploadLog"), {
        method: "post", body: v, headers: { token: token }
    })
}

//

let mode: Ref<'logs' | 'items'> = ref('logs');
let sum_log = ref(0), sum_item = ref(0);
dayjs.extend(relativeTime)

// logs ui
const logs_opt = [
    "日志名",
    "群号",
    "创建时间之前",
    "创建时间之后",
]
const logs_opt_val = ref(0)
let input_val = ref('')

let logs_data: Ref<Log[]> = ref([]);
const logs = computed(() => {
    let logs: Log[] = []

    for (const v of logs_data.value) {
        if (v.pitch === undefined) {
            v.pitch = false
        }
        if (input_val.value == '') {
            logs.push(v)
            continue
        }
        // 过滤
        if (logs_opt[logs_opt_val.value] == logs_opt[0]) {
            if (v.name.indexOf(input_val.value) > -1) logs.push(v);
            continue
        }
        if (logs_opt[logs_opt_val.value] == logs_opt[1]) {
            if (v.groupId.indexOf(input_val.value) > -1) logs.push(v);
            continue
        }
        let d1 = dayjs(input_val.value)
        if (!d1.isValid()) {
            console.debug('时间格式错误', input_val.value)
            break
        }
        let d2 = dayjs.unix(v.createdAt)
        if (logs_opt[logs_opt_val.value] == logs_opt[2]) {
            if (d2.isBefore(d1)) logs.push(v);
            continue
        }
        if (logs_opt[logs_opt_val.value] == logs_opt[3]) {
            if (d2.isAfter(d1)) logs.push(v);
            continue
        }
    }
    return logs
})

async function refreshLogs() {
    [sum_log.value, sum_item.value] = await getInfo()
    logs_data.value = await getLogs()
    ElMessage({
        message: '刷新日志列表完成',
        type: 'success',
    })
}

async function DelLog(v: Log, flag = true) {
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
            type: 'success',
        })
    }
}

function DelLogs() {
    let ls = []
    for (const v of logs_data.value) {
        if (v.pitch == true) {
            ls.push(v)
        }
    }
    for (const v of ls) {
        DelLog(v, false)
    }
    refreshLogs()
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
    item_data.value = await getItems(log)
    mode.value = 'items'
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
    <template v-if="mode == 'logs'">
        <ElCard>
            <div style="display: flex;justify-content: space-between;align-items: center;">
                <h4>跑团日志 / Story</h4>
                <ElButton @click="getLogs(); getInfo()">刷新日志列表</ElButton>
            </div>
            <span style="margin-right: 1rem;">记录过 {{ sum_log }} 份日志</span>
            <span style="margin-right: 1rem;">共计 {{ sum_item }} 条消息</span>
        </ElCard>
        <ElDivider></ElDivider>
        <ElInput style="width: 30rem;" :placeholder="logs_opt_val > 1 ? '时间格式：yyyy-mm-dd，如2023-08-16' : ''"
            v-model="input_val">
            <template #prepend>
                <ElSelect v-model="logs_opt_val" style="width: 10rem;">
                    <ElOption v-for="v, i in logs_opt" :label="v" :value="i" :key="i" />
                </ElSelect>
            </template>
        </ElInput>
        <ElButtonGroup style="margin-top: 5px;display: block;">
            <ElButton @click="logs.forEach(v => v.pitch = v.pitch ? false : true)" :icon="Select">全选</ElButton>
            <ElButton :icon="Delete" @click="DelLogs()"></ElButton>
        </ElButtonGroup>
        <template v-for="i in logs" :key="i.id">
            <ElCard style="margin-top: 10px;" shadow="hover">
                <span style="padding-right: 1rem;">日志名：{{ i.name }}</span>
                <ElCheckbox v-model="i.pitch" style="float: right;" />
                <br />
                <span>创建于： {{ i.groupId }}</span><br>
                <span style="padding-right: 1rem;">创建时：{{ dayjs.unix(i.createdAt).format('YYYY-MM-DD') }}</span>
                <ElTag size="small">{{ dayjs.unix(i.createdAt).fromNow() }}</ElTag><br />
                <span style="padding-right: 1rem;">更新时：{{ dayjs.unix(i.updatedAt).format('YYYY-MM-DD') }}</span>
                <ElTag size="small">{{ dayjs.unix(i.updatedAt).fromNow() }}</ElTag><br />
                <div style="margin-top: 1rem;">
                    <ElButton @click="openItem(i)">查看</ElButton>
                    <!--<ElButton>下载到本地</ElButton>-->
                    <ElButton @click="UploadLog(i)">上传到云端</ElButton>
                    <ElButton type="danger" @click="DelLog(i); console.log('---')">删除</ElButton>
                </div>
            </ElCard>
        </template>
    </template>
    <template v-if="mode == 'items'">
        <ElCard shadow="never">
            <div style="display: flex;justify-content: space-between;align-items: center;">
                <h4>跑团日志 / Story</h4>
                <ElButton @click="closeItem()">返回日志列表</ElButton>
            </div>
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
    </template>
</template>