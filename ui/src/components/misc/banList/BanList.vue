<script setup lang="ts">
import type { UploadUserFile } from "element-plus";
import {
  Download,
  Delete,
  Plus,
  Upload,
} from '@element-plus/icons-vue'
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { getBanConfigList, importBanConfig, postMapAddOne, postMapDelOne } from "~/api/banconfig";

dayjs.extend(relativeTime)

const store = useStore()

const recordList = ref<any[]>([])
const recordPage = ref({
  no: 1,
  total: 0,
  pageSize: 10,
})

const showBanned = ref(true)
const showWarn = ref(true)
const showTrusted = ref(true)
const showOthers = ref(true)
const dialogAddShow = ref(false)

const banRankText = new Map<number, string>()
banRankText.set(-30, '禁止')
banRankText.set(-10, '警告')
banRankText.set(+30, '信任')
banRankText.set(0, '常规')

const addData = ref<{ id: string, rank: number, name:string, reason: string }>({
  id: '',
  rank: -30,
  reason: '',
  name: ''
});

const doAdd = async () => {
  if (addData.value.id === '') return
  await postMapAddOne(addData.value.id, addData.value.rank, addData.value.name, addData.value.reason)
  await refreshList()
  ElMessage.success('已保存')
  dialogAddShow.value = false
}

const searchBy = ref('')

const filteredRecordList = computed<any[]>(() => {
  if (recordList.value) {
    let items = []
    for (let [k, _v] of Object.entries(recordList.value)) {
      const v = _v as any
      let ok = false
      if (v.rank === -30 && showBanned.value) {
        ok = true
      }
      if (v.rank === -10 && showWarn.value) {
        ok = true
      }
      if (v.rank === 30 && showTrusted.value) {
        ok = true
      }
      if (v.rank === 0 && showOthers.value) {
        ok = true
      }

      if (ok && searchBy.value !== '') {
        let a = false
        let b = false
        if (v.ID.indexOf(searchBy.value) !== -1) {
          a = true
        }
        if (v.name.indexOf(searchBy.value) !== -1) {
          b = true
        }
        ok = a || b
      }

      v.rankText = banRankText.get(v.rank)

      if (ok) items.push(v)
    }

    return items
  }
  return []
})

const groupItems = computed(() => {
  const start = (recordPage.value.no - 1) * recordPage.value.pageSize;
  const end = start + recordPage.value.pageSize;
  return filteredRecordList.value.slice(start, end);
})

const refreshList = async () => {
  const lst = await getBanConfigList()
  recordList.value = lst
  recordPage.value.total = lst.length
  recordPage.value.no = 1
}

const deleteOne = async (i: any, index: number) => {
  const res = await ElMessageBox.confirm(
      '是否删除此记录？',
      '删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
  );
  if (res) {
    await postMapDelOne(i)
    await refreshList()
    ElMessage.success('已保存')
  }
}

const beforeUpload = async (file: UploadUserFile) => {

  const c = await importBanConfig(file)
  if (c.result) {
    ElMessage.success('导入黑白名单完成')
    await nextTick(async () => {
      await refreshList()
    })
  } else {
    ElMessage.error('导入黑白名单失败！' + c.err)
  }
}

onBeforeMount(async () => {
  await refreshList()
})
</script>

