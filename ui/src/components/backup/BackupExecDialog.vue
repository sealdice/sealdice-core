<template>
  <n-modal v-model:show="show" preset="card" title="立即备份" class="backup-dialog" :mask-closable="false">
    <n-space vertical size="large">
      <div>
        <div class="backup-dialog__label">备份范围</div>
        <BackupSelectionGroup v-model:value="selections" :disabled="pending" />
      </div>

      <div>
        <div class="backup-dialog__label">备份文件名预览</div>
        <n-text type="info" class="backup-dialog__preview">
          {{ preview }}
        </n-text>
      </div>
    </n-space>

    <template #footer>
      <n-space justify="end">
        <n-button :disabled="pending" @click="show = false">
          取消
        </n-button>
        <n-button type="primary" :loading="pending" @click="emit('submit')">
          立即备份
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  buildBackupFilenamePreview,
  formatBackupSelection,
  type BackupSelectionKey,
} from '@/features/backup/viewModel';
import BackupSelectionGroup from './BackupSelectionGroup.vue';

const show = defineModel<boolean>('show', { required: true });
const selections = defineModel<BackupSelectionKey[]>('selections', { required: true });

const props = defineProps<{
  timestamp: string;
  pending: boolean;
}>();

const emit = defineEmits<{
  submit: [];
}>();

const preview = computed(() =>
  buildBackupFilenamePreview(props.timestamp, formatBackupSelection(selections.value), false),
);
</script>

<style scoped>
.backup-dialog {
  width: min(640px, calc(100vw - 32px));
}

.backup-dialog__label {
  margin-bottom: 8px;
  font-weight: 650;
}

.backup-dialog__preview {
  overflow-wrap: anywhere;
}
</style>
