<template>
  <div class="condition-list" role="list">
    <RepeatableItem
      v-for="(cond, index) in listModel"
      :key="conditionKeyOf(cond)"
      :title="`条件 ${index + 1}`"
      remove-label="删除条件"
      @remove="deleteByIndex(index)"
    >
      <div class="condition-fields">
        <label class="condition-field condition-mode">
          <span>模式</span>
          <n-select v-model:value="cond.condType" :options="condTypeOptions" size="small" />
        </label>

        <template v-if="cond.condType === 'textMatch'">
          <label class="condition-field condition-method">
            <span>
              方式
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-icon size="14">
                    <i-tabler-help-circle />
                  </n-icon>
                </template>
                匹配方式一览：<br />精确匹配：完全相同时触发。<br />任意相符：如
                <n-text code class="sd-code-text">aa|bb</n-text>
                ，则 aa 或 bb 都能触发。<br />包含文本：包含此文本触发。<br />不含文本：不包含此文本触发。<br />模糊匹配：文本相似时触发<br />正则匹配：正则表达式匹配<br />前缀匹配：文本以内容为开头<br />后缀匹配：文本以此内容为结尾
              </n-tooltip>
            </span>
            <n-select
              v-model:value="cond.matchType"
              :options="matchTypeOptions"
              size="small"
              placeholder="选择方式"
            />
          </label>

          <label class="condition-field condition-value">
            <span>内容</span>
            <n-input
              v-model:value="cond.value as string"
              size="small"
              :class="{ 'sd-code-text': cond.matchType === 'matchRegex' }"
            />
          </label>
        </template>

        <template v-else-if="cond.condType === 'exprTrue'">
          <label class="condition-field condition-value">
            <span>
              表达式
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-icon size="14">
                    <i-tabler-help-circle />
                  </n-icon>
                </template>
                举例：<br />
                <span class="sd-code-text">$t1 == '张三'</span> // 正则匹配的第一个组内容是张三<br />
                <span class="sd-code-text">$m个人计数器 >= 10</span
                ><br />友情提醒，匹配失败时无提示，请先自行在「指令测试」测好
              </n-tooltip>
            </span>
            <n-input
              v-model:value="cond.value as string"
              type="textarea"
              size="small"
              class="sd-code-text"
              :autosize="{ minRows: 1, maxRows: 10 }"
            />
          </label>
        </template>

        <template v-else-if="cond.condType === 'textLenLimit'">
          <label class="condition-field condition-method">
            <span>方式</span>
            <n-select
              v-model:value="cond.matchOp"
              :options="matchOpOptions"
              size="small"
              placeholder="选择方式"
            />
          </label>

          <label class="condition-field condition-number">
            <span>长度</span>
            <n-input-number v-model:value="cond.value as number" :min="0" size="small" />
          </label>
        </template>
      </div>
    </RepeatableItem>
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue';
import RepeatableItem from '@/components/shared/RepeatableItem.vue';

interface ReplyCondition {
  condType: string;
  value: string | number | undefined;
  matchType: string;
  matchOp?: string;
}

const matchTypeOptions = [
  { label: '精确匹配', value: 'matchExact' },
  { label: '任意相符', value: 'matchMulti' },
  { label: '包含文本', value: 'matchContains' },
  { label: '不含文本', value: 'matchNotContains' },
  { label: '模糊匹配', value: 'matchFuzzy' },
  { label: '正则匹配', value: 'matchRegex' },
  { label: '前缀匹配', value: 'matchPrefix' },
  { label: '后缀匹配', value: 'matchSuffix' },
];

const condTypeOptions = [
  { label: '文本匹配', value: 'textMatch' },
  { label: '文本长度', value: 'textLenLimit' },
  { label: '表达式为真', value: 'exprTrue' },
];

const matchOpOptions = [
  { label: '大于等于', value: 'ge' },
  { label: '小于等于', value: 'le' },
];

const listModel = defineModel<ReplyCondition[]>();
const emit = defineEmits<{
  change: [];
  deleteCondition: [index: number];
}>();

watch(listModel, () => emit('change'), { deep: true });

const conditionKeys = new WeakMap<ReplyCondition, string>();
let nextConditionKey = 0;

const conditionKeyOf = (condition: ReplyCondition): string => {
  const existing = conditionKeys.get(condition);
  if (existing) return existing;
  nextConditionKey += 1;
  const key = `condition-${nextConditionKey}`;
  conditionKeys.set(condition, key);
  return key;
};

const deleteByIndex = (index: number) => {
  emit('deleteCondition', index);
};
</script>

<style scoped>
.condition-list {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: var(--sd-item-stack-gap);
}

.condition-fields {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  align-items: flex-end;
  gap: 0.6rem;
}

.condition-field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.25rem;
  color: var(--sd-text-secondary);
  font-size: 0.84rem;
}

.condition-field > span {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  line-height: 1;
}

.condition-mode {
  width: 8rem;
  flex: 0 0 8rem;
}

.condition-method {
  width: 8rem;
  flex: 0 0 8rem;
}

.condition-value {
  flex: 1 1 auto;
  min-width: 0;
}

.condition-number {
  width: 8rem;
  flex: 0 0 8rem;
}

@media screen and (max-width: 700px) {
  .mobile-changeline {
    flex-direction: column;
  }

  .condition-list {
    gap: 0.5rem;
  }

  .condition-fields {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    align-items: end;
    gap: 0.5rem;
  }

  .condition-value,
  .condition-number {
    grid-column: 1 / -1;
  }

  .condition-mode,
  .condition-method,
  .condition-value,
  .condition-number {
    width: 100%;
    flex-basis: auto;
  }
}
</style>
