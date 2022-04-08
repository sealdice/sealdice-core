<template>
  <p>内存占用: {{filesize(store.curDice.baseInfo.memoryUsedSys || 0)}}</p>
  <el-table :data="store.curDice.logs" style="width: 100%;" class="hidden-xs-only">
    <el-table-column label="时间" width="130" >
      <template #default="scope">
        <div style="display: flex; align-items: center">
          <el-icon><timer /></el-icon>
          <span style="margin-left: 10px">{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="level" label="级别" width="100" />
    <el-table-column prop="msg" label="信息" />
  </el-table>

  <el-table :data="store.curDice.logs" style="width: 100%;" class="hidden-sm-and-up">
    <el-table-column label="时间" width="90" >
      <template #default="scope">
        <div style="display: flex; align-items: center">
          <!-- <el-icon><timer /></el-icon> -->
          <span>{{ dayjs.unix(scope.row.ts).format('HH:mm:ss') }}</span>
        </div>
      </template>
    </el-table-column>
    <!-- <el-table-column prop="level" label="级别" width="60" /> -->
    <el-table-column prop="msg" label="信息" />
  </el-table>

  <!-- <div v-for="i in store.curDice.logs">
    {{i}}
  </div> -->
</template>

<script lang="ts" setup>
import { Timer } from '@element-plus/icons-vue'
import { computed, onBeforeMount, onBeforeUnmount, onMounted } from 'vue';
import { useStore } from '~/store';
import * as dayjs from 'dayjs'
import filesize from 'filesize'

const store = useStore()


let timerId: number

onBeforeMount(async () => {
  await store.logFetchAndClear()

  timerId = setInterval(() => {
    store.logFetchAndClear()
  }, 5000) as any
})

onBeforeUnmount(() => {
  clearInterval(timerId)
})
</script>
