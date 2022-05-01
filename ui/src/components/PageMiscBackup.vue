<template>
  <h2>设置</h2>
  <div>
    <div>
      <el-checkbox v-model="cfg.autoBackupEnable">开启自动备份</el-checkbox>
      <div>
        <span>备份间隔:
          <el-tooltip raw-content content="备份间隔请参阅 <a href='https://pkg.go.dev/github.com/robfig/cron' target='_blank'>cron文档</a>">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </span>
        <el-input v-model="cfg.autoBackupTime" style="width: 12rem"></el-input>
      </div>
      <div style="margin-top: 1rem">
        <el-button @click="doSave">保存设置</el-button>
        <el-button @click="doBackup">立即备份</el-button>
      </div>
    </div>
    <h4>如何恢复备份？</h4>
    <div>将骰子彻底关闭，解压备份压缩包到骰子目录。若提示“是否覆盖？”选择“全部”即可(覆盖data目录)。</div>
  </div>

  <h2>已备份文件</h2>
  <div v-for="i in data.items" style="display: flex;" class="bak-item">
    <span style="flex: 1">{{ i.name }}</span>
    <a :href="`${urlBase}/backup/download?name=${encodeURIComponent(i.name)}&token=${encodeURIComponent(store.token)}`" style="text-decoration: none">
      <el-button style="width: 9rem;">下载 - {{ filesize(i.fileSize) }}</el-button>
    </a>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref } from 'vue';
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import filesize from 'filesize'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Location,
  Document,
  Menu as IconMenu,
  Setting,
  CirclePlusFilled,
  CircleClose,
  QuestionFilled,
  BrushFilled
} from '@element-plus/icons-vue'

const store = useStore()

const data = ref<{
  items: any[]
}>({
  items: []
})

const cfg = ref<any>({})

const refreshList = async () => {
  const lst = await store.backupList()
  data.value = lst
}

const configGet = async () => {
  const data = await store.backupConfigGet()
  cfg.value = data
}

const doBackup = async () => {
  const ret = await store.backupDoSimple()
  await refreshList()
  if (ret.testMode) {
    ElMessage.success('展示模式无法备份')
  } else {
    ElMessage.success('已进行备份')
  }
}

const doSave = async () => {
  await store.backupConfigSave(cfg.value)
  ElMessage.success('已保存')
}

onBeforeMount(async () => {
  await configGet()
  await refreshList()
})
</script>

<style lang="scss">
@media screen and (max-width: 700px) {
  .bak-item {
    flex-direction: column;
    & > span {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }
}
</style>