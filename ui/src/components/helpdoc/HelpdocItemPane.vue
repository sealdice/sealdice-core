<script setup lang="tsx">
import { NFlex, NText, type DataTableColumns } from 'naive-ui';
import type { HelpTextVo } from '@/api';
import type { HelpdocItemQueryModel } from '@/features/helpdoc/queries';

const query = defineModel<HelpdocItemQueryModel>('query', { required: true });

defineProps<{
  loading: boolean;
  items: HelpTextVo[];
  total: number;
  groupOptions: { label: string; value: string }[];
  columns: DataTableColumns<HelpTextVo>;
}>();

const emit = defineEmits<{
  search: [];
  reset: [];
}>();
</script>

<template>
  <n-spin :show="loading">
    <main class="item-list-container">
      <header>
        <n-form
          :model="query"
          size="small"
          class="flex flex-wrap"
          label-width="auto"
          label-placement="left"
          inline
        >
          <n-form-item label="序号">
            <n-input-number v-model:value="query.id" placeholder="" clearable />
          </n-form-item>
          <n-form-item label="分组">
            <n-select
              v-model:value="query.group"
              placeholder="选择分组"
              filterable
              clearable
              :options="groupOptions"
            />
          </n-form-item>
          <n-form-item label="来源文件">
            <n-input v-model:value="query.from" placeholder="" clearable />
          </n-form-item>
          <n-form-item label="词条名">
            <n-input v-model:value="query.title" placeholder="" clearable />
          </n-form-item>
          <n-form-item>
            <n-flex size="small">
              <n-button type="info" secondary @click="emit('search')">查询</n-button>
              <n-button secondary @click="emit('reset')">重置</n-button>
            </n-flex>
          </n-form-item>
        </n-form>
      </header>

      <n-data-table class="item-list" :columns="columns" :data="items" size="small" :bordered="false" remote :scroll-x="980" />

      <footer>
        <n-flex class="item-list-pagination" align="center" justify="end" wrap>
          <n-text depth="3">共 {{ total }} 条</n-text>
          <n-pagination
            v-model:page="query.pageNum"
            v-model:page-size="query.pageSize"
            show-size-picker
            show-quick-jumper
            :page-sizes="[10, 20, 30, 50]"
            :page-slot="5"
            :item-count="total"
          />
        </n-flex>
      </footer>
    </main>
  </n-spin>
</template>

<style scoped>
.item-list-container {
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.item-list {
  width: 100%;
}

.item-list-pagination {
  margin-top: 10px;
}
</style>
