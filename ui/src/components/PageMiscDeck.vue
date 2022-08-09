<template>
  <h2>设置</h2>
  <div>
    <div>
      <div style="margin-top: 1rem">
        <el-upload
          class="upload-demo"
          action=""
          multiple
          accept="application/json, .yaml, .yml, .deck"
          :before-upload="beforeUpload"
          :file-list="fileList"
        >
          <el-button type="">上传牌堆(json/yaml/deck)</el-button>
          <el-tooltip raw-content content="deck牌堆: 一种单文件带图的牌堆格式<br>在牌堆文件中使用./images/xxx.png的相对路径引用图片。并连同图片目录一起打包成zip，修改扩展名为deck即可制作">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>

          <template #tip>
            <div class="el-upload__tip">
            </div>
          </template>
        </el-upload>

        <!-- <el-button @click="doSave">上传牌堆(json/yaml/zip)</el-button> -->
        <el-button @click="doBackup">重新加载</el-button>
      </div>
    </div>
  </div>

  <h2>牌堆信息</h2>
  <div v-for="i,index in data" style="display: flex; flex-direction: column;" class="deck-item">
    <div style="display: flex; justify-content: space-between; align-content: center; align-items: center">
      <h4 style="flex: 1">{{ i.name }}</h4>
      <el-button @click="doDelete(i, index)">删除</el-button>
    </div>
    <!-- <div>
      <el-checkbox v-model="i.enable" @click.native="setEnable(index, !i.enable)">启用(此状态不保存，重载后重置)</el-checkbox>       
    </div> -->
    <div>作者: {{i.author || '<佚名>'}}  版本: {{i.version || '<未定义>'}}</div>
    <div>
      <div>牌组列表:</div>
      <div class="deck-keys">
        <span v-for="_,c of i.command">{{c}}</span>
      </div>
    </div>
    <!-- <div>{{i}}</div> -->
    <!-- <a :href="`${urlBase}/sd-api/backup/download?name=${encodeURIComponent(i.name)}&token=${encodeURIComponent(store.token)}`" style="text-decoration: none">
      <el-button style="width: 9rem;">下载 - {{ filesize(i.fileSize) }}</el-button>
    </a> -->
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, onBeforeUnmount, onMounted, ref } from 'vue';
import { useStore } from '~/store'
import { urlBase } from '~/backend'
import filesize from 'filesize'
import { ElMessage, ElMessageBox } from 'element-plus'
// import type { UploadProps, UploadUserFile } from 'element-plus'
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
import { UploadRawFile } from 'element-plus/lib/components/upload/src/upload';

const store = useStore()

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

const fileList = ref<any[]>([
])

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

const beforeUpload = async (file: UploadRawFile) => {
  let fd = new FormData()
  fd.append('file', file)
  await store.deckUpload({ form: fd })
  ElMessage.success('上传完成')
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

.deck-keys {
  display:flex;
  flex-flow: wrap;

  & > span {
    margin-right: 1rem;
    // width: fit-content;
  }
}
</style>