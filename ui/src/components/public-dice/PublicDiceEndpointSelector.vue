<template>
  <section class="public-dice-endpoints">
    <div class="public-dice-endpoints__head">
      <h2>选择要上报的终端</h2>
      <span>{{ checkedRowKeys.length }} / {{ rows.length }}</span>
    </div>

    <n-empty v-if="!loading && rows.length === 0" description="暂无可上报终端" />

    <n-data-table
      v-else
      v-model:checked-row-keys="checkedRowKeys"
      class="public-dice-endpoints__table"
      :columns="columns"
      :data="rows"
      :loading="loading"
      :row-key="rowKey"
      :bordered="false"
      :scroll-x="640"
      size="small"
    />

    <div class="public-dice-endpoints__mobile-list">
      <label
        v-for="row in rows"
        :key="row.id"
        :class="['public-dice-endpoint-card', { 'public-dice-endpoint-card--disabled': disabled }]"
      >
        <n-checkbox
          :checked="checkedSet.has(row.id)"
          :disabled="disabled"
          @update:checked="setRowChecked(row.id, $event === true)"
        />
        <span class="public-dice-endpoint-card__main">
          <strong>{{ row.userId }}</strong>
          <span>{{ row.protocol }}</span>
        </span>
        <n-tag size="small" :type="row.stateTagType" :bordered="false">
          {{ row.stateText }}
        </n-tag>
      </label>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, h } from 'vue';
import { NTag, type DataTableColumns, type DataTableRowKey } from 'naive-ui';
import type { PublicDiceEndpointRow } from '@/features/publicDice/viewModel';

const props = defineProps<{
  rows: PublicDiceEndpointRow[];
  disabled: boolean;
  loading?: boolean;
}>();

const checkedRowKeys = defineModel<DataTableRowKey[]>('checkedRowKeys', { required: true });

const checkedSet = computed(() => new Set(checkedRowKeys.value.map(String)));

const rowKey = (row: PublicDiceEndpointRow) => row.id;

const columns = computed<DataTableColumns<PublicDiceEndpointRow>>(() => [
  {
    type: 'selection',
    disabled: () => props.disabled,
  },
  {
    title: '账号',
    key: 'userId',
    sorter: 'default',
    minWidth: 160,
  },
  {
    title: '平台',
    key: 'platform',
    sorter: 'default',
    width: 120,
  },
  {
    title: '协议',
    key: 'protocol',
    sorter: 'default',
    minWidth: 160,
  },
  {
    title: '状态',
    key: 'stateText',
    sorter: 'default',
    width: 110,
    render: row => h(NTag, { size: 'small', type: row.stateTagType, bordered: false }, { default: () => row.stateText }),
  },
]);

function setRowChecked(id: string, checked: boolean) {
  if (props.disabled) return;
  const next = new Set(checkedRowKeys.value.map(String));
  if (checked) {
    next.add(id);
  } else {
    next.delete(id);
  }
  checkedRowKeys.value = Array.from(next);
}
</script>

<style scoped>
.public-dice-endpoints {
  min-width: 0;
}

.public-dice-endpoints__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.public-dice-endpoints__head h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 650;
}

.public-dice-endpoints__head span {
  color: var(--sd-text-muted);
  font-size: 13px;
}

.public-dice-endpoints__mobile-list {
  display: none;
}

.public-dice-endpoint-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--sd-border);
  border-radius: 8px;
  background: var(--sd-bg-elevated);
}

.public-dice-endpoint-card--disabled {
  cursor: not-allowed;
}

.public-dice-endpoint-card__main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.public-dice-endpoint-card__main strong,
.public-dice-endpoint-card__main span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.public-dice-endpoint-card__main span {
  color: var(--sd-text-muted);
  font-size: 12px;
}

@media (max-width: 760px) {
  .public-dice-endpoints__table {
    display: none;
  }

  .public-dice-endpoints__mobile-list {
    display: grid;
    gap: 10px;
  }
}
</style>
