<template>
  <el-space class="flex justify-between" alignment="flex-end">
    <h4>敏感词列表</h4>
    <el-space>
      <el-text v-if="filterCount > 0" size="small" type="info">已过滤 {{ filterCount }} 条</el-text>
      <el-input v-model="filter" size="small" clearable/>
    </el-space>
  </el-space>

  <main style="margin-top: 1rem;">
    <div class="w-full">
      <el-auto-resizer>
        <template #default="{ width }">
          <el-table-v2 class="w-full" :width="width" :height="430"
                       :columns="columns" :data="filteredWords">
          </el-table-v2>
        </template>
      </el-auto-resizer>
    </div>
  </main>
</template>
<script setup lang="tsx">
import {urlPrefix, useStore} from "~/store";
import {backend} from "~/backend";
import {useCensorStore} from "~/components/mod/censor/censor";
import type {Column} from "element-plus";
import type {CellRendererParams} from "element-plus/es/components/table-v2/src/types";
import {template} from "lodash-es";

const columns: Column<any>[] = [
  {
    key: "level",
    title: "级别",
    dataKey: 'level',
    width: 60,
    minWidth: 60,
    align: "center",
    cellRenderer: ({cellData: level}: CellRendererParams<number>) => {
      switch (level) {
        case 1:
          return <el-tag type="info" size="small" disable-transitions>提醒</el-tag>
        case 2:
          return <el-tag size="small" disable-transitions>注意</el-tag>
        case 3:
          return <el-tag type="warning" size="small" disable-transitions>警告</el-tag>
        case 4:
          return <el-tag type="danger" size="small" disable-transitions>危险</el-tag>
        default:
          return <el-tag type="info" size="small" disable-transitions>未知</el-tag>
      }
    },
  },
  {
    key: "related",
    title: "匹配词汇",
    dataKey: 'related',
    width: 800,
    cellRenderer: ({cellData: related, rowData}: CellRendererParams<SensitiveRelatedWord[]>) => {
      if (related) {
        return <el-space size="small" wrap>
          {related.map(word => <el-text key={word.word}>{word.word}</el-text>)}
        </el-space>
      } else {
        return <el-space>
          <el-text>{rowData.main}</el-text>
        </el-space>
      }
    },
  },
]

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

const words = ref<SensitiveWord[]>([])
const filter = ref<string>('')
const filterCount = computed(() => words.value.length - filteredWords.value.length)
const filteredWords = computed(() => words.value.filter(word => {
  if (filter.value === '') {
    return true
  }
  const val = filter.value.toLowerCase()
  return word.main.toLowerCase().includes(val)
      || word.related.map(w => w.word.toLowerCase().includes(val)).includes(true)
}))

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