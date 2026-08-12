<template>
  <h4>词库列表</h4>
  <header class="censor-files-header">
    <n-upload
      action=""
      multiple
      accept="application/text,.txt,application/toml,.toml"
      :show-file-list="false"
      :custom-request="handleUpload"
    >
      <n-button type="primary" secondary>
        <template #icon>
          <n-icon><i-tabler-upload /></n-icon>
        </template>
        导入
      </n-button>
    </n-upload>
    <n-flex class="censor-files-template-actions" wrap>
      <n-button type="primary" size="tiny" text @click="downloadTomlTemplate">
        <template #icon>
          <n-icon><i-tabler-download /></n-icon>
        </template>
        下载 toml 词库模板
      </n-button>
      <n-button type="primary" size="tiny" text @click="downloadTxtTemplate">
        <template #icon>
          <n-icon><i-tabler-file-download /></n-icon>
        </template>
        下载 txt 词库模板
      </n-button>
    </n-flex>
  </header>
  <main class="mt-4">
    <ResponsiveDataView :compact-at="560" aria-label="审查词库文件">
      <template #table>
        <n-data-table :columns="columns" :data="files" :scroll-x="520" />
      </template>
      <template #compact>
        <ul class="censor-files-list">
          <li v-for="file in files" :key="file.key" class="censor-files-list__item">
            <strong>{{ file.name }}</strong>
            <div class="censor-files-list__counts">
              <span v-for="level in sensitiveLevels" :key="level">
                <CensorSensitiveTag :level="level" />
                <b>{{ file.count?.[level] ?? 0 }}</b>
              </span>
            </div>
          </li>
        </ul>
      </template>
    </ResponsiveDataView>
  </main>
</template>

<script setup lang="tsx">
import type { DataTableColumns, UploadCustomRequestOptions } from 'naive-ui';
import type { CensorFileInfo } from '@/api';
import ResponsiveDataView from '@/components/shared/ResponsiveDataView.vue';
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
    minWidth: 180,
    ellipsis: { tooltip: true },
  },
  {
    title: () => <CensorSensitiveTag level={1} />,
    key: 'count[1]',
    minWidth: 82,
    render: row => row.count?.[1] ?? 0,
  },
  {
    title: () => <CensorSensitiveTag level={2} />,
    key: 'count[2]',
    minWidth: 82,
    render: row => row.count?.[2] ?? 0,
  },
  {
    title: () => <CensorSensitiveTag level={3} />,
    key: 'count[3]',
    minWidth: 82,
    render: row => row.count?.[3] ?? 0,
  },
  {
    title: () => <CensorSensitiveTag level={4} />,
    key: 'count[4]',
    minWidth: 82,
    render: row => row.count?.[4] ?? 0,
  },
];
const sensitiveLevels = [1, 2, 3, 4] as const;

async function handleUpload(options: UploadCustomRequestOptions) {
  try {
    await props.uploadFile(options.file.file as File);
    options.onFinish();
  } catch {
    options.onError();
  }
}
</script>

<style scoped>
.censor-files-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.censor-files-header :deep(.n-upload) {
  width: auto;
}

.censor-files-list {
  display: grid;
  margin: 0;
  padding: 0;
  gap: 0.625rem;
  list-style: none;
}

.censor-files-list__item {
  display: grid;
  min-width: 0;
  border-bottom: 1px solid var(--sd-border-soft);
  gap: 0.65rem;
  padding: 0.75rem 0;
}

.censor-files-list__item strong {
  overflow-wrap: anywhere;
}

.censor-files-list__counts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem 0.75rem;
}

.censor-files-list__counts span {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

@media screen and (max-width: 639.9px) {
  .censor-files-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
