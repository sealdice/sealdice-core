<template>
  <div class="flex items-center justify-between px-4">
    <n-space>
      <template v-if="changed">
        <n-icon size="18" color="var(--n-primary-color)">
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/><line x1="12" y1="8" x2="12" y2="12" stroke="currentColor" stroke-width="2"/><circle cx="12" cy="17" r="0.5" fill="currentColor"/></svg>
        </n-icon>
        <n-text type="info">
          变更如下：
        </n-text>
      </template>
      <template v-else>
        <n-icon size="18" color="var(--n-text-color-3)">
          <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/><line x1="12" y1="16" x2="12" y2="12" stroke="currentColor" stroke-width="2"/><circle cx="12" cy="17" r="0.5" fill="currentColor"/></svg>
        </n-icon>
        <n-text type="tertiary">
          无变更
        </n-text>
      </template>
    </n-space>

    <n-space v-if="changed" vertical align="center">
      <n-switch v-model:value="split">
        <template #checked>双列</template>
        <template #unchecked>单列</template>
      </n-switch>
      <n-checkbox v-model:checked="folding">折叠无变更</n-checkbox>
    </n-space>
  </div>

  <div v-show="split" class="flex items-center justify-around py-2">
    <h3 class="pl-8">原内容</h3>
    <n-icon size="18">
      <svg viewBox="0 0 24 24"><line x1="5" y1="12" x2="19" y2="12" stroke="currentColor" stroke-width="2"/><polyline points="12 5 19 12 12 19" fill="none" stroke="currentColor" stroke-width="2"/></svg>
    </n-icon>
    <h3 class="pr-8">新内容</h3>
  </div>

  <VueDiff
    v-if="changed"
    :mode="mode"
    theme="light"
    :language="props.lang"
    :folding="folding"
    :prev="props.old"
    :current="props.new"
  />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import VueDiff from 'vue-diff';
import 'vue-diff/dist/index.css';

interface Props {
  old: string;
  new: string;
  lang?: string;
}

const props = withDefaults(defineProps<Props>(), {
  lang: 'text',
  old: '',
  new: '',
});

const changed = computed(() => props.old !== props.new);
const split = ref(false);
const folding = ref(false);
const mode = computed(() => (split.value ? 'split' : 'unified'));
</script>
