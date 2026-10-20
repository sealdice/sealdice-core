<template>
  <McMention
    v-model="mentionOpen"
    :prefix="props.prefixes"
    :options-count="props.options.length"
    menu-class="tool-test-command-mention"
    @search-change="handleSearchChange"
  >
    <McInput
      ref="inputRef"
      class="tool-test-command-composer__input sd-code-text"
      :value="props.modelValue"
      :loading="props.loading"
      :disabled="props.disabled"
      :placeholder="props.placeholder || '输入消息，回车发送'"
      :max-length="2000"
      @change="updateInput"
      @submit="submitInput"
    >
      <template #head>
        <div class="tool-test-command-composer__prefixes">
          <span class="tool-test-command-composer__label">指令前缀</span>
          <div class="tool-test-command-composer__values">
            <code v-for="prefix in props.prefixes" :key="prefix">{{ prefix }}</code>
          </div>
          <span class="tool-test-command-composer__hint">当前上下文</span>
        </div>
      </template>
    </McInput>
    <template #menu>
      <McList
        :data="props.options"
        :input-el="inputElement"
        enable-short-key
        @select="selectOption"
      />
    </template>
  </McMention>
</template>

<script setup lang="ts">
import { nextTick, onMounted, shallowRef } from 'vue';
import { McInput } from '@matechat/core/Input';
import { McList } from '@matechat/core/List';
import { McMention } from '@matechat/core/Mention';
import type { ToolTestCommandOption } from '@/features/toolTest/model';

const props = defineProps<{
  modelValue: string;
  options: ToolTestCommandOption[];
  prefixes: string[];
  loading?: boolean;
  disabled?: boolean;
  placeholder?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
  submit: [value: string];
}>();

const inputRef = shallowRef<InstanceType<typeof McInput> | null>(null);
const inputElement = shallowRef<HTMLInputElement | HTMLTextAreaElement>();
const mentionOpen = shallowRef(false);
const mentionRange = shallowRef({ triggerIndex: -1, cursorIndex: -1 });

function handleSearchChange(event: { triggerIndex: number; cursorIndex: number }) {
  mentionRange.value = {
    triggerIndex: event.triggerIndex,
    cursorIndex: event.cursorIndex,
  };
  if (!props.options.length) mentionOpen.value = false;
}

function selectOption(option: ToolTestCommandOption) {
  const { triggerIndex, cursorIndex } = mentionRange.value;
  if (triggerIndex < 0 || cursorIndex < triggerIndex) return;

  const nextValue = `${props.modelValue.slice(0, triggerIndex)}${option.value}${props.modelValue.slice(cursorIndex)}`;
  emit('update:modelValue', nextValue);
  mentionOpen.value = false;

  void nextTick(() => {
    const input = inputElement.value;
    if (!input) return;
    input.focus();
    const cursor = triggerIndex + option.value.length;
    input.setSelectionRange(cursor, cursor);
  });
}

function updateInput(value: string) {
  emit('update:modelValue', value);
}

function submitInput(value: string) {
  emit('submit', value);
}

onMounted(() => {
  inputElement.value = inputRef.value?.getInput() as
    | HTMLInputElement
    | HTMLTextAreaElement
    | undefined;
});
</script>

<style scoped>
.tool-test-command-composer__prefixes {
  display: flex;
  min-height: 1.75rem;
  align-items: center;
  gap: 0.55rem;
  padding: 0 16px 0.55rem;
  border-bottom: 1px solid var(--devui-dividing-line);
  color: var(--devui-text);
  font-size: 0.75rem;
}

.tool-test-command-composer__label {
  flex: 0 0 auto;
  color: var(--devui-text);
  font-weight: 600;
}

.tool-test-command-composer__values {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.tool-test-command-composer__values code {
  padding: 0.16rem 0.45rem;
  border: 1px solid var(--devui-dividing-line);
  border-radius: var(--devui-border-radius, var(--sd-radius-xs));
  color: var(--devui-primary);
  background: var(--devui-list-item-hover-bg);
  font-family: var(--sd-font-code);
  font-weight: 600;
}

.tool-test-command-composer__input :deep(.mc-textarea),
.tool-test-command-composer__input :deep(.editable-container) {
  font-family: var(--sd-font-code);
}

.tool-test-command-composer__hint {
  margin-left: auto;
  color: var(--devui-light-text);
  font-size: 0.75rem;
}

:global(.tool-test-command-mention) {
  --devui-border-radius: var(--sd-radius-md);
  --devui-connected-overlay-bg: var(--sd-bg-elevated);
  --devui-dividing-line: var(--sd-border-soft);
  --devui-font-size: 0.875rem;
  --devui-gray-form-control-bg: var(--sd-bg-control);
  --devui-list-item-active-bg: var(--sd-bg-selected);
  --devui-list-item-active-text: var(--sd-text-primary);
  --devui-list-item-hover-bg: var(--sd-bg-hover);
  --devui-list-item-hover-text: var(--sd-text-primary);
  --devui-text: var(--sd-text-primary);
  padding: 0.35rem;
}

@media (max-width: 640px) {
  .tool-test-command-composer__hint {
    display: none;
  }
}
</style>
