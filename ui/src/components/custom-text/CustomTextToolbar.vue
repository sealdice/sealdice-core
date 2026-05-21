<script setup lang="ts">
const keyword = defineModel<string>('keyword', { required: true });

defineProps<{
  previewLoading: boolean;
}>();

const emit = defineEmits<{
  refreshPreview: [];
  openImport: [];
}>();
</script>

<template>
  <div class="custom-text-toolbar">
    <n-flex align="center" size="small" class="custom-text-search">
      <n-text>搜索：</n-text>
      <span class="custom-text-search-input">
        <n-input v-model:value="keyword" size="small" clearable>
          <template #prefix>
            <n-icon><i-carbon-search /></n-icon>
          </template>
        </n-input>
      </span>
    </n-flex>
    <n-flex align="center" size="small" class="custom-text-actions">
      <n-button
        type="info"
        secondary
        :loading="previewLoading"
        @click="emit('refreshPreview')"
      >
        刷新预览
      </n-button>
      <n-button type="info" secondary @click="emit('openImport')">导入/导出</n-button>
    </n-flex>
  </div>
</template>

<style scoped>
.custom-text-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-top: 1rem;
}

@media screen and (max-width: 767.9px) {
  .custom-text-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .custom-text-search,
  .custom-text-actions,
  .custom-text-search-input {
    width: 100%;
  }

  .custom-text-actions {
    justify-content: flex-start;
  }

  .custom-text-search-input :deep(.n-input) {
    width: 100%;
  }
}
</style>
