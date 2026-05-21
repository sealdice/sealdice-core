<script setup lang="ts">
import { computed } from 'vue';
import type { TextItemCompatibleInfo } from '@/api';

const props = defineProps<{
  info: TextItemCompatibleInfo;
}>();

const version = computed(() => props.info.version === 'v1' ? 'v1 [建议修改]' : props.info.version);
const exists = computed(() => props.info.presetExists ? '是' : '否');
</script>

<template>
  <n-descriptions
    label-placement="left"
    label-align="left"
    separator=" "
    :column="1"
    content-class="whitespace-nowrap break-words"
  >
    <n-descriptions-item>
      <template #label>
        <n-tag type="success" size="small" :bordered="false">引擎版本</n-tag>
      </template>
      {{ version }}
    </n-descriptions-item>
    <n-descriptions-item>
      <template #label>
        <n-tag type="info" size="small" :bordered="false">V2 预览</n-tag>
      </template>
      {{ info.textV2 || info.errV2 }}
    </n-descriptions-item>
    <n-descriptions-item>
      <template #label>
        <n-tag type="warning" size="small" :bordered="false">V1 预览</n-tag>
      </template>
      {{ info.textV1 || info.errV1 }}
    </n-descriptions-item>
    <n-descriptions-item>
      <template #label>
        <n-tag type="success" size="small" :bordered="false">存在预设</n-tag>
      </template>
      {{ exists }} [存在时预览较为可靠]
    </n-descriptions-item>
  </n-descriptions>
</template>
