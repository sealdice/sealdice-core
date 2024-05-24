<template>
  <div style="display: flex; justify-content: space-between; align-items: center;">
    <h2>备份</h2>
    <div>
      <el-button type="success" :icon="DocumentChecked" @click="doSave">保存设置</el-button>
      <el-button type="primary" @click="doBackup">立即备份</el-button>
    </div>
  </div>
  <div>
    <el-form label-position="left">
    <h3>自动备份</h3>
    <el-checkbox v-model="cfg.autoBackupEnable">开启</el-checkbox>
    <div v-if="cfg.autoBackupEnable" style="margin-top: 1rem;">
      <el-form-item>
        <template #label>
          <span>备份间隔
            <el-tooltip raw-content
                        content="备份间隔表达式请参阅 <a href='https://pkg.go.dev/github.com/robfig/cron' target='_blank'>cron文档</a>">
              <el-icon><question-filled/></el-icon>
            </el-tooltip>
          </span>
        </template>
        <el-input v-model="cfg.autoBackupTime" style="width: 12rem;"></el-input>
      </el-form-item>
    </div>
    <h3>自动清理</h3>
      <el-form-item label="清理模式">
        <el-radio-group v-model="cfg.backupCleanStrategy">
          <el-radio-button :label="0">关闭</el-radio-button>
          <el-radio-button :label="1">保留一定数量</el-radio-button>
          <el-radio-button :label="2">保留一定时间内</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="保留数量" v-if="cfg.backupCleanStrategy === 1">
        <el-input-number v-model="cfg.backupCleanKeepCount" :min="1" :step="1"/>
      </el-form-item>
      <el-form-item v-if="cfg.backupCleanStrategy === 2">
        <template #label>
          <span>保留时间
            <el-tooltip>
              <template #content>
                请输入带时间单位的时间间隔。支持的时间单位只有 h m s（分别代表小时、分钟、秒）。<br />
                示例：<br />
                720h：代表保留 720 小时（即 30 天）内的备份<br />
                10.5h：代表保留 10.5 小时（即 10 小时 30 分）内的备份<br />
                10h30m：保留 10 小时 30 分内备份的另一种写法
              </template>
              <el-icon><question-filled/></el-icon>
            </el-tooltip>
          </span>
        </template>
        <el-input v-model="cfg.backupCleanKeepDur" style="width: 12rem;"/>
      </el-form-item>
      <el-form-item v-if="cfg.backupCleanStrategy !== 0">
        <template #label>
          <span>触发方式
            <el-tooltip raw-content
                        content="自动备份后：在每次自动备份完成后，顺便进行备份清理。<br/>定时：按照给定的 cron 表达式，单独触发清理。">
              <el-icon><question-filled/></el-icon>
            </el-tooltip>
          </span>
        </template>
        <el-checkbox-group v-model="backupCleanTriggers">
          <el-checkbox :label="CleanTrigger.AfterAutoBackup">自动备份后</el-checkbox>
          <el-checkbox :label="CleanTrigger.Cron">定时</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item v-if="cfg.backupCleanStrategy !== 0">
        <template #label>
          <span>定时间隔
            <el-tooltip raw-content
                        content="定时间隔表达式请参阅 <a href='https://pkg.go.dev/github.com/robfig/cron' target='_blank'>cron文档</a>">
              <el-icon><question-filled/></el-icon>
            </el-tooltip>
          </span>
        </template>
        <el-input v-model="cfg.backupCleanCron" style="width: 12rem"/>
      </el-form-item>
    </el-form>
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
import {computed, onBeforeMount, onBeforeUnmount, onMounted, ref, watch} from 'vue';
import {useStore} from '~/store'
import {urlBase} from '~/backend'
import {filesize} from 'filesize'
import {CheckboxValueType, ElMessage, ElMessageBox} from 'element-plus'
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
import DiffViewer from "~/components/utils/diff-viewer.vue";
import {sum} from "lodash-es";

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
  if (data.backupCleanTrigger) {
    let triggers: CleanTrigger[] = []
    if (data.backupCleanTrigger & CleanTrigger.Cron) {
      triggers.push(CleanTrigger.Cron)
    }
    if (data.backupCleanTrigger & CleanTrigger.AfterAutoBackup) {
      triggers.push(CleanTrigger.AfterAutoBackup)
    }
    backupCleanTriggers.value = triggers
  }
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
const selectedBaks = ref<any[]>([]) // 他不是string[]，是备份项的一种格式
const checkAllBaks = ref(false)
const isIndeterminate = ref(true)

const enterBatchDelete = async () => {
  selectedBaks.value = data.value.items.filter((_, index) => index >= 5)
  showBatchDelete.value = true
}

const handleCheckAllChange = (val: CheckboxValueType) => {
  selectedBaks.value = val ? data.value.items : []
  isIndeterminate.value = false
}

const handleCheckedBakChange = (value: CheckboxValueType[]) => {
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

const enum CleanTrigger {
  // 定时
  Cron = 1 << 0,
  // 自动备份后
  AfterAutoBackup = 1 << 1,
}

const backupCleanTriggers = ref<CleanTrigger[]>()

watch(backupCleanTriggers, (newStrategies) => {
  cfg.value.backupCleanTrigger = sum(newStrategies)
})

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