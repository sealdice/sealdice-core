<template>
  <h4>敏感词列表</h4>
  <!--  <header>-->
  <!--    <el-button type="primary" :icon="Refresh" @click="refreshWords" plain>刷新列表</el-button>-->
  <!--  </header>-->
  <main style="margin-top: 1rem;">
    <el-table table-layout="auto" :data="words" :default-sort="{ prop: 'level', order: 'ascending' }">
      <el-table-column label="级别" width="80px">
        <template #default="scope">
          <el-tag v-if="scope.row.level === 1" type="info" size="small" disable-transitions>提醒</el-tag>
          <el-tag v-else-if="scope.row.level === 2" size="small" disable-transitions>注意</el-tag>
          <el-tag v-else-if="scope.row.level === 3" type="warning" size="small" disable-transitions>警告</el-tag>
          <el-tag v-else-if="scope.row.level === 4" type="danger" size="small" disable-transitions>危险</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="匹配词汇">
        <template #default="scope">
          <el-space v-if="scope.row.related" wrap>
            <el-text v-for="word of scope.row.related" :key="word.word">{{ word.word }}</el-text>
          </el-space>
          <el-space v-else>
            <el-text>{{ scope.row.main }}</el-text>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </main>
</template>
<script setup lang="ts">
import {urlPrefix, useStore} from "~/store";
import {backend} from "~/backend";
import {useCensorStore} from "~/components/mod/censor/censor";

onMounted(() => {
  refreshWords()
})

const url = (p: string) => urlPrefix + "/censor/" + p;
const censorStore = useCensorStore()
const store = useStore()
const token = store.token

interface SensitiveRelatedWord {
  word: string
  reason: string
}

interface SensitiveWord {
  main: string
  level: number
  related: SensitiveRelatedWord[]
}

const words = ref<SensitiveWord[]>()

censorStore.$subscribe(async (_, state) => {
  if (state.wordsNeedRefresh === true) {
    await refreshWords()
    state.wordsNeedRefresh = false
  }
})

const refreshWords = async () => {
  const c: { result: false } | {
    result: true,
    data: SensitiveWord[]
  } = await backend.get(url("words"), {headers: {token}})
  if (c.result) {
    words.value = c.data
  }
}
</script>