<script setup lang="ts">
import { reactive } from 'vue';
import draggable from 'vuedraggable';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import ConditionBuilder from './ConditionBuilder.vue';

interface ReplyCondition {
  condType: string;
  value: string | number | undefined;
  matchType?: string;
  matchOp?: string;
}

type ReplyMessage = [string, number];

interface ReplyResult {
  resultType: string;
  delay: number;
  message: ReplyMessage[];
}

interface ReplyTask {
  name?: string;
  enable: boolean;
  conditions: ReplyCondition[];
  results: ReplyResult[];
}

const props = withDefaults(
  defineProps<{
    tasks: ReplyTask[];
    startIndex?: number;
  }>(),
  {
    startIndex: 0,
  },
);
const emit = defineEmits<{
  change: [];
  deleteRule: [index: number];
}>();

const breakpoints = useBreakpoints(breakpointsTailwind);
const notMobile = breakpoints.greater('sm');
const foldedRules = reactive<Record<number, boolean>>({});

const resultTypeOptions = [
  { label: '回复', value: 'replyToSender' },
  { label: '私聊回复', value: 'replyPrivate' },
  { label: '群内回复', value: 'replyGroup' },
];

let taskKeySeed = 0;
const taskKeys = new WeakMap<ReplyTask, string>();

function getTaskKey(task: ReplyTask) {
  let key = taskKeys.get(task);
  if (!key) {
    key = `reply-rule-${props.startIndex}-${taskKeySeed++}`;
    taskKeys.set(task, key);
  }
  return key;
}

function toggleFold(index: number) {
  const absoluteIndex = props.startIndex + index;
  foldedRules[absoluteIndex] = !foldedRules[absoluteIndex];
}

function isFolded(index: number) {
  return foldedRules[props.startIndex + index] === true;
}

function summarizeRule(task: ReplyTask) {
  const firstCondition = task.conditions?.[0];
  if (!firstCondition) {
    return '无条件';
  }
  if (firstCondition.condType === 'textMatch') {
    return `文本匹配：${String(firstCondition.value ?? '')}`;
  }
  if (firstCondition.condType === 'exprTrue') {
    return `表达式：${String(firstCondition.value ?? '')}`;
  }
  if (firstCondition.condType === 'textLenLimit') {
    const op = firstCondition.matchOp === 'ge' ? '大于等于' : '小于等于';
    return `长度${op}${String(firstCondition.value ?? '')}`;
  }
  return '自定义条件';
}

const deleteItem = (index: number) => {
  emit('deleteRule', index);
};

const deleteAnyItem = <T,>(lst: T[], index: number) => {
  lst.splice(index, 1);
  emit('change');
};

const addCond = (condList: ReplyCondition[]) => {
  condList.push({
    condType: 'textMatch',
    matchType: 'matchExact',
    value: '要匹配的文本',
  });
  emit('change');
};

const addResult = (results: ReplyResult[]) => {
  results.push({ resultType: 'replyToSender', delay: 0, message: [['说点什么', 1]] });
  emit('change');
};

const addMessageItem = (messages: ReplyMessage[]) => {
  messages.push(['怎么辉石呢', 1]);
  emit('change');
};

const removeMessageItem = (messages: ReplyMessage[], index: number) => {
  messages.splice(index, 1);
  emit('change');
};
</script>

