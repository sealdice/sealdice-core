<template>
  <div class="setting-string-list">
    <div v-for="(item, index) in rows" :key="index" class="setting-string-list-row">
      <n-input
        class="setting-string-list-input"
        :value="item"
        placeholder="条目内容，如 . 或 !"
        @update:value="updateRow(index, $event)"
      />
      <n-tooltip>
        <template #trigger>
          <n-button quaternary circle type="error" aria-label="删除该项" @click="removeRow(index)">
            <template #icon><i-tabler-trash /></template>
          </n-button>
        </template>
        删除该项
      </n-tooltip>
    </div>

    <div class="setting-string-list-add">
      <n-input
        v-model:value="pendingItem"
        class="setting-string-list-input"
        placeholder="输入新条目后回车或点击添加"
        clearable
        @keydown.enter.prevent="addItem"
      />
      <n-button dashed :disabled="!pendingItem.trim()" @click="addItem">
        <template #icon><i-tabler-plus /></template>
        添加
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { normalizeStringListValues } from '@/features/baseSetting/viewModel';

const message = useMessage();
const model = defineModel<string[]>({ required: true });

// 本地行副本：允许正在编辑的空行存在，提交到 model 时才做清理。
const rows = ref<string[]>([...model.value]);
const pendingItem = ref('');

// 外部（如远端数据刷新、放弃更改）更新 model 时同步回本地行。
watch(
  () => model.value,
  value => {
    const committed = normalizeStringListValues(value);
    if (JSON.stringify(committed) === JSON.stringify(normalizeStringListValues(rows.value))) return;
    rows.value = [...value];
  },
  { deep: true }
);

function commit(items: string[]) {
  rows.value = [...items];
  model.value = normalizeStringListValues(items);
}

function updateRow(index: number, value: string) {
  const normalized = value.trim();
  if (
    normalized &&
    rows.value.some((row, rowIndex) => rowIndex !== index && row.trim() === normalized)
  ) {
    message.warning('该项已存在');
    return;
  }
  commit(rows.value.map((row, rowIndex) => (rowIndex === index ? value : row)));
}

function removeRow(index: number) {
  commit(rows.value.filter((_, rowIndex) => rowIndex !== index));
}

function addItem() {
  const value = pendingItem.value.trim();
  if (!value) return;
  if (rows.value.some(row => row.trim() === value)) {
    message.warning('该项已存在');
    pendingItem.value = '';
    return;
  }
  commit([...rows.value, value]);
  pendingItem.value = '';
}
</script>

<style scoped>
.setting-string-list {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 0.625rem;
}

.setting-string-list-row,
.setting-string-list-add {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.setting-string-list-input {
  flex: 1 1 auto;
  min-width: 0;
  max-width: 24rem;
}
</style>
