<template>
  <RepeatableList add-label="添加 Master">
    <RepeatableItem
      v-for="(master, index) in rows"
      :key="index"
      :title="`Master ${index + 1}`"
      remove-label="删除该 Master"
      @remove="removeRow(index)"
    >
      <n-input
        class="master-list-input"
        :value="master"
        placeholder="平台:账号，如 QQ:12345"
        @update:value="updateRow(index, $event)"
      />
    </RepeatableItem>

    <template #footer>
      <div class="master-list-add">
        <n-input
          v-model:value="pendingMaster"
          class="master-list-input"
          placeholder="输入新 Master，如 QQ:12345"
          clearable
          @keydown.enter.prevent="addMaster"
        />
        <n-button
          type="primary"
          secondary
          size="small"
          :disabled="!pendingMaster.trim()"
          @click="addMaster"
        >
          <template #icon><i-tabler-plus /></template>
          添加 Master
        </n-button>
      </div>
    </template>
  </RepeatableList>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import RepeatableItem from '@/components/shared/RepeatableItem.vue';
import RepeatableList from '@/components/shared/RepeatableList.vue';
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
  const normalized = value.trim();
  if (
    normalized &&
    rows.value.some((row, rowIndex) => rowIndex !== index && row.trim() === normalized)
  ) {
    message.warning('该 Master 已存在');
    return;
  }
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
.master-list-add {
  display: flex;
  width: min(100%, 36rem);
  align-items: center;
  gap: var(--sd-space-xs);
}

.master-list-input {
  width: 100%;
  min-width: 0;
  max-width: 30rem;
}

@media (max-width: 640px) {
  .master-list-add {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
