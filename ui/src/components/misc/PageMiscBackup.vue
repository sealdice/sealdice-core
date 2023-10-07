<template>
  <h2>设置</h2>
  <div>
    <div>
      <el-checkbox v-model="cfg.autoBackupEnable">开启自动备份</el-checkbox>
      <div>
        <span>备份间隔:
          <el-tooltip raw-content
                      content="备份间隔请参阅 <a href='https://pkg.go.dev/github.com/robfig/cron' target='_blank'>cron文档</a>">
            <el-icon><question-filled/></el-icon>
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

  <div style="display: flex; justify-content: space-between; align-items: center">
    <h2>已备份文件</h2>
    <el-button type="danger" :icon="Delete" @click="enterBatchDelete">进入批量删除页面</el-button>
  </div>

  <div size="small" direction="vertical" class="backup-list" fill>
    <div class="backup-line" v-for="i in data.items" :key="i.name" style="display: flex; justify-content: space-between;">
      <el-text size="large">{{ i.name }}</el-text>
      <el-space size="small" wrap style="margin-left: 1px; justify-content: flex-end;">
        <el-button size="small" tag="a" style="text-decoration: none; width: 8rem;"
                   :href="`${urlBase}/sd-api/backup/download?name=${encodeURIComponent(i.name)}&token=${encodeURIComponent(store.token)}`">
          下载 - {{ filesize(i.fileSize) }}
        </el-button>
        <el-button type="danger" size="small" :icon="Delete" plain
                   @click="bakDeleteConfirm(i.name)"></el-button>
      </el-space>
    </div>
  </div>

  <el-dialog v-model="showBatchDelete" title="批量删除备份" class="diff-dialog">
    <el-alert :closable="false" style="margin-bottom: 1.5rem;" title="默认勾选最近的 5 个备份之前的历史备份，可自行调整。"></el-alert>
    <el-space size="large" alignment="center" style="margin-bottom: 1rem;">
      <el-checkbox
          v-model="checkAllBaks"
          :indeterminate="isIndeterminate"
          @change="handleCheckAllChange">{{ checkAllBaks ? '取消全选' : '全选' }}</el-checkbox>
    <el-text type="info" size="small">已勾选 {{ selectedBaks.length }} 个备份，共 {{ filesize(selectedBaks.map(bak => bak.fileSize).reduce((a, b) => a + b, 0)) }}</el-text>
    </el-space>
    <el-checkbox-group v-model="selectedBaks" @change="handleCheckedBakChange">
      <div v-for="i of data.items" :key="i.name">
        <el-checkbox :label="i">
          <template #default>{{ i.name }}</template>
        </el-checkbox>
      </div>
    </el-checkbox-group>
    <template #footer>
      <el-space wrap>
        <el-button @click="showBatchDelete = false">取消</el-button>
        <el-button type="danger" :disabled="!(selectedBaks && selectedBaks.length > 0)"
                   @click="bakBatchDeleteConfirm">删除所选
        </el-button>
      </el-space>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import {computed, onBeforeMount, onBeforeUnmount, onMounted, ref} from 'vue';
import {useStore} from '~/store'
import {urlBase} from '~/backend'
import filesize from 'filesize'
import {ElMessage, ElMessageBox} from 'element-plus'
import {
  Location,
  Document,
  Menu as IconMenu,
  Setting,
  CirclePlusFilled,
  CircleClose,
  Delete,
  QuestionFilled,
  BrushFilled, DocumentChecked
} from '@element-plus/icons-vue'
import DiffViewer from "~/components/mod/diff-viewer.vue";

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

const bakDeleteConfirm = async (name: string) => {
  const ret = await ElMessageBox.confirm('确认删除？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  if (ret) {
    const r = await store.backupDelete(name)
    if (!r.success) {
      ElMessage.error('删除失败')
    } else {
      ElMessage.success('已删除')
    }
  }
  await refreshList()
}

const showBatchDelete = ref<boolean>(false)
const selectedBaks = ref<string[]>([])
const checkAllBaks = ref(false)
const isIndeterminate = ref(true)

const enterBatchDelete = async () => {
  selectedBaks.value = data.value.items.filter((_, index) => index >= 5)
  showBatchDelete.value = true
}

const handleCheckAllChange = (val: boolean) => {
  selectedBaks.value = val ? data.value.items : []
  isIndeterminate.value = false
}

const handleCheckedBakChange = (value: string[]) => {
  const checkedCount = value.length
  checkAllBaks.value = checkedCount === data.value.items.length
  isIndeterminate.value = checkedCount > 0 && checkedCount < data.value.items.length
}

const bakBatchDeleteConfirm = async () => {
  const ret = await ElMessageBox.confirm('确认删除所选备份？删除的内容无法找回！', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  })
  if (ret) {
    const res = await store.backupBatchDelete(selectedBaks.value.map(bak => bak.name))
    if (res.result) {
      ElMessage.success('已删除所选备份')
    } else {
      ElMessage.error('有备份删除失败！失败文件：\n' + res.fails.join("\n"))
    }
  }
  showBatchDelete.value = false
  await refreshList()
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
.backup-list {
  display: flex;
  flex-direction: column;

  .backup-line {
    padding: 5px 0;
  }

  .backup-line:not(:first-child) {
    border-top: 1px solid var(--el-border-color);
  }
}
</style>