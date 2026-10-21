<template>
  <RepeatableList add-label="添加前缀" @add="addItem">
    <RepeatableItem
      v-for="(item, index) in rows"
      :key="index"
      inline
      remove-label="删除该前缀"
      @remove="removeRow(index)"
    >
      <template #leading>
        <span class="setting-string-list-index">前缀 {{ index + 1 }}</span>
      </template>
      <div class="setting-string-list-control">
        <n-input
          :ref="instance => setInputRef(index, instance)"
          :value="item"
          placeholder="如 . 或 !"
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
import { normalizeStringListValues } from '@/features/baseSetting/viewModel';

const message = useMessage();
const model = defineModel<string[]>({ required: true });

// 本地行副本：允许正在编辑的空行存在，提交到 model 时才做清理。
const rows = ref<string[]>([...model.value]);
const inputRefs = ref<Array<InputInst | null>>([]);

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

function setInputRef(index: number, instance: unknown) {
  inputRefs.value[index] = (instance as InputInst | null) ?? null;
}

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

async function addItem() {
  const index = rows.value.length;
  commit([...rows.value, '']);
  await nextTick();
  inputRefs.value[index]?.focus();
}
</script>

<style scoped>
/* 子项序号是附属信息，使用弱化排版，避免与上方字段标签争夺层级。 */
.setting-string-list-index {
  color: var(--sd-text-muted);
  font-size: 0.8rem;
  font-weight: 400;
  line-height: 1.4;
}

.setting-string-list-control {
  width: 8rem;
  max-width: 100%;
}
</style>
