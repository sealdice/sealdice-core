<script setup lang="ts">
import type {Resource} from "~/store";
import {useStore} from "~/store";
import {filesize} from "filesize";
import {urlBase} from "~/backend";
import {CopyDocument, Delete, Download, Search, Upload} from "@element-plus/icons-vue";
import ClipboardJS from 'clipboard'

const store = useStore();

const loading = ref(true);
const images = ref<Resource[]>([]);

const drawer = ref(false)
const currentResource = ref<Resource>({} as Resource)

const fileList = ref<any[]>([])

const refreshResources = async () => {
  loading.value = true
  const imagesRes = await store.resourceList("image");
  if (imagesRes.result) {
    images.value = imagesRes.data
  } else {
    ElMessage.error(imagesRes.err)
  }
  loading.value = false
}

const deleteResource = async (resource: Resource) => {
  ElMessageBox.confirm(
      `确认删除「${resource.name}（${resource.path}）」吗？删除后将无法找回`,
      '删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
  ).then(async () => {
    const res = await store.resourceDelete(resource.path);
    if (res.result) {
      ElMessage.success("删除成功")
    } else {
      ElMessage.error(res.err)
    }
    await nextTick(async () => {
      await refreshResources()
    })
  })
}

const handleShow = async (resource: Resource) => {
  currentResource.value = resource
  drawer.value = true
}

const beforeUpload = async (file: any) => { // UploadRawFile
  if (file.type !== 'image/jpeg'
      && file.type !== 'image/png'
      && file.type !== 'image/gif') {
    ElMessage.error('上传的文件不是图片！')
    return false
  }
  let fd = new FormData()
  fd.append('files', file)
  try {
    const resp = await store.resourceUpload({form: fd});
    if (resp.result) {
      ElMessage.success('上传完成');
    } else {
      ElMessage.error(resp.err);
    }
  } catch (e: any) {
    ElMessage.error(e.toString());
  } finally {
    await nextTick(async () => {
      await refreshResources()
    })
  }
}

const copySealCode = async () => {
  ElMessage.success('复制海豹码成功！')
}

onBeforeMount(async () => {
  loading.value = true
  await refreshResources()
  new ClipboardJS('.resource-seal-code-copy-btn')
})

</script>

<template>
  <h2>资源管理</h2>
  <div class="tip">
    <el-collapse class="helptips">
      <el-collapse-item name="1">
        <template #title>
          <el-text tag="strong">查看帮助</el-text>
        </template>

        <el-text tag="p">
          <div>此处可以上传图片等资源，方便引用。</div>
        </el-text>
      </el-collapse-item>
    </el-collapse>
  </div>

  <main>
    <h3 class="flex items-center justify-between">
      <span>图片列表</span>
      <el-upload action="" multiple accept=".png, .jpg, jpeg, .gif"
                 :before-upload="beforeUpload" :file-list="fileList" :show-file-list="false">
        <el-button type="primary" :icon="Upload">上传图片</el-button>
      </el-upload>
    </h3>
    <el-table v-loading="loading" :data="images" table-layout="auto">
      <el-table-column align="center" min-width="64px">
        <template #default="scope">
          <resource-render class="min-w-10" :key="scope.row.path" :data="scope.row" mini/>
        </template>
      </el-table-column>
      <el-table-column prop="path" label="路径"/>
      <el-table-column align="center" prop="size" label="大小">
        <template #default="scope">
          {{ filesize(scope.row.size) }}
        </template>
      </el-table-column>
      <el-table-column fixed="right">
        <template #default="scope">
          <el-space size="small" direction="vertical">
            <el-button type="primary" link size="small" :icon="CopyDocument" plain
                       v-if="scope.row.type === 'image'"
                       class="resource-seal-code-copy-btn" :data-clipboard-text="`[图:${scope.row.path}]`"
                       @click="copySealCode()">
              复制海豹码
            </el-button>
            <el-button type="primary" link size="small" :icon="Search" plain
                       @click="handleShow(scope.row)">
              详情
            </el-button>
            <el-button type="success" link size="small" :icon="Download" plain tag="a" style="text-decoration: none;"
                       :href="`${urlBase}/sd-api/resource/download?path=${encodeURIComponent(scope.row.path)}&token=${encodeURIComponent(store.token)}`">
              下载
            </el-button>
            <el-button type="danger" link size="small" :icon="Delete" plain
                       @click="deleteResource(scope.row)">
              删除
            </el-button>
          </el-space>
        </template>
      </el-table-column>
    </el-table>
  </main>

  <el-drawer
      v-model="drawer"
      title="详情"
      class="resource-detail-drawer"
      direction="rtl">
    <el-space class="mx-auto" size="large" direction="vertical" alignment="center">
      <div class="max-w-xs">
        <resource-render :key="currentResource.path" :data="currentResource"/>
      </div>
      <el-descriptions title="" :column="1">
        <el-descriptions-item label="文件名">{{ currentResource.name }}</el-descriptions-item>
        <el-descriptions-item label="路径">{{ currentResource.path }}</el-descriptions-item>
        <el-descriptions-item label="大小">{{ filesize(currentResource.size) }}</el-descriptions-item>
      </el-descriptions>
    </el-space>
  </el-drawer>
</template>

<style scoped lang="scss">
.helptips {
  background-color: #f3f5f7;

  :deep(.el-collapse-item__header) {
    background-color: #f3f5f7;
  }

  :deep(.el-collapse-item__wrap) {
    background-color: #f3f5f7;
  }
}

.el-loading-mask {
  z-index: 9;
}

.el-drawer__body {
  display: flex;
  justify-content: center;
}

@media screen and (max-width: 700px) {
  .resource-detail-drawer {
    width: 50% !important;
  }
}

@media screen and (min-width: 700px) and (max-width: 1100px) {
  .resource-detail-drawer {
    width: 40% !important;
  }
}

@media screen and (min-width: 1100px) {
  .resource-detail-drawer {
    width: 30% !important;
  }
}
</style>