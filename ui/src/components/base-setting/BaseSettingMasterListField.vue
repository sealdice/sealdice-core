<template>
  <div class="master-list">
    <div v-for="(master, index) in rows" :key="index" class="master-list-row">
      <n-input
        class="master-list-input"
        :value="master"
        placeholder="平台:账号，如 QQ:12345"
        @update:value="updateRow(index, $event)"
      />
      <n-tooltip>
        <template #trigger>
          <n-button
            quaternary
            circle
            type="error"
            aria-label="删除该 Master"
            @click="removeRow(index)"
          >
            <template #icon><i-tabler-trash /></template>
          </n-button>
        </template>
        删除该 Master
      </n-tooltip>
    </div>

    <div class="master-list-add">
      <n-input
        v-model:value="pendingMaster"
        class="master-list-input"
        placeholder="输入新 Master 后回车或点击添加，如 QQ:12345"
        clearable
        @keydown.enter.prevent="addMaster"
      />
      <n-button dashed :disabled="!pendingMaster.trim()" @click="addMaster">
        <template #icon><i-tabler-plus /></template>
        添加 Master
      </n-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { normalizeMasterListValues } from '@/features/baseSetting/viewModel';

const message = useMessage();
const model = defineModel<string[]>({ required: true });

// 本地行副本：允许正在编辑的空行存在，提交到 model 时才做清理。
const rows = ref<string[]>([...model.value]);
const pendingMaster = ref('');

// 外部（如远端数据刷新、放弃更改）更新 model 时同步回本地行。
watch(
  () => model.value,
  value => {
    const committed = normalizeMasterListValues(value);
    if (JSON.stringify(committed) === JSON.stringify(normalizeMasterListValues(rows.value))) return;
    rows.value = [...value];
  },
  { deep: true }
);

function commit(items: string[]) {
  rows.value = [...items];
  model.value = normalizeMasterListValues(items);
}

function updateRow(index: number, value: string) {
  commit(rows.value.map((row, rowIndex) => (rowIndex === index ? value : row)));
}

function removeRow(index: number) {
  commit(rows.value.filter((_, rowIndex) => rowIndex !== index));
}

function addMaster() {
  const value = pendingMaster.value.trim();
  if (!value) return;
  if (rows.value.some(row => row.trim() === value)) {
    message.warning('该 Master 已存在');
    pendingMaster.value = '';
    return;
  }
  commit([...rows.value, value]);
  pendingMaster.value = '';
}
</script>

<style scoped>
.master-list {
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 0.625rem;
}

.master-list-row,
.master-list-add {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.master-list-input {
  flex: 1 1 auto;
  min-width: 0;
  max-width: 24rem;
}
</style>
