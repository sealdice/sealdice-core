<template>
  <n-descriptions
    label-placement="left"
    label-align="left"
    separator=" "
    :column="1"
    content-class="whitespace-pre-wrap break-words"
  >
    <n-descriptions-item>
      <template #label>
        <n-tag size="small" :bordered="false">引擎版本</n-tag>
      </template>
      {{ version }}
    </n-descriptions-item>
    <n-descriptions-item>
      <template #label>
        <n-tag size="small" :bordered="false">V2 预览</n-tag>
      </template>
      <div role="region" aria-label="V2 原文预览">{{ info.textV2 || info.errV2 }}</div>
    </n-descriptions-item>
    <n-descriptions-item v-if="$slots['v2-markdown']">
      <template #label><n-tag size="small" :bordered="false">V2 Markdown</n-tag></template>
      <slot name="v2-markdown" />
    </n-descriptions-item>
    <n-descriptions-item>
      <template #label>
        <n-tag size="small" :bordered="false">V1 预览</n-tag>
      </template>
      <div role="region" aria-label="V1 原文预览">{{ info.textV1 || info.errV1 }}</div>
    </n-descriptions-item>
    <n-descriptions-item v-if="$slots['v1-markdown']">
      <template #label><n-tag size="small" :bordered="false">V1 Markdown</n-tag></template>
      <slot name="v1-markdown" />
    </n-descriptions-item>
    <n-descriptions-item>
      <template #label>
        <n-tag size="small" :bordered="false">存在预设</n-tag>
      </template>
      {{ exists }} [存在时预览较为可靠]
    </n-descriptions-item>
  </n-descriptions>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { TextItemCompatibleInfo } from '@/api';

const props = defineProps<{
  info: TextItemCompatibleInfo;
}>();

const version = computed(() =>
  props.info.version === 'v1' ? 'v1 [建议修改]' : props.info.version
);
const exists = computed(() => (props.info.presetExists ? '是' : '否'));
</script>
