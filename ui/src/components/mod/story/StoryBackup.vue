<template>
  <div class="tip">
    <el-text>
      每次向染色器上传跑团日志之前，都会在本地先保留一份备份，再进行上传。<br/>
      确定不再需要时，你可以在此处删除这些备份文件。<br/>
      <br/>
      <strong>删除此处的备份文件不会使日志丢失。</strong>
    </el-text>
  </div>

  <header style="display: flex; justify-content: space-between; align-items: center; margin: 1rem 0; padding: 0 1rem;">
    <el-space size="large" alignment="center">
      <el-checkbox
          v-model="checkAllBackups"
          :indeterminate="isIndeterminate"
          :disabled="!(backups && backups.length > 0)"
          @change="handleCheckAllChange">{{ checkAllBackups ? '取消全选' : '全选' }}
      </el-checkbox>
      <el-text type="info" size="small">已勾选 {{ selectedBackups.length }} 个备份，共
        {{ filesize(selectedBackups.map(bak => bak.fileSize).reduce((a, b) => a + b, 0)) }}
      </el-text>
    </el-space>
    <el-button type="danger" :disabled="!(selectedBackups && selectedBackups.length > 0)"
               @click="backupBatchDeleteConfirm">删除所选
    </el-button>
  </header>

  <main class="backup-list">
    <el-checkbox-group v-model="selectedBackups" @change="handleCheckedBackupChange">
      <div class="backup-line" v-for="backup in backups" :key="backup.name">
        <el-checkbox :label="backup" size="large">
          <template #default>{{ backup.name }}</template>
        </el-checkbox>
        <el-space size="small" wrap style="margin-left: 1px; justify-content: flex-end;">
          <el-button size="small" tag="a" style="text-decoration: none; width: 8rem;"
                     :href="`${urlBase}/sd-api/story/backup/download?name=${encodeURIComponent(backup.name)}&token=${encodeURIComponent(store.token)}`">
            下载 - {{ filesize(backup.fileSize) }}
          </el-button>
          <el-button type="danger" size="small" :icon="Delete" plain
                     @click="bakDeleteConfirm(backup.name)">删除
          </el-button>
        </el-space>
      </div>
    </el-checkbox-group>
  </main>
</template>

<script setup lang="ts">
import type {Backup} from "./story";
import type {CheckboxValueType} from "element-plus";
import {Delete} from "@element-plus/icons-vue";
import {filesize} from "filesize";
import {useStore} from "~/store";
import { getStoryBackUpList,postStoryBatchDel } from '~/api/story'
import {urlBase} from "~/backend";

const store = useStore()

const backups = ref<Backup[]>([])

const refreshList = async () => {
  let resp = await getStoryBackUpList();
  if (resp?.result) {
    backups.value = resp.data
  }
  selectedBackups.value = []
}

const bakDownloadConfirm = async (name: string) => {
  const res = await postStoryBatchDel([name])
  if (res?.result) {
    ElMessage.success('已删除')
  } else {
    ElMessage.error('删除失败')
  }
}

const bakDeleteConfirm = async (name: string) => {
  const ret = await ElMessageBox.confirm('确认删除？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  if (ret) {
    const res = await postStoryBatchDel([name])
    if (res?.result) {
      ElMessage.success('已删除')
    } else {
      ElMessage.error('删除失败')
    }
  }
  await refreshList()
}

const selectedBackups = ref<any[]>([])
const checkAllBackups = ref(false)
const isIndeterminate = ref(true)

const handleCheckAllChange = (val: CheckboxValueType) => {
  selectedBackups.value = val ? backups.value : []
  isIndeterminate.value = false
}

const handleCheckedBackupChange = (value: CheckboxValueType[]) => {
  const checkedCount = value.length
  checkAllBackups.value = checkedCount === backups.value.length
  isIndeterminate.value = checkedCount > 0 && checkedCount < backups.value.length
}

const backupBatchDeleteConfirm = async () => {
  const ret = await ElMessageBox.confirm('确认删除选择的所有跑团日志备份？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  if (ret) {
    const res = await postStoryBatchDel(selectedBackups.value.map(bak => bak.name))
    if (res.result) {
      ElMessage.success('已删除所选备份')
    } else {
      ElMessage.error('有备份删除失败！失败文件：\n' + res.fails.join("\n"))
    }
  }
  await refreshList()
}

onBeforeMount(async () => {
  await refreshList()
})

</script>

<style scoped lang="css">
.backup-list {
  display: flex;
  flex-direction: column;

  .backup-line {
    padding: 3px 1rem;
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
  }

  .backup-line:not(:first-child) {
    border-top: 1px solid var(--el-border-color);
  }
}
</style>