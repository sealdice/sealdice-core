<template>
  <section class="sd-query-toolbar">
    <ProSearchForm
      :form="form"
      :columns="columns"
      size="small"
      label-placement="left"
      :label-width="labelWidth"
      :cols="cols"
      :loading="loading"
      :search-button-props="{ type: 'primary', size: 'small' }"
      :reset-button-props="{ size: 'small' }"
      :collapse-button-props="false"
    />
  </section>
</template>

<script setup lang="ts" generic="T extends Record<string, any> = Record<string, unknown>">
import { createProSearchForm, ProSearchForm, type ProSearchFormColumns } from 'pro-naive-ui';

withDefaults(
  defineProps<{
    form: ReturnType<typeof createProSearchForm<T>>;
    columns: ProSearchFormColumns;
    loading?: boolean;
    labelWidth?: string | number;
    cols?: string | number;
  }>(),
  {
    loading: false,
    labelWidth: 84,
    cols: '1 s:2 l:3',
  }
);
</script>

<style scoped>
.sd-query-toolbar {
  padding: 0.875rem 1rem;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-xs);
  background: var(--sd-bg-elevated-soft);
}
</style>
