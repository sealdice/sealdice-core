<template>
  <div class="censor-log-container">
    <header class="censor-log-header">
      <el-button type="primary" :icon="Refresh" @click="refreshCensorLog">刷新</el-button>
      <el-pagination class="pagination" layout="sizes, prev, pager, next" background small
                     :current-page="logQuery.pageNum" :total="logQuery.total"
                     :pager-count="5" :default-page-size="20" :page-size="logQuery.pageSize"
                     @current-change="handleCurrentPageChange" @size-change="handlePageSizeChange"/>
    </header>
    <el-table style="margin-top: 1rem;" table-layout="auto" :data="logs">
      <el-table-column label="命中级别" width="60px">
        <template #default="scope">
          <el-tag v-if="scope.row.highestLevel === 1" type="info" size="small" disable-transitions>提醒</el-tag>
          <el-tag v-else-if="scope.row.highestLevel === 2" size="small" disable-transitions>注意</el-tag>
          <el-tag v-else-if="scope.row.highestLevel === 3" type="warning" size="small" disable-transitions>警告</el-tag>
          <el-tag v-else-if="scope.row.highestLevel === 4" type="danger" size="small" disable-transitions>危险</el-tag>
          <el-tag v-else type="info" size="small" disable-transitions>忽略</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="消息类型" width="60px">
        <template #default="scope">
          <el-text v-if="scope.row.msgType === 'private'">私聊</el-text>
          <el-text v-else-if="scope.row.msgType === 'group'">群</el-text>
          <el-text v-else>未知</el-text>
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
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import {useCensorStore} from "~/components/mod/censor/censor";
import {Refresh} from "@element-plus/icons-vue";
import { getCensorLogs } from "~/api/censor";

const censorStore = useCensorStore()

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
const logQuery = ref({
  pageNum: 1,
  pageSize: 10,
  total: 0,
})

censorStore.$subscribe(async (_, state) => {
  if (state.logsNeedRefresh === true) {
    await refreshCensorLog()
    state.logsNeedRefresh = false
  }
})

const handleCurrentPageChange = async (val: number) => {
  logQuery.value.pageNum = val
  await refreshCensorLog()
}

const handlePageSizeChange = async (val: number) => {
  logQuery.value.pageNum = 1
  logQuery.value.pageSize = val
  await refreshCensorLog()
}

const refreshCensorLog = async () => {
  const c: { result: false } | {
    result: true,
    data: CensorLog[],
    total: number
  } = await getCensorLogs(
      logQuery.value.pageNum,
      logQuery.value.pageSize
  )
  if (c?.result) {
    logs.value = c.data
    logQuery.value.total = c.total
  } else {
    ElMessage.error("无法获取拦截日志")
  }
}

onMounted(async () => {
  await refreshCensorLog()
})
</script>

<style scoped lang="css">
.censor-log-container {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.censor-log-header {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  text-align: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.pagination {
  background-color: #f3f5f7;
}
</style>