<template>
  <draggable
    class="drag-area"
    tag="div"
    :list="tasks"
    handle=".rule-drag-handle"
    :group="{ name: 'g1' }"
    :item-key="getTaskKey"
    @end="emit('change')">
    <template #item="{ element: el, index }">
      <article class="rule-panel" :class="{ 'is-collapsed': isFolded(index), 'is-disabled': !el.enable }">
        <header class="rule-panel-head">
          <div class="rule-title">
            <span class="rule-index">#{{ props.startIndex + index + 1 }}</span>
            <n-checkbox v-model:checked="el.enable" @update:checked="emit('change')">开启</n-checkbox>
            <span v-if="isFolded(index)" class="rule-summary">{{ summarizeRule(el) }}</span>
          </div>

          <div class="rule-actions">
            <n-button class="rule-drag-handle" size="small" quaternary circle>
              <template #icon>
                <n-icon><i-carbon-move /></n-icon>
              </template>
            </n-button>
            <n-button type="error" size="small" secondary @click="deleteItem(index)">
              <template #icon>
                <n-icon><i-carbon-row-delete /></n-icon>
              </template>
              <template v-if="notMobile" #default>删除</template>
            </n-button>
            <n-button size="small" quaternary circle @click="toggleFold(index)">
              <template #icon>
                <n-icon>
                  <i-carbon-chevron-right v-if="isFolded(index)" />
                  <i-carbon-chevron-down v-else />
                </n-icon>
              </template>
            </n-button>
          </div>
        </header>

        <div v-if="!isFolded(index)" class="rule-panel-body">
          <section class="rule-block condition-block">
            <div class="rule-block-head">
              <div>
                <h4>条件</h4>
                <p>需同时满足，即 and</p>
              </div>
              <n-button type="success" size="small" secondary @click="addCond(el.conditions)">
                <template #icon>
                  <n-icon><i-carbon-add-large /></n-icon>
                </template>
                增加
              </n-button>
            </div>

            <div class="rule-block-body">
              <ConditionBuilder v-if="el.conditions?.length" v-model="el.conditions" />
              <n-empty v-else description="当前无条件" size="small" />
            </div>
          </section>

          <section class="rule-block result-block">
            <div class="rule-block-head">
              <div>
                <h4>结果</h4>
                <p>顺序执行</p>
              </div>
              <n-button type="success" size="small" secondary @click="addResult(el.results)">
                <template #icon>
                  <n-icon><i-carbon-add-large /></n-icon>
                </template>
                增加
              </n-button>
            </div>

            <div class="rule-block-body">
              <n-empty v-if="!el.results?.length" description="当前无结果" size="small" />
              <article v-for="(result, rIdx) in el.results || []" :key="rIdx" class="result-panel">
                <div class="result-head">
                  <div class="result-fields">
                    <label class="result-field result-mode">
                      <span>模式</span>
                      <n-select
                        v-model:value="result.resultType"
                        :options="resultTypeOptions"
                        size="small"
                        @update:value="emit('change')"
                      />
                    </label>

                    <label class="result-field result-delay">
                      <span>
                        延迟
                        <n-tooltip trigger="hover">
                          <template #trigger>
                            <n-icon size="14"><i-carbon-help-filled /></n-icon>
                          </template>
                          文本将在此延迟后发送，单位秒，可小数。<br />注意随机延迟仍会被加入，如果你希望保证发言顺序，记得考虑这点。
                        </n-tooltip>
                      </span>
                      <n-input-number
                        v-model:value="result.delay"
                        size="small"
                        @update:value="emit('change')"
                      />
                    </label>
                  </div>

                  <n-button type="error" size="small" secondary @click="deleteAnyItem(el.results, rIdx)">
                    <template #icon>
                      <n-icon><i-carbon-row-delete /></n-icon>
                    </template>
                    <template v-if="notMobile" #default>删除结果</template>
                  </n-button>
                </div>

                <div
                  v-if="['replyToSender', 'replyPrivate', 'replyGroup'].includes(result.resultType)"
                  class="result-content"
                >
                  <div class="result-options">
                    <n-text>回复文本（随机选择）</n-text>
                  </div>

                  <div v-for="(msg, mIdx) in result.message" :key="mIdx" class="message-row">
                    <n-tooltip trigger="hover">
                      <template #trigger>
                        <n-button
                          v-if="mIdx === 0"
                          type="success"
                          size="tiny"
                          quaternary
                          circle
                          @click="addMessageItem(result.message)"
                        >
                          <template #icon>
                            <n-icon><i-carbon-add-filled /></n-icon>
                          </template>
                        </n-button>
                        <n-button
                          v-else
                          type="error"
                          size="tiny"
                          quaternary
                          circle
                          @click="removeMessageItem(result.message, mIdx)"
                        >
                          <template #icon>
                            <n-icon><i-carbon-close-outline /></n-icon>
                          </template>
                        </n-button>
                      </template>
                      {{ mIdx === 0 ? '点击添加一个回复语，海豹将会随机抽取一个回复' : '点击删除你不想要的回复语' }}
                    </n-tooltip>
                    <n-input
                      v-model:value="msg[0]"
                      type="textarea"
                      class="reply-text"
                      :autosize="{ minRows: 1 }"
                      @update:value="emit('change')"
                    />
                  </div>
                </div>
              </article>
            </div>
          </section>
        </div>
      </article>
    </template>
  </draggable>
</template>

<style scoped>
.drag-area {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-height: 50px;
  overflow-x: hidden;
  padding-top: 0.25rem;
  padding-bottom: 0.5rem;
}

.rule-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--sd-border);
  border-radius: 6px;
  background: var(--sd-bg-elevated);
}

.rule-panel.is-disabled {
  opacity: 0.72;
}

.rule-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-width: 0;
  border-bottom: 1px solid var(--sd-border-soft);
  background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 62%);
  padding: 0.55rem 0.75rem;
}

.rule-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
}

.rule-index {
  flex: 0 0 auto;
  color: var(--sd-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.8rem;
}

.rule-summary {
  min-width: 0;
  overflow: hidden;
  color: var(--sd-text-muted);
  font-size: 0.85rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rule-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.35rem;
}

.rule-drag-handle {
  cursor: grab;
}

.rule-panel-body {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.75rem;
}

.rule-block {
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  border-radius: 6px;
  background: color-mix(in srgb, var(--sd-bg-elevated), var(--sd-bg-page) 34%);
  padding: 0.75rem;
}

.rule-block-head,
.result-head,
.result-options,
.result-fields,
.message-row {
  display: flex;
  min-width: 0;
}

.rule-block-head,
.result-head,
.result-options {
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.rule-block-head {
  margin-bottom: 0.65rem;
}

.rule-block-head h4 {
  margin: 0;
  font-size: 0.92rem;
}

.rule-block-head p {
  margin: 0.12rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.8rem;
}

.rule-block-body {
  min-width: 0;
}

.result-panel {
  min-width: 0;
  border: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated);
  padding: 0.7rem;
}

.result-panel + .result-panel {
  margin-top: 0.65rem;
}

.result-content {
  min-width: 0;
  margin-top: 0.65rem;
}

.result-fields {
  flex: 1 1 auto;
  align-items: flex-end;
  gap: 0.6rem;
}

.result-field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.25rem;
  color: var(--sd-text-secondary);
  font-size: 0.84rem;
}

.result-field > span {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  line-height: 1;
}

.result-mode {
  width: 9rem;
  flex: 0 0 9rem;
}

.result-delay {
  width: 6rem;
  flex: 0 0 6rem;
}

.message-row {
  align-items: flex-start;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.reply-text {
  flex: 1 1 auto;
  min-width: 0;
}

@media screen and (max-width: 700px) {
  .mobile-changeline {
    flex-direction: column;
  }

  .rule-panel-head,
  .rule-block-head,
  .result-head,
  .result-options,
  .result-fields {
    align-items: flex-start;
    flex-direction: column;
  }

  .result-mode,
  .result-delay {
    width: 100%;
    flex-basis: auto;
  }

  .rule-actions {
    align-self: flex-end;
  }
}
</style>
