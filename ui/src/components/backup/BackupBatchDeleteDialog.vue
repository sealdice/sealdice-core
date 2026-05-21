<script setup lang="ts">
import { computed } from 'vue';
import { filesize } from 'filesize';
import type { FileItem } from '@/api';

type CheckboxOption = {
  label: string;
  value: string;
};

const show = defineModel<boolean>('show', { required: true });
const selectedNames = defineModel<string[]>('selectedNames', { required: true });

const props = defineProps<{
  items: FileItem[];
  pending: boolean;
}>();

const emit = defineEmits<{
  submit: [];
}>();

const options = computed<CheckboxOption[]>(() =>
  props.items.map(item => ({
    label: item.name,
    value: item.name,
  })),
);

const selectedItems = computed(() =>
  props.items.filter(item => selectedNames.value.includes(item.name)),
);

const selectedSize = computed(() =>
  selectedItems.value.reduce((sum, item) => sum + item.fileSize, 0),
);

const checkedAll = computed({
  get: () => props.items.length > 0 && selectedNames.value.length === props.items.length,
  set: value => {
    selectedNames.value = value ? props.items.map(item => item.name) : [];
  },
});

const indeterminate = computed(() =>
  selectedNames.value.length > 0 && selectedNames.value.length < props.items.length,
);
</script>

<template>
  <n-modal v-model:show="show" preset="card" title="批量删除备份" class="backup-dialog" :mask-closable="false">
    <n-alert type="warning" :bordered="false" class="backup-dialog__alert">
      默认勾选最近的 5 个备份之前的历史备份，可自行调整。删除后无法找回。
    </n-alert>

    <n-space vertical>
      <div class="backup-dialog__toolbar">
        <n-checkbox v-model:checked="checkedAll" :indeterminate="indeterminate" :disabled="pending">
          {{ checkedAll ? '取消全选' : '全选' }}
        </n-checkbox>
        <n-text type="info">
          已勾选 {{ selectedNames.length }} 个，共 {{ filesize(selectedSize) }}
        </n-text>
      </div>

      <n-checkbox-group v-model:value="selectedNames" :disabled="pending">
        <n-scrollbar class="backup-dialog__scroll">
          <n-space vertical size="small">
            <n-checkbox
              v-for="option in options"
              :key="String(option.value)"
              :value="option.value"
            >
              {{ option.label }}
            </n-checkbox>
          </n-space>
        </n-scrollbar>
      </n-checkbox-group>
    </n-space>

    <template #footer>
      <n-space justify="end">
        <n-button :disabled="pending" @click="show = false">
          取消
        </n-button>
        <n-button
          type="error"
          :disabled="selectedNames.length === 0"
          :loading="pending"
          @click="emit('submit')"
        >
          删除所选
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style scoped>
.backup-dialog {
  width: min(680px, calc(100vw - 32px));
}

.backup-dialog__alert {
  margin-bottom: 16px;
}

.backup-dialog__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.backup-dialog__scroll {
  max-height: min(46vh, 420px);
}

@media (max-width: 560px) {
  .backup-dialog__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
