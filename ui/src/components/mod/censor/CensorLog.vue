<template>
  <header>
    <el-button type="primary" :icon="Refresh" @click="refreshCensorLog">刷新</el-button>
  </header>
  <el-table style="margin-top: 1rem;" table-layout="auto" :data="logs">
    <el-table-column label="命中级别" width="60px">
      <template #default="scope">
        <el-tag v-if="scope.row.highestLevel === 1" type="info" size="small" disable-transitions>提醒</el-tag>
        <el-tag v-else-if="scope.row.highestLevel === 2" size="small" disable-transitions>注意</el-tag>
        <el-tag v-else-if="scope.row.highestLevel === 3" type="warning" size="small" disable-transitions>警告</el-tag>
        <el-tag v-else-if="scope.row.highestLevel === 4" type="danger" size="small" disable-transitions>危险</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="消息类型" width="60px">
      <template #default="scope">
        <el-text v-if="scope.row.msgType === 'private'">私聊</el-text>
        <el-text v-else-if="scope.row.msgType === 'group'">群</el-text>
      </template>
    </el-table-column>
    <el-table-column label="用户" prop="userId"></el-table-column>
    <el-table-column label="群" prop="groupId"></el-table-column>
    <el-table-column label="内容" prop="content"></el-table-column>
    <el-table-column label="消息时间">
      <template #default="scope">
        {{ dayjs.unix(scope.row.createdAt).format('YYYY-MM-DD HH:mm:ss') }}
      </template>
    </el-table-column>
  </el-table>
</template>

<script lang="ts" setup>
import {onMounted, ref} from "vue";
import {backend} from "~/backend";
import {urlPrefix, useStore} from "~/store";
import dayjs from "dayjs";
import {useCensorStore} from "~/components/mod/censor/censor";
import {Refresh} from "@element-plus/icons-vue";

const url = (p: string) => urlPrefix + "/censor/" + p;
const censorStore = useCensorStore()
const store = useStore()
const token = store.token

interface CensorLog {
  id: number
  msgType: string
  userId: string
  groupId: string
  content: string
  highestLevel: string
  createAt: number
}

const logs = ref<CensorLog[]>()

censorStore.$subscribe(async (_, state) => {
  if (state.logsNeedRefresh === true) {
    await refreshCensorLog()
    state.logsNeedRefresh = false
  }
})

const refreshCensorLog = async () => {
  const c: { result: false } | {
    result: true,
    data: CensorLog[]
  } = await backend.get(url("logs/page"), {headers: {token}})
  if (c.result) {
    logs.value = c.data
  }
}

onMounted(async () => {
  await refreshCensorLog()
})
</script>