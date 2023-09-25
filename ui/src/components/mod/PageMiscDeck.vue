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
          <el-text type="info" size="small">目前支持 json/yaml/deck 格式的牌堆</el-text>
          <el-tooltip raw-content
            content="deck牌堆: 一种单文件带图的牌堆格式<br />在牌堆文件中使用./images/xxx.png的相对路径引用图片。并连同图片目录一起打包成zip，修改扩展名为deck即可制作">
            <el-icon size="small"><question-filled /></el-icon>
          </el-tooltip>
        </el-space>
      </header>
      <main class="deck-list-main">
        <el-card class="deck-item" v-for="i, index in data" :key="index" shadow="hover">
          <template #header>
            <div class="deck-item-header">
              <el-space>
                <el-text size="large" tag="b">{{ i.name }}</el-text>
                <el-text>{{ i.version }}</el-text>
              </el-space>
              <el-space>
                <!-- <el-button :icon="Download" type="success" size="small">更新</el-button> -->
                <!-- <el-button :icon="Tools" type="primary" size="small">设置</el-button> -->
                <el-button :icon="Delete" type="danger" size="small" plain @click="doDelete(i, index)">删除</el-button>
              </el-space>
            </div>
          </template>
          <el-descriptions>
            <el-descriptions-item :span="3" label="作者">{{ i.author || '<佚名>' }}</el-descriptions-item>
            <el-descriptions-item :span="3" v-if="i.desc" label="简介">{{ i.desc }}</el-descriptions-item>
            <el-descriptions-item :span="3" label="牌堆列表">
              <el-tag v-for="_, c of i.command" :key="c" size="small" style="margin-right: 0.5rem;"
                :disable-transitions="true">
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
  </el-tabs>
</template>

<script lang="ts" setup>
import { onBeforeMount, ref } from 'vue';
import { useStore } from '~/store'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  QuestionFilled,
  Upload,
  Refresh,
  Search,
  Delete
} from '@element-plus/icons-vue'

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
