<template>
  <RepeatableList add-label="添加 Master" @add="addMaster">
    <RepeatableItem
      v-for="(master, index) in rows"
      :key="index"
      :title="`Master ${index + 1}`"
      inline
      remove-label="删除该 Master"
      @remove="removeRow(index)"
    >
      <div class="master-list-control">
        <n-input
          :ref="instance => setInputRef(index, instance)"
          :value="master"
          placeholder="如 QQ:12345"
          @update:value="updateRow(index, $event)"
        />
      </div>
    </RepeatableItem>
  </RepeatableList>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import type { InputInst } from 'naive-ui';
import RepeatableItem from '@/components/shared/RepeatableItem.vue';
import RepeatableList from '@/components/shared/RepeatableList.vue';
import { normalizeMasterListValues } from '@/features/baseSetting/viewModel';

const message = useMessage();
const model = defineModel<string[]>({ required: true });

// 本地行副本：允许正在编辑的空行存在，提交到 model 时才做清理。
const rows = ref<string[]>([...model.value]);
const inputRefs = ref<Array<InputInst | null>>([]);

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

function setInputRef(index: number, instance: unknown) {
  inputRefs.value[index] = (instance as InputInst | null) ?? null;
}

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

async function addMaster() {
  const index = rows.value.length;
  commit([...rows.value, '']);
  await nextTick();
  inputRefs.value[index]?.focus();
}
</script>

<style scoped>
.master-list-control {
  width: 11rem;
  max-width: 100%;
}
</style>
