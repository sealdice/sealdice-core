<template>
  <n-popover v-if="uploadTasks.length > 0" trigger="click" placement="bottom-end">
    <template #trigger>
      <n-badge :value="activeCount" :show="activeCount > 0" :max="99">
        <n-button quaternary circle size="small" :aria-label="`上传队列：${activeCount} 个进行中`">
          <template #icon>
            <n-icon
              class="sd-upload-indicator__icon"
              :class="{ 'sd-upload-indicator__icon--active': activeCount > 0 }"
              size="1.15rem"
            >
              <i-tabler-cloud-upload />
            </n-icon>
          </template>
        </n-button>
      </n-badge>
    </template>

    <div class="sd-upload-panel">
      <header class="sd-upload-panel__head">
        <strong>上传队列</strong>
        <span class="sd-upload-panel__summary">{{ summaryText }}</span>
        <n-button v-if="activeCount > 0" text size="tiny" @click="cancelAllUploads"
          >全部取消</n-button
        >
      </header>

      <div v-for="task in uploadTasks" :key="task.id" class="sd-upload-panel__item">
        <div class="sd-upload-panel__row">
          <strong class="sd-upload-panel__name">{{ task.filename }}</strong>
          <n-tag size="small" :bordered="false" :type="statusTagType(task.status)">
            {{ statusLabel(task.status) }}
          </n-tag>
        </div>
        <n-progress
          v-if="isActiveStatus(task.status)"
          type="line"
          :percentage="task.progress"
          :height="6"
        />
        <div class="sd-upload-panel__row">
          <span v-if="task.errorText" class="sd-upload-panel__error">{{ task.errorText }}</span>
          <n-button
            v-if="task.status === 'error' && task.retryable"
            size="tiny"
            secondary
            @click="retryUpload(task.id)"
          >
            重试
          </n-button>
          <n-button
            v-if="task.status !== 'success'"
            size="tiny"
            secondary
            @click="cancelUpload(task.id)"
          >
            取消
          </n-button>
        </div>
      </div>
    </div>
  </n-popover>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useUploadStore } from '@/features/upload/store';
import { isActiveUpload as isActiveStatus, type UploadTaskStatus } from '@/features/upload/queue';

const uploadStore = useUploadStore();
const { tasks: uploadTasks } = storeToRefs(uploadStore);
const { cancelAll: cancelAllUploads, cancel: cancelUpload, retry: retryUpload } = uploadStore;

const activeCount = computed(
  () => uploadTasks.value.filter(task => isActiveStatus(task.status)).length
);
const failedCount = computed(
  () => uploadTasks.value.filter(task => task.status === 'error').length
);
const summaryText = computed(
  () => `${activeCount.value} 个上传中${failedCount.value ? `，${failedCount.value} 个失败` : ''}`
);

function statusLabel(status: UploadTaskStatus): string {
  if (status === 'error') return '失败';
  if (status === 'success') return '已完成';
  if (status === 'canceled') return '已取消';
  return '上传中';
}

function statusTagType(status: UploadTaskStatus): 'error' | 'success' | 'info' {
  if (status === 'error') return 'error';
  if (status === 'success') return 'success';
  return 'info';
}
</script>

<style scoped>
.sd-upload-indicator__icon--active {
  animation: sd-upload-spin 1.1s linear infinite;
}

@keyframes sd-upload-spin {
  from {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-3px);
  }
  to {
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .sd-upload-indicator__icon--active {
    animation: none;
  }
}

.sd-upload-panel {
  display: grid;
  width: min(78vw, 320px);
  gap: 0.5rem;
}

.sd-upload-panel__head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.sd-upload-panel__summary {
  flex: 1 1 auto;
  color: var(--sd-text-muted);
  font-size: 0.78rem;
}

.sd-upload-panel__item {
  display: grid;
  gap: 0.35rem;
  border-top: 1px solid var(--sd-border-soft);
  padding-top: 0.5rem;
}

.sd-upload-panel__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.sd-upload-panel__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sd-upload-panel__error {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  color: var(--sd-error, #d03050);
  font-size: 0.8rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
