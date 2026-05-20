<script setup lang="tsx">
import type { DataTableColumns, UploadCustomRequestOptions } from 'naive-ui';
import type { CensorFileInfo } from '@/api';
import CensorSensitiveTag from './CensorSensitiveTag.vue';

const props = defineProps<{
  files: CensorFileInfo[];
  uploadFile: (file: File) => Promise<void>;
  downloadTomlTemplate: () => Promise<void>;
  downloadTxtTemplate: () => Promise<void>;
}>();

const columns: DataTableColumns<CensorFileInfo> = [
  {
    title: '文件名',
    key: 'name',
  },
  {
    title: () => <CensorSensitiveTag level={1} />,
    key: 'count[1]',
    render: row => row.count?.[1] ?? 0,
  },
  {
    title: () => <CensorSensitiveTag level={2} />,
    key: 'count[2]',
    render: row => row.count?.[2] ?? 0,
  },
  {
    title: () => <CensorSensitiveTag level={3} />,
    key: 'count[3]',
    render: row => row.count?.[3] ?? 0,
  },
  {
    title: () => <CensorSensitiveTag level={4} />,
    key: 'count[4]',
    render: row => row.count?.[4] ?? 0,
  },
];

async function handleUpload(options: UploadCustomRequestOptions) {
  try {
    await props.uploadFile(options.file.file as File);
    options.onFinish();
  } catch {
    options.onError();
  }
}
</script>

<template>
  <h4>词库列表</h4>
  <header class="page-header">
    <n-upload
      action=""
      multiple
      accept="application/text,.txt,application/toml,.toml"
      :show-file-list="false"
      :custom-request="handleUpload"
    >
      <n-button type="info" secondary>
        <template #icon>
          <n-icon><i-carbon-upload /></n-icon>
        </template>
        导入
      </n-button>
    </n-upload>
    <n-flex>
      <n-button type="primary" size="tiny" text @click="downloadTomlTemplate">
        <template #icon>
          <n-icon><i-carbon-download /></n-icon>
        </template>
        下载 toml 词库模板
      </n-button>
      <n-button type="primary" size="tiny" text @click="downloadTxtTemplate">
        <template #icon>
          <n-icon><i-carbon-save /></n-icon>
        </template>
        下载 txt 词库模板
      </n-button>
    </n-flex>
  </header>
  <main class="mt-4">
    <n-data-table :columns="columns" :data="files" />
  </main>
</template>
