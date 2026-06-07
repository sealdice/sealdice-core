<template>
  <n-flex class="custom-text-filter-row mb-8 mt-4" align="center" wrap>
    <n-radio-group v-model:value="mode" @update:value="emit('modeChange', $event as CustomTextFilterMode)">
      <n-radio
        v-for="item of filterModes"
        :key="item.value"
        :value="item.value"
        :label="item.desc"
      />
    </n-radio-group>
    <n-flex v-if="mode === 'group'" align="center" class="custom-text-group-filter">
      <n-text>分组：</n-text>
      <span class="custom-text-group-select">
        <n-select
          v-model:value="group"
          filterable
          tag
          :options="groups.map(item => ({ label: item, value: item }))"
        />
      </span>
    </n-flex>
  </n-flex>
</template>

<script setup lang="ts">
import type { CustomTextFilterMode } from '@/features/customText/viewModel';

const mode = defineModel<CustomTextFilterMode>('mode', { required: true });
const group = defineModel<string>('group', { required: true });

defineProps<{
  groups: string[];
}>();

const emit = defineEmits<{
  modeChange: [mode: CustomTextFilterMode];
}>();

const filterModes = [
  { value: 'all', desc: '全部' },
  { value: 'unmodified', desc: '默认文案' },
  { value: 'modified', desc: '修改过' },
  { value: 'group', desc: '指定分组' },
  { value: 'deprecated', desc: '旧版文本' },
] satisfies Array<{ value: CustomTextFilterMode; desc: string }>;
</script>

<style scoped>
@media screen and (max-width: 767.9px) {
  .custom-text-filter-row,
  .custom-text-group-filter,
  .custom-text-group-select {
    width: 100%;
  }

  .custom-text-group-select :deep(.n-select) {
    width: 100%;
  }
}
</style>