<template>
  <header class="flex flex-wrap-reverse gap-y-4 justify-between">
    <el-space>
      <el-text size="large">搜索：</el-text>
      <el-input v-model="searchBy" class="w-64" placeholder="请输入帐号或名字的一部分"></el-input>
    </el-space>

    <el-space>
      <el-button type="success" :icon="Plus" @click="dialogAddShow = true">添加</el-button>
      <el-upload action="" multiple accept="application/json,.json" :show-file-list="false"
                 :before-upload="beforeUpload" style="display: flex; align-items: center;">
        <el-button type="success" :icon="Upload" plain>导入</el-button>
      </el-upload>
      <el-button type="primary" :icon="Download" plain tag="a" target="_blank"
                 :href="`${urlBase}/sd-api/banconfig/export`" style="text-decoration: none;">
        导出
      </el-button>
    </el-space>
  </header>

  <el-space class="my-2">
    <el-text size="large">级别：</el-text>
    <el-checkbox v-model="showBanned">拉黑</el-checkbox>
    <el-checkbox v-model="showWarn">警告</el-checkbox>
    <el-checkbox v-model="showTrusted">信任</el-checkbox>
    <el-checkbox v-model="showOthers">其它</el-checkbox>
  </el-space>

  <main class="mt-4">
    <el-space :fill="true" class="w-full">
      <el-card v-for="(i, index) in groupItems" :key="i.ID" shadow="hover" class="w-full">
        <template #header>
          <div class="flex flex-wrap gap-4 justify-between">
            <el-space alignment="center">
              <el-tag v-if="i.rankText === '禁止'" type="danger" disable-transitions>{{ i.rankText }}</el-tag>
              <el-tag v-else-if="i.rankText === '警告'" type="warning" disable-transitions>{{ i.rankText }}</el-tag>
              <el-tag v-else-if="i.rankText === '信任'" type="success" disable-transitions>{{ i.rankText }}</el-tag>
              <el-tag v-else disable-transitions>{{ i.rankText }}</el-tag>
              <el-space size="small" alignment="center" wrap>
                <el-text size="large" tag="strong">{{ i.ID }}</el-text>
                <el-text>「{{ i.name }}」</el-text>
                <el-text size="small" tag="em">怒气值：{{ i.score }}</el-text>
              </el-space>
            </el-space>
            <el-space>
              <el-button :icon="Delete" type="danger" size="small" plain @click="deleteOne(i, index)">删除</el-button>
            </el-space>
          </div>
        </template>
        <el-space style="display: block;" direction="vertical">
          <div v-for="(j, index) in i.reasons" :key="index">
            <el-space size="small" wrap>
              <el-tooltip raw-content :content="dayjs.unix(i.times[index]).format('YYYY-MM-DD HH:mm:ssZ[Z]')">
                <el-tag size="small" type="info" disable-transitions>{{ dayjs.unix(i.times[index]).fromNow() }}</el-tag>
              </el-tooltip>
              <el-text>在&lt;{{ i.places[index] }}>，原因：「{{j}}」</el-text>
            </el-space>
          </div>
        </el-space>
      </el-card>
    </el-space>
  </main>

  <footer class="mt-4 flex justify-center">
    <el-pagination class="bg-[#f3f5f7]" type="small" :pager-count=3
                   layout="prev, pager, next, total" background hide-on-single-page
                   v-model:current-page="recordPage.no"
                   v-model:page-size="recordPage.pageSize"
                   :total="filteredRecordList.length"/>
  </footer>

  <el-dialog v-model="dialogAddShow" title="添加用户/群组" :close-on-click-modal="false" :close-on-press-escape="false" :show-close="false" class="the-dialog">
    <el-form label-width="6rem">
      <el-form-item label="用户ID" required>
        <el-input v-model="addData.id" placeholder="必须为 QQ:12345 或 QQ-Group:12345 格式"></el-input>
      </el-form-item>
      <el-form-item label="名称">
        <el-input v-model="addData.name" placeholder="自动"></el-input>
      </el-form-item>
      <el-form-item label="原因">
        <el-input v-model="addData.reason" placeholder="骰主后台设置"></el-input>
      </el-form-item>
      <el-form-item label="身份">
        <el-radio-group v-model="addData.rank">
          <el-radio
              v-for="item in [{'label': '禁用', value: -30}, {'label': '信任', value: 30}]"
              :key="item.value"
              :label="item.label"
              :value="item.value"
          />
        </el-radio-group>
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
          <el-button @click="dialogAddShow = false">取消</el-button>
          <el-button type="success" @click="doAdd">添加</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<style scoped lang="css">

</style>