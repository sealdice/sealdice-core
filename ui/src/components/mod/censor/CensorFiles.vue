<template>
  <h4>词库列表</h4>
  <header class="page-header">
    <el-upload action="" multiple accept="application/text,.txt,application/toml,.toml"
               :before-upload="beforeUpload">
      <el-button type="primary" :icon="Upload">导入</el-button>
    </el-upload>
    <el-space>
      <el-button style="text-decoration: none" type="success" tag="a" target="_blank" link size="small"
                 :href="`${urlBase}/sd-api/censor/files/template/toml`" :icon="Download">
        下载 toml 词库模板
      </el-button>
      <el-button style="text-decoration: none" type="success" tag="a" target="_blank" link size="small"
                 :href="`${urlBase}/sd-api/censor/files/template/txt`" :icon="Download">下载 txt 词库模板
      </el-button>
    </el-space>
  </header>
  <main style="margin-top: 1rem;">
    <el-table table-layout="auto" :data="files">
      <el-table-column fixed label="文件名" prop="name"></el-table-column>
      <el-table-column prop="count[1]">
        <template #header>
          <el-tag type="info" disable-transitions>提醒</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="count[2]">
        <template #header>
          <el-tag disable-transitions>注意</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="count[3]">
        <template #header>
          <el-tag type="warning" disable-transitions>警告</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="count[4]">
        <template #header>
          <el-tag type="danger" disable-transitions>危险</el-tag>
        </template>
      </el-table-column>
      <el-table-column fixed="right">
        <template #default="scope">
          <el-button size="small" type="danger" :icon="Delete" plain @click="deleteFile(scope.row.key)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </main>
</template>

<script setup lang="ts">
import {Delete, Download, Upload} from "@element-plus/icons-vue";
import {urlPrefix, useStore} from "~/store";
import {backend, urlBase} from "~/backend";
import {onBeforeMount, ref} from "vue";
import {useCensorStore} from "~/components/mod/censor/censor";
import {ElMessage, ElMessageBox, UploadUserFile} from "element-plus";

onBeforeMount(() => {
  refreshFiles()
})

const store = useStore()
const url = (p: string) => urlPrefix + "/censor/" + p;
const token = store.token
const censorStore = useCensorStore()

interface SensitiveWordFile {
  key: string
  path: string,
  counter: number[]
}

const files = ref<SensitiveWordFile[]>()

censorStore.$subscribe(async (_, state) => {
  if (state.filesNeedRefresh === true) {
    await refreshFiles()
    state.filesNeedRefresh = false
  }
})

const refreshFiles = async () => {
  const c: { result: false } | {
    result: true,
    data: SensitiveWordFile[]
  } = await backend.get(url("files"), {headers: {token}})
  if (c.result) {
    files.value = c.data
  }
}

const beforeUpload = async (file: UploadUserFile) => {
  let fd = new FormData()
  fd.append('file', file as unknown as Blob)

  const c = await censorStore.fileUpload({form: fd})
  if (c.result) {
    await refreshFiles()
    ElMessage.success('上传完成，请在全部操作完成后，手动重载拦截')
    censorStore.markReload()
  } else {
    ElMessage.error('上传失败！' + c.err)
  }
}

const deleteFile = async (key: string) => {
  await ElMessageBox.confirm(
      '是否删除此词库？',
      '删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
  ).then(async () => {
    const c: { result: true } | { result: false, err: string }
        = await backend.delete(url("files"), {
      headers: {token},
      data: {keys: [key]}
    })
    if (c.result) {
      ElMessage.success('删除词库完成，请在全部操作完成后，手动重载拦截')
      censorStore.markReload()
    } else {
      ElMessage.error('删除词库失败！' + c.err)
    }
  })
}

</script>
