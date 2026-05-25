<template>
  <n-checkbox-group v-model:value="value" :disabled="disabled">
    <div class="backup-selection-group">
      <n-checkbox
        v-for="option in BACKUP_SELECTION_OPTIONS"
        :key="option.key"
        :value="option.key"
        :disabled="disabled || option.disabled"
      >
        {{ option.label }}
      </n-checkbox>
    </div>
  </n-checkbox-group>
</template>

<script setup lang="ts">
import { BACKUP_SELECTION_OPTIONS, type BackupSelectionKey } from '@/features/backup/viewModel';

const value = defineModel<BackupSelectionKey[]>('value', { required: true });

defineProps<{
  disabled?: boolean;
}>();
</script>

<style scoped>
.backup-selection-group {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 16px;
}

@media (max-width: 720px) {
  .backup-selection-group {
    grid-template-columns: 1fr;
  }
}
</style>
