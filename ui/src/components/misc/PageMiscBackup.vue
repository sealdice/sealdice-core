<template>
  <div style="display: flex; justify-content: space-between; align-items: center;">
    <h2>备份</h2>
    <div>
      <el-button type="success" :icon="DocumentChecked" @click="doSave">保存设置</el-button>
      <el-button type="primary" @click="showBackup = true">立即备份</el-button>
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
      <el-form-item label="备份范围">
        <el-checkbox-group v-model="cfg.autoBackupSelectionList">
          <el-checkbox label="基础（含自定义回复）" value="base" checked disabled/>
          <el-checkbox label="JS 插件" value="js"/>
          <el-checkbox label="牌堆" value="deck"/>
          <el-checkbox label="帮助文档" value="helpdoc"/>
          <el-checkbox label="敏感词库" value="censor"/>
          <el-checkbox label="人名信息" value="name"/>
          <el-checkbox label="图片" value="image"/>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item label="备份文件名预览">
        <el-text type="info">bak_{{ now }}_auto_r{{ cfg.autoBackupSelection.toString(16) }}_&lt;随机值&gt;.zip</el-text>
      </el-form-item>
    </div>
    <h3>自动清理</h3>
      <el-form-item label="清理模式">
        <el-radio-group v-model="cfg.backupCleanStrategy">
          <el-radio-button :value="0">关闭</el-radio-button>
          <el-radio-button :value="1">保留一定数量</el-radio-button>
          <el-radio-button :value="2">保留一定时间内</el-radio-button>
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
    <div class="backup-line flex flex-wrap justify-between gap-2" v-for="i in data.items" :key="i.name">
      <div class="flex flex-col">
        <el-text class="self-start" size="large">{{ i.name }}</el-text>
        <el-text class="self-start" v-if="(i?.selection ?? 0) >= 0" size="small" type="info">此备份包含：{{ parseSelectionDesc(i.selection).join('、') }}</el-text>
        <el-text class="self-start" v-else size="small" type="warning">此备份内容无法识别</el-text>
      </div>
      <el-space size="small" wrap class="justify-end">
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

  <el-dialog v-model="showBackup" title="立即备份" class="diff-dialog">
    <el-space direction="vertical" alignment="flex-start">
      <div>
        <span>备份范围：</span>
        <el-checkbox-group v-model="backupSelections">
          <el-checkbox label="基础（含自定义回复）" value="base" checked disabled/>
          <el-checkbox label="JS 插件" value="js"/>
          <el-checkbox label="牌堆" value="deck"/>
          <el-checkbox label="帮助文档" value="helpdoc"/>
          <el-checkbox label="敏感词库" value="censor"/>
          <el-checkbox label="人名信息" value="name"/>
          <el-checkbox label="图片" value="image"/>
        </el-checkbox-group>
      </div>
      <div class="flex flex-wrap">
        <span>备份文件名预览：</span>
        <el-text type="info">bak_{{ now }}_r{{ formatSelection(backupSelections).toString(16) }}_&lt;随机值&gt;.zip</el-text>
      </div>
    </el-space>
    <template #footer>
      <el-space wrap>
        <el-button @click="showBackup = false">取消</el-button>
        <el-button type="primary" @click="doBackup">立即备份</el-button>
      </el-space>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import type {CheckboxValueType} from "element-plus";
import {useStore} from '~/store'
import {urlBase} from '~/backend'
import {filesize} from 'filesize'
import {
  Delete,
  QuestionFilled,
  DocumentChecked
} from '@element-plus/icons-vue'
import {sum} from "lodash-es";
import { dayjs } from 'element-plus'

const store = useStore()

const data = ref<{
  items: any[]
}>({
  items: []
})

const cfg = ref<any>({})
const now = ref(dayjs().format('YYMMDD_HHmmss'))
const showBackup = ref<boolean>(false)
const backupSelections = ref<string[]>(['base', 'js', 'deck', 'helpdoc', 'censor', 'name', 'image'])

const parseSelection = (selection: number): string[] => {
  const list = ['base']
  const jsMark = selection & 0b000001
  if (jsMark) {
    list.push('js')
  }
  const deckMark = selection & 0b000010
  if (deckMark) {
    list.push('deck')
  }
  const helpdocMark = selection & 0b000100
  if (helpdocMark) {
    list.push('helpdoc')
  }
  const censorMark = selection & 0b001000
  if (censorMark) {
    list.push('censor')
  }
  const nameMark = selection & 0b010000
  if (nameMark) {
    list.push('name')
  }
  const resourceMark = selection & 0b100000
  if (resourceMark) {
    list.push('image')
  }
  return list
}

const parseSelectionDesc = (selection: number): string[] => {
  const list = ['基础']
  const jsMark = selection & 0b000001
  if (jsMark) {
    list.push('JS 插件')
  }
  const deckMark = selection & 0b000010
  if (deckMark) {
    list.push('牌堆')
  }
  const helpdocMark = selection & 0b000100
  if (helpdocMark) {
    list.push('帮助文档')
  }
  const censorMark = selection & 0b001000
  if (censorMark) {
    list.push('敏感词库')
  }
  const nameMark = selection & 0b010000
  if (nameMark) {
    list.push('人名信息')
  }
  const resourceMark = selection & 0b100000
  if (resourceMark) {
    list.push('图片')
  }
  return list
}

const formatSelection = (selections: string[]): number => {
  let mark = 0
  if (selections.includes('js')) {
    mark |= 0b000001
  }
  if (selections.includes('deck')) {
    mark |= 0b000010
  }
  if (selections.includes('helpdoc')) {
    mark |= 0b000100
  }
  if (selections.includes('censor')) {
    mark |= 0b001000
  }
  if (selections.includes('name')) {
    mark |= 0b010000
  }
  if (selections.includes('image')) {
    mark |= 0b100000
  }
  return mark
}

watch(() => cfg.value.autoBackupSelectionList, (v) => {
  cfg.value.autoBackupSelection = formatSelection(v)
})

const refreshList = async () => {
  const lst = await store.backupList()
  data.value = lst
}

const configGet = async () => {
  const data = await store.backupConfigGet()
  cfg.value = data
  cfg.value.autoBackupSelectionList = parseSelection(data.autoBackupSelection)
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
  const ret = await store.backupDoSimple({
    selection: formatSelection(backupSelections.value)
  })
  showBackup.value = false
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

const refreshNow = async () => {
    now.value = dayjs().format('YYMMDD_HHmmss');
    await setTimeout(refreshNow, 1000);
 };

onBeforeMount(async () => {
  await configGet()
  await refreshList()
  await refreshNow()
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