<template>
  <div class="help-config-tags">
    <n-flex align="center" wrap>
      <n-tag size="small" :bordered="false">{{ props.group.value }}</n-tag>
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
    </n-flex>

    <n-flex v-if="inputVisible" align="center" size="small" wrap>
      <n-input
        ref="inputRef"
        v-model:value="inputValue"
        size="small"
        class="alias-input"
        placeholder="输入新别名"
        @keyup.enter="confirmInput"
      />
      <n-button type="primary" size="small" @click="confirmInput">确定</n-button>
      <n-button size="small" @click="cancelInput">取消</n-button>
    </n-flex>
    <n-button v-else type="primary" secondary size="small" class="alias-add" @click="showInput">
      <template #icon>
        <n-icon><i-tabler-plus /></n-icon>
      </template>
      添加别名
    </n-button>
  </div>
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
  cancelInput();
}

function cancelInput() {
  inputVisible.value = false;
  inputValue.value = '';
}
</script>

<style scoped>
.help-config-tags {
  display: flex;
  min-height: 28px;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--sd-space-xs);
}

.alias-input {
  width: min(100%, 16rem);
  min-width: 8rem;
}

.alias-add {
  align-self: flex-start;
}
</style>
