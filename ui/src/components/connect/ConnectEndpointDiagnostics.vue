<template>
  <n-modal
    :show="true"
    preset="card"
    title="运行诊断"
    closable
    style="width: min(40rem, calc(100vw - 2rem))"
    @update:show="show => !show && emit('close')"
  >
    <div class="diagnostics" :aria-busy="loading">
      <div class="diagnostics__heading">
        <div>
          <n-text strong>{{ name }}</n-text>
          <p>查看核心返回的连接状态、登录进度和指令统计。</p>
        </div>
        <n-button size="small" secondary :loading="loading" @click="emit('refresh')">
          <template #icon>
            <n-icon><i-tabler-refresh /></n-icon>
          </template>
          刷新诊断
        </n-button>
      </div>

      <n-alert v-if="error" type="error">
        {{ error }}<template v-if="endpoint">。下方保留上次成功读取的数据。</template>
      </n-alert>

      <n-descriptions v-if="endpoint && state" :column="1" size="small" label-placement="left">
        <n-descriptions-item label="连接状态">
          <n-tag size="small" :type="state.tagType" :bordered="false">{{ state.text }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="是否启用">
          {{ endpoint.enable ? '已启用' : '已禁用' }}
        </n-descriptions-item>
        <n-descriptions-item label="登录流程">{{ workflow }}</n-descriptions-item>
        <n-descriptions-item v-for="[label, value] in metrics" :key="label" :label="label">
          {{ value }}
        </n-descriptions-item>
      </n-descriptions>
      <n-skeleton v-else-if="loading" text :repeat="4" />
      <n-empty v-else-if="!error" description="此账号已不在连接列表中" />
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { EndPointInfo } from '@/api';
import {
  getEndpointMetricRows,
  getEndpointStateMeta,
  getWorkflowText,
} from '@/features/connect/endpointDisplay';

const props = defineProps<{
  name: string;
  endpoint: EndPointInfo | null;
  loading: boolean;
  error: string;
}>();
const emit = defineEmits<{ close: []; refresh: [] }>();

const state = computed(() => (props.endpoint ? getEndpointStateMeta(props.endpoint.state) : null));
const workflow = computed(() => getWorkflowText(props.endpoint?.workflow ?? null) || '暂无记录');
const metrics = computed(() => (props.endpoint ? getEndpointMetricRows(props.endpoint) : []));
</script>

<style scoped>
.diagnostics {
  display: grid;
  gap: var(--sd-space-md);
  overflow-wrap: anywhere;
}

.diagnostics__heading {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--sd-space-sm);
}

.diagnostics__heading p {
  margin: var(--sd-space-xs) 0 0;
  color: var(--sd-text-secondary);
}
</style>
