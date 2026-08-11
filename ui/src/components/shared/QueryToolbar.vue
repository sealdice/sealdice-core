<template>
  <section class="sd-query-toolbar">
    <div class="sd-query-toolbar__form">
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
    </div>
    <div v-if="$slots.actions" class="sd-query-toolbar__actions">
      <slot name="actions" />
    </div>
    <div v-if="$slots.meta" class="sd-query-toolbar__meta">
      <slot name="meta" />
    </div>
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
  display: grid;
  min-width: 0;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: start;
  gap: 1rem;
}

.sd-query-toolbar__form {
  min-width: 0;
  padding: 0.875rem 1rem;
  border: 1px solid var(--sd-border-soft);
  border-radius: 6px;
  background: var(--sd-bg-elevated-soft);
}

.sd-query-toolbar__actions {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 0.5rem;
  padding-top: 0.15rem;
}

.sd-query-toolbar__meta {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 0.6rem;
  min-width: 0;
  color: var(--sd-text-secondary);
  font-size: 0.85rem;
}

@media (max-width: 860px) {
  .sd-query-toolbar {
    grid-template-columns: minmax(0, 1fr);
  }

  .sd-query-toolbar__actions {
    justify-content: flex-start;
    padding-top: 0;
  }
}
</style>
