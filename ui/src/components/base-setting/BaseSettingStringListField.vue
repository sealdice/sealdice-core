<template>
  <n-select
    class="setting-string-list"
    :value="model"
    :options="options"
    multiple
    filterable
    tag
    clearable
    max-tag-count="responsive"
    placeholder="输入后回车添加"
    @update:value="updateValue"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { buildBaseSettingStringListOptions } from '@/features/baseSetting/viewModel';

const model = defineModel<string[]>({ required: true });

const options = computed(() => buildBaseSettingStringListOptions(model.value));

function updateValue(value: string[] | null) {
  model.value = buildBaseSettingStringListOptions(value ?? []).map(option => option.value);
}
</script>

<style scoped>
.setting-string-list {
  width: 100%;
}
</style>
