<template>
  <div class="flex items-center justify-between px-4">
    <n-space>
      <template v-if="changed">
        <n-icon size="18" color="var(--n-primary-color)">
          <i-tabler-info-circle />
        </n-icon>
        <n-text type="info"> 变更如下： </n-text>
      </template>
      <template v-else>
        <n-icon size="18" color="var(--n-text-color-3)">
          <i-tabler-info-circle />
        </n-icon>
        <n-text type="tertiary"> 无变更 </n-text>
      </template>
    </n-space>

    <n-space v-if="changed" vertical align="end">
      <n-radio-group v-model:value="split" size="small" aria-label="对比布局">
        <n-radio-button :value="false">单列</n-radio-button>
        <n-radio-button :value="true">双列</n-radio-button>
      </n-radio-group>
      <n-checkbox v-model:checked="folding">折叠无变更</n-checkbox>
    </n-space>
  </div>

  <div v-show="split" class="flex items-center justify-around py-2">
    <h3 class="pl-8">原内容</h3>
    <n-icon size="18">
      <i-tabler-arrow-right />
    </n-icon>
    <h3 class="pr-8">新内容</h3>
  </div>

  <div v-if="changed" class="diff-viewer-code">
    <VueDiff
      :mode="mode"
      :theme="props.theme"
      :language="props.lang"
      :folding="folding"
      :prev="props.old"
      :current="props.new"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import VueDiff from 'vue-diff';
import 'vue-diff/dist/index.css';

interface Props {
  old: string;
  new: string;
  lang?: string;
  theme?: 'light' | 'dark';
}

const props = withDefaults(defineProps<Props>(), {
  lang: 'text',
  old: '',
  new: '',
  theme: 'light',
});

const changed = computed(() => props.old !== props.new);
const split = ref(false);
const folding = ref(false);
const mode = computed(() => (split.value ? 'split' : 'unified'));
</script>

<style scoped>
.diff-viewer-code :deep(.vue-diff-viewer),
.diff-viewer-code :deep(code) {
  font-family: var(--sd-font-code);
}
</style>
