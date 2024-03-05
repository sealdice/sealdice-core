<template>
  <header class="page-header">
    <el-button type="primary" :icon="Refresh" @click="doBackup">重载牌堆</el-button>
  </header>

  <el-tabs v-model="mode" :stretch="true">
    <el-tab-pane label="牌堆列表" name="list">
      <header class="deck-list-header">
        <el-space>
          <el-upload class="upload" action="" multiple :before-upload="beforeUpload" :file-list="fileList">
            <el-button type="primary" :icon="Upload">上传牌堆</el-button>
          </el-upload>
          <el-button class="link-button" type="info" :icon="Search" size="small" link tag="a" target="_blank"
            href="https://github.com/sealdice/draw">获取牌堆</el-button>
        </el-space>
        <el-space>
          <el-text type="info" size="small">目前支持 json/yaml/deck/toml 格式的牌堆</el-text>
          <el-tooltip raw-content>
            <template #content>
              deck牌堆: 一种单文件带图的牌堆格式<br />
              在牌堆文件中使用./images/xxx.png的相对路径引用图片。并连同图片目录一起打包成zip，修改扩展名为deck即可制作<br />
              <br />
              toml牌堆：海豹支持的新牌堆格式。格式更加友好，还提供了包括云牌组在内的更多功能支持。
            </template>
            <el-icon size="small"><question-filled /></el-icon>
          </el-tooltip>
        </el-space>
      </header>
      <main class="deck-list-main">
        <el-card class="deck-item" v-for="(i, index) in data" :key="index" shadow="hover">
          <template #header>
            <div class="deck-item-header">
              <el-space direction="vertical" alignment="normal">
                <el-space size="small" alignment="center">
                  <el-text size="large" tag="b">{{ i.name }}</el-text>
                  <el-text>{{ i.version }}</el-text>
                  <el-tag size="small" :type="i.fileFormat === 'toml' ? 'success' : 'primary'" disable-transitions>{{ i.fileFormat }}</el-tag>
                </el-space>
                <el-text v-if="i.cloud" type="primary" size="small">
                  <el-icon><MostlyCloudy /></el-icon>
                  作者提供云端内容，请自行鉴别安全性
                </el-text>
                <el-text v-if="i.fileFormat === 'jsonc'" type="warning" size="small">
                  <el-icon><Warning /></el-icon>
                  注意：该牌堆的格式并非标准 JSON ，而是允许尾逗号与注释语法的扩展 JSON
                </el-text>
              </el-space>
              <el-space>
                <el-popconfirm v-if="i.updateUrls && i.updateUrls.length > 0" width="220"
                               confirm-button-text="确认"
                               cancel-button-text="取消"
                               @confirm="doCheckUpdate(i, index)"
                               title="更新地址由牌堆作者提供，是否确认要检查该牌堆更新？">
                  <template #reference>
                    <el-button :icon="Download" type="success" size="small" plain :loading="diffLoading">更新</el-button>
                  </template>
                </el-popconfirm>
                <el-button :icon="Delete" type="danger" size="small" plain @click="doDelete(i, index)">删除</el-button>
              </el-space>
            </div>
          </template>
          <el-descriptions>
            <el-descriptions-item :span="3" label="作者">{{ i.author || '&lt;佚名>' }}</el-descriptions-item>
            <el-descriptions-item :span="3" v-if="i.desc" label="简介">{{ i.desc }}</el-descriptions-item>
            <el-descriptions-item :span="3" label="牌堆列表">
              <el-tag v-for="(visible, c) of i.command" :key="c" size="small" :type="visible ? 'primary' : 'info'" style="margin-right: 0.5rem;" disable-transitions>
                {{ c }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item v-if="i.license" label="许可协议">{{ i.license }}</el-descriptions-item>
            <el-descriptions-item v-if="i.date" label="发布时间">{{ i.date }}</el-descriptions-item>
            <el-descriptions-item v-if="i.updateDate" label="更新时间">{{ i.updateDate }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </main>
    </el-tab-pane>

    <el-dialog v-model="showDiff" title="牌堆内容对比" class="diff-dialog">
      <diff-viewer :lang="deckCheck.format" :old="deckCheck.old" :new="deckCheck.new"/>
      <template #footer>
        <el-space wrap>
          <el-button @click="showDiff = false">取消</el-button>
          <el-button v-if="!(deckCheck.old === deckCheck.new)" type="success" :icon="DocumentChecked" @click="deckUpdate">确认更新</el-button>
        </el-space>
      </template>
    </el-dialog>
  </el-tabs>
</template>

<script lang="ts" setup>
import { onBeforeMount, ref } from 'vue';
import { useStore } from '~/store'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  QuestionFilled,
  Upload,
  Download,
  Refresh,
  Search,
  Delete,
  MostlyCloudy,
  DocumentChecked,
  Warning,
} from '@element-plus/icons-vue'
import DiffViewer from "~/components/mod/diff-viewer.vue";

const store = useStore()

const mode = ref<string>('list')

const data = ref<any[]>([])

const cfg = ref<any>({})

const refreshList = async () => {
  const lst = await store.deckList()
  data.value = lst
}

const configGet = async () => {
  const data = await store.backupConfigGet()
  cfg.value = data
}

const fileList = ref<any[]>([])

const doBackup = async () => {
  const ret = await store.deckReload()
  if (ret.testMode) {
    ElMessage.success('展示模式无法重载牌堆')
  } else {
    ElMessage.success('已重载')
    await refreshList()
  }
}

const doDelete = async (data: any, index: number) => {
  ElMessageBox.confirm(
    `删除牌堆《${data.name}》，确定吗？`,
    '删除',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async (data) => {
    await store.deckDelete({ index })
    await store.deckReload()
    await refreshList()
    ElMessage.success('牌堆已删除')
  })
}

let lastSetEnable = 0
const setEnable = async (index: number, enable: boolean) => {
  const now = (new Date()).getTime()
  if (now - lastSetEnable < 100) return
  lastSetEnable = now
  const ret = await store.deckSetEnable({ index, enable })
  ElMessage.success('完成')
}

const doSave = async () => {
  await store.backupConfigSave(cfg.value)
  ElMessage.success('已保存')
}

const beforeUpload = async (file: any) => { // UploadRawFile
  let fd = new FormData()
  fd.append('file', file)
  await store.deckUpload({ form: fd })
  ElMessage.success('上传完成，即将自动重载牌堆')
  await store.deckReload()
  await refreshList()
}

onBeforeMount(async () => {
  await configGet()
  await refreshList()
})

const showDiff = ref<boolean>(false)
const diffLoading = ref<boolean>(false)

interface DeckCheckResult {
  old: string,
  new: string,
  format: 'json' | 'yaml' | 'toml',
  tempFileName: string,
  index: number,
}

const deckCheck = ref<DeckCheckResult>({
  old: "",
  new: "",
  format: "json",
  tempFileName: "",
  index: -1
})

const doCheckUpdate = async (data: any, index: number) => {
  diffLoading.value = true
  const checkResult = await store.deckCheckUpdate({ index });
  diffLoading.value = false
  if (checkResult.result) {
    deckCheck.value = { ...checkResult, index }
    showDiff.value = true
  } else {
    ElMessage.error('检查更新失败！' + checkResult.err)
  }
}

const deckUpdate = async () => {
  const res = await store.deckUpdate(deckCheck.value);
  if (res.result) {
    showDiff.value = false
    ElMessage.success('更新成功，即将自动重载牌堆')
    await store.deckReload()
    await refreshList()
  } else {
    showDiff.value = false
    ElMessage.error('更新失败！' + res.err)
  }
}

</script>

<style lang="scss">
@media screen and (max-width: 700px) {
  .bak-item {
    flex-direction: column;

    &>span {
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }
}

@media screen and (max-width: 700px){
  .diff-dialog {
    width: 90% !important;
  }
}

@media screen and (min-width: 700px) and (max-width: 900px){
  .diff-dialog {
    width: 80% !important;
  }
}

@media screen and (min-width: 900px) and (max-width: 1100px){
  .diff-dialog {
    width: 65% !important;
  }
}

@media screen and (min-width: 1100px){
  .diff-dialog {
    width: 50% !important;
  }
}

.deck-keys {
  display: flex;
  flex-flow: wrap;

  &>span {
    margin-right: 1rem;
    // width: fit-content;
  }
}

.deck-control {
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
}

.deck-list-header {
  margin-bottom: 1rem;
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}

.deck-list-main {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem
}

.deck-item-header {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}

.deck-item {
  width: 100%;
}

.edit-operation {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
}

.upload {
  >ul {
    display: none;
  }
}

.link-button {
  text-decoration: none;
}
</style>
