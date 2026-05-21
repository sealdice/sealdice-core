<script setup lang="ts">
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';

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
  deleteCondition: [index: number];
}>();

const breakpoints = useBreakpoints(breakpointsTailwind);
const notMobile = breakpoints.greater('sm');
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

<template>
  <div
    v-for="(cond, index) in listModel"
    :key="conditionKeyOf(cond)"
    class="condition-item">
    <div class="condition-head">
      <div class="condition-fields">
        <label class="condition-field condition-mode">
          <span>模式</span>
          <n-select
            v-model:value="cond.condType"
            :options="condTypeOptions"
            size="small"
          />
        </label>

        <template v-if="cond.condType === 'textMatch'">
          <label class="condition-field condition-method">
            <span>
              方式
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-icon size="14">
                    <i-carbon-help-filled />
                  </n-icon>
                </template>
                匹配方式一览：<br/>精确匹配：完全相同时触发。<br/>任意相符：如aa|bb，则aa或bb都能触发。<br/>包含文本：包含此文本触发。<br/>不含文本：不包含此文本触发。<br/>模糊匹配：文本相似时触发<br/>正则匹配：正则表达式匹配<br/>前缀匹配：文本以内容为开头<br/>后缀匹配：文本以此内容为结尾
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
            <n-input v-model:value="(cond.value as string)" size="small" />
          </label>
        </template>

        <template v-else-if="cond.condType === 'exprTrue'">
          <label class="condition-field condition-value">
            <span>
              表达式
              <n-tooltip trigger="hover">
                <template #trigger>
                  <n-icon size="14">
                    <i-carbon-help-filled />
                  </n-icon>
                </template>
                举例：<br/>$t1 == '张三' // 正则匹配的第一个组内容是张三<br/>$m个人计数器 >= 10<br/>友情提醒，匹配失败时无提示，请先自行在「指令测试」测好
              </n-tooltip>
            </span>
            <n-input v-model:value="(cond.value as string)" type="textarea" size="small" :autosize="{ minRows: 1, maxRows: 10 }" />
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
            <n-input-number v-model:value="(cond.value as number)" :min="0" size="small" />
          </label>
        </template>
      </div>

      <n-button type="error" size="small" ghost @click="deleteByIndex(index)">
        <template #icon>
          <n-icon>
            <i-carbon-row-delete />
          </n-icon>
        </template>
        <template v-if="notMobile" #default> 删除条件 </template>
      </n-button>
    </div>
  </div>
</template>

<style scoped>
.condition-item {
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  border-radius: 6px;
  background: var(--sd-bg-elevated);
  padding: 0.65rem;
}

.condition-item + .condition-item {
  margin-top: 0.75rem;
}

.condition-head,
.condition-fields {
  display: flex;
  min-width: 0;
}

.condition-head {
  align-items: flex-end;
  justify-content: space-between;
  gap: 0.75rem;
}

.condition-fields {
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

  .condition-head,
  .condition-fields {
    align-items: stretch;
    flex-direction: column;
  }

  .condition-mode,
  .condition-method {
    width: 100%;
    flex-basis: auto;
  }

  .condition-number {
    width: 100%;
    flex-basis: auto;
  }
}
</style>
