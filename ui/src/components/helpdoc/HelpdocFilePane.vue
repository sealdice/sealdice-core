<template>
  <ListActions class="helpdoc-action-block">
    <n-button type="primary" secondary @click="emit('openUpload')">
      <template #icon>
        <n-icon><i-tabler-upload /></n-icon>
      </template>
      上传
    </n-button>
    <template #end>
      <n-button secondary @click="emit('openConfig')">
        <template #icon>
          <n-icon><i-tabler-settings /></n-icon>
        </template>
        设置
      </n-button>
      <n-button
        v-show="checkedKeys.length > 0"
        type="error"
        secondary
        :loading="deleting"
        @click="emit('deleteFiles')"
      >
        <template #icon>
          <n-icon><i-tabler-trash /></n-icon>
        </template>
        删除所选
      </n-button>
    </template>
  </ListActions>

  <section v-if="activeUploadTasks.length" class="upload-panel">
    <div class="upload-panel-head">
      <h3>上传队列</h3>
      <n-tag size="small" :bordered="false"> {{ activeUploadTasks.length }} 项 </n-tag>
    </div>
    <article v-for="task in activeUploadTasks" :key="task.id" class="upload-item">
      <div class="upload-item-head">
        <div>
          <strong>{{ task.filename }}</strong>
          <span class="upload-meta">{{ Math.round(task.fileSize / 1024) }} KB</span>
        </div>
        <n-button
          v-if="task.status === 'error'"
          size="tiny"
          secondary
          @click="emit('retryTask', task)"
        >
          重试
        </n-button>
      </div>
      <n-progress type="line" :percentage="task.progress" :status="getTaskStatusType(task)" />
      <div class="upload-detail">
        <span>分块 {{ task.uploadedChunks.length }} / {{ task.expectedChunks || '-' }}</span>
        <span v-if="task.errorText" class="upload-error">{{ task.errorText }}</span>
      </div>
    </article>
  </section>

  <n-spin :show="loading">
    <main class="helpdoc-file-block">
      <header class="file-tree-title">
        <n-text strong>文件名</n-text>
        <n-text strong>分组</n-text>
      </header>
      <ListEmptyState v-if="!loading && docTree.length === 0" description="暂无帮助文档文件" />
      <n-tree
        v-else
        v-model:checked-keys="checkedKeys"
        :data="docTree"
        label-field="label"
        block-line
        default-expand-all
        show-line
        cascade
        checkable
        :render-prefix="renderPrefix"
        :render-label="renderLabel"
        :render-suffix="renderSuffix"
      />
    </main>
  </n-spin>
</template>

<script setup lang="tsx">
import { h } from 'vue';
import { NButton, NProgress, NTag, NText } from 'naive-ui';
import ListActions from '@/components/shared/ListActions.vue';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
import type { HelpDocTreeOption } from '@/features/helpdoc/viewModel';
import type { ResumableUploadTask } from '@/features/upload/resumableUpload';

type TreeRenderContext = {
  option: Record<string, unknown>;
  checked: boolean;
  selected: boolean;
};

const checkedKeys = defineModel<Array<string | number>>('checkedKeys', { required: true });

defineProps<{
  docTree: HelpDocTreeOption[];
  loading: boolean;
  activeUploadTasks: ResumableUploadTask[];
  deleting: boolean;
}>();

const emit = defineEmits<{
  openUpload: [];
  openConfig: [];
  deleteFiles: [];
  retryTask: [task: ResumableUploadTask];
}>();

function renderPrefix({ option }: TreeRenderContext) {
  const raw = option as HelpDocTreeOption | undefined;
  const icon = raw?.icon;
  if (icon === 'folder') return <i-tabler-folder color="var(--sd-text-muted)" />;
  if (icon === 'json' || icon === 'xlsx') {
    return <i-tabler-file-text color="var(--sd-text-secondary)" />;
  }
  return <i-tabler-file-text />;
}

function renderLabel({ option }: TreeRenderContext) {
  const raw = option as HelpDocTreeOption | undefined;
  return h(
    NText,
    { class: raw?.raw.deleted ? 'del-line file-info' : 'file-info' },
    { default: () => option.label as string }
  );
}

function renderSuffix({ option }: TreeRenderContext) {
  const raw = option as HelpDocTreeOption | undefined;
  if (!raw?.tag) return null;
  return h(
    NTag,
    { size: 'small', type: raw.tag.type, bordered: false },
    { default: () => raw.tag?.label ?? '' }
  );
}

function getTaskStatusType(task: ResumableUploadTask) {
  if (task.status === 'error') return 'error';
  if (task.status === 'success') return 'success';
  return 'default';
}
</script>

<style scoped>
.helpdoc-action-block {
  margin-bottom: 0.75rem;
}

.helpdoc-file-block {
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-sm);
  background: var(--sd-bg-elevated);
  color: var(--sd-text-primary);
  padding: 0.75rem;
}

.file-tree-title {
  display: flex;
  justify-content: space-between;
  padding: 0 23px 6px 50px;
  border-bottom: 1px solid var(--sd-border);
  margin-bottom: 0.5rem;
}

.file-info {
  flex: auto;
  width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.del-line {
  text-decoration: line-through;
}

.upload-panel {
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-sm);
  padding: 0.75rem;
  margin-bottom: 0.75rem;
  background: var(--sd-bg-elevated-soft);
}

.upload-panel-head,
.upload-item-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.upload-panel-head h3 {
  margin: 0 0 0.5rem;
  font-size: 0.95rem;
}

.upload-item {
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-sm);
  padding: 0.625rem;
  background: var(--sd-bg-elevated);
}

.upload-item + .upload-item {
  margin-top: 0.5rem;
}

.upload-meta,
.upload-detail {
  color: var(--n-text-color-disabled);
  font-size: 0.78rem;
}

.upload-meta {
  margin-left: 0.5rem;
}

.upload-detail {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.25rem;
}

.upload-error {
  color: var(--sd-error);
}

@media screen and (max-width: 639.9px) {
  .upload-panel-head,
  .upload-item-head,
  .upload-detail {
    align-items: flex-start;
    flex-direction: column;
  }

  .file-tree-title {
    gap: 0.75rem;
    padding-left: 0;
  }

  .upload-meta {
    display: block;
    margin-left: 0;
  }
}
</style>
