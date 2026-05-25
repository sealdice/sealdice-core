<template>
  <n-flex align="center" wrap class="help-config-tags">
    <n-tag type="success" size="small" :bordered="false">{{ props.group.value }}</n-tag>
    <n-tag
      v-for="alias in groupAliases"
      :key="alias"
      size="small"
      closable
      :bordered="false"
      @close="emit('removeAlias', props.group.value, alias)"
    >
      {{ alias }}
    </n-tag>

    <n-input
      v-if="inputVisible"
      ref="inputRef"
      v-model:value="inputValue"
      size="tiny"
      autosize
      class="alias-input"
      @keyup.enter="confirmInput"
      @blur="confirmInput"
    />
    <n-button v-if="inputVisible" size="tiny" tertiary @click="confirmInput">
      确定
    </n-button>
    <n-button v-else size="tiny" tertiary @click="showInput">
      <template #icon>
        <n-icon><i-carbon-add-large /></n-icon>
      </template>
      新别名
    </n-button>
  </n-flex>
</template>

<script setup lang="ts">
import { computed, nextTick, shallowRef } from 'vue';

const props = defineProps<{
  group: {
    value: string;
    label: string;
  };
  aliases: Map<string, string[]>;
}>();

const emit = defineEmits<{
  addAlias: [group: string, alias: string];
  removeAlias: [group: string, alias: string];
}>();

const inputVisible = shallowRef(false);
const inputValue = shallowRef('');
const inputRef = shallowRef<HTMLInputElement | null>(null);
const groupAliases = computed(() => props.aliases.get(props.group.value) ?? []);

function showInput() {
  inputVisible.value = true;
  nextTick(() => {
    inputRef.value?.focus();
  });
}

function confirmInput() {
  const value = inputValue.value.trim();
  if (value) {
    emit('addAlias', props.group.value, value);
  }
  inputVisible.value = false;
  inputValue.value = '';
}
</script>

<style scoped>
.help-config-tags {
  min-height: 28px;
}

.alias-input {
  min-width: 6rem;
}
</style>
