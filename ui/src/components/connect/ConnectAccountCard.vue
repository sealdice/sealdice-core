<template>
  <article class="connect-account-card">
    <n-card size="small" :bordered="true">
      <template #header>
        <div class="connect-account-card__identity">
          <h2 class="connect-account-card__name">{{ displayName }}</h2>
          <div class="connect-account-card__meta">
            <span>{{ endpoint.userId || endpoint.id }}</span>
            <span>{{ protocolLabel }}</span>
          </div>
        </div>
      </template>

      <template #header-extra>
        <n-space size="small" wrap justify="end" class="connect-account-card__status-list">
          <n-tag size="small" :type="stateMeta.tagType" :bordered="false">
            {{ stateMeta.text }}
          </n-tag>
          <n-tag v-if="!endpoint.enable" size="small" type="warning" :bordered="false">
            已禁用
          </n-tag>
          <n-tag v-if="workflowTag" size="small" :type="workflowTag.type" :bordered="false">
            {{ workflowTag.text }}
          </n-tag>
        </n-space>
      </template>

      <dl class="connect-account-card__metrics">
        <div
          v-for="([label, value], index) in metrics"
          :key="label"
          class="connect-account-card__metric"
        >
          <dt>{{ label }}</dt>
          <dd :class="{ 'connect-account-card__metric-value--time': index === 2 }">{{ value }}</dd>
        </div>
      </dl>

      <n-descriptions
        class="connect-account-card__details"
        size="small"
        label-placement="left"
        :column="1"
      >
        <n-descriptions-item v-for="[label, value] in details" :key="label" :label="label">
          <span class="connect-account-card__detail-value">{{ value }}</span>
        </n-descriptions-item>
      </n-descriptions>

      <n-alert
        v-if="actionError"
        type="error"
        :show-icon="false"
        class="connect-account-card__error"
      >
        {{ actionError }}
      </n-alert>

      <template #footer>
        <div class="connect-account-card__footer">
          <n-text depth="3" class="connect-account-card__id">ID: {{ endpoint.id }}</n-text>
          <div class="connect-account-card__actions">
            <n-button v-if="qrCode" size="small" tertiary @click="emit('showQRCode', endpoint)">
              <template #icon
                ><n-icon><i-ep-picture /></n-icon
              ></template>
              二维码
            </n-button>
            <n-button
              size="small"
              secondary
              :disabled="writeDisabled"
              @click="emit('edit', endpoint)"
            >
              <template #icon
                ><n-icon><i-ep-edit /></n-icon
              ></template>
              修改
            </n-button>
            <n-button
              size="small"
              secondary
              :type="endpoint.enable ? 'warning' : 'success'"
              :loading="pendingAction === 'enable'"
              :disabled="writeDisabled"
              @click="emit('toggleEnable', endpoint, !endpoint.enable)"
            >
              <template #icon
                ><n-icon><i-ep-switch-button /></n-icon
              ></template>
              {{ endpoint.enable ? '禁用' : '启用' }}
            </n-button>
            <n-button
              size="small"
              type="error"
              secondary
              :loading="pendingAction === 'delete'"
              :disabled="writeDisabled"
              @click="emit('delete', endpoint)"
            >
              <template #icon
                ><n-icon><i-ep-delete /></n-icon
              ></template>
              删除
            </n-button>
          </div>
        </div>
        <n-text v-if="isTestMode" depth="3" class="connect-account-card__readonly-note">
          展示模式下不可修改账号
        </n-text>
      </template>
    </n-card>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { EndPointInfo, WorkflowResp } from '@/api';
import {
  adapterOf,
  getEndpointDetailRows,
  getEndpointMetricRows,
  getEndpointProtocolLabel,
  getEndpointStateMeta,
  getWorkflowTag,
} from '@/features/connect/endpointDisplay';

const props = withDefaults(
  defineProps<{
    endpoint: EndPointInfo;
    workflow: WorkflowResp | null;
    qrCode?: string;
    isTestMode: boolean;
    pendingAction?: 'enable' | 'delete' | null;
    hasPendingOperation: boolean;
    actionError?: string;
  }>(),
  {
    qrCode: '',
    pendingAction: null,
    actionError: '',
  }
);

const emit = defineEmits<{
  showQRCode: [endpoint: EndPointInfo];
  edit: [endpoint: EndPointInfo];
  toggleEnable: [endpoint: EndPointInfo, enable: boolean];
  delete: [endpoint: EndPointInfo];
}>();

const displayName = computed(
  () => props.endpoint.nickname || props.endpoint.userId || props.endpoint.id
);
const stateMeta = computed(() => getEndpointStateMeta(props.endpoint.state));
const workflowTag = computed(() => getWorkflowTag(props.workflow));
const metrics = computed(() => getEndpointMetricRows(props.endpoint));
const details = computed(() => getEndpointDetailRows(props.endpoint, props.workflow));
const protocolLabel = computed(() =>
  getEndpointProtocolLabel({
    platform: props.endpoint.platform,
    protocolType: props.endpoint.protocolType,
    adapter: adapterOf(props.endpoint),
  })
);
const writeDisabled = computed(() => props.isTestMode || props.hasPendingOperation);
</script>

<style scoped>
.connect-account-card {
  min-width: 0;
}

.connect-account-card :deep(.n-card) {
  height: 100%;
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated);
}

.connect-account-card :deep(.n-card__header) {
  align-items: flex-start;
  padding-bottom: var(--sd-space-sm);
}

.connect-account-card :deep(.n-card__header-extra) {
  max-width: 52%;
  margin-left: var(--sd-space-sm);
}

.connect-account-card :deep(.n-card__footer) {
  padding-top: var(--sd-space-sm);
}

.connect-account-card__identity {
  min-width: 0;
}

.connect-account-card__name {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--sd-text-primary);
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.4;
}

.connect-account-card__meta,
.connect-account-card__footer,
.connect-account-card__actions {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--sd-space-xs);
}

.connect-account-card__meta {
  margin-top: 0.25rem;
  color: var(--sd-text-muted);
  font-size: 0.8125rem;
  overflow-wrap: anywhere;
}

.connect-account-card__status-list {
  justify-content: flex-end;
}

.connect-account-card__metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sd-space-xs);
  margin: 0;
  padding: var(--sd-space-sm) 0;
  border-top: 1px solid var(--sd-border-soft);
  border-bottom: 1px solid var(--sd-border-soft);
}

.connect-account-card__metric {
  min-width: 0;
}

.connect-account-card__metric dt {
  color: var(--sd-text-muted);
  font-size: 0.75rem;
  line-height: 1.4;
}

.connect-account-card__metric dd {
  margin: 0.2rem 0 0;
  overflow-wrap: anywhere;
  color: var(--sd-text-primary);
  font-size: 0.9375rem;
  font-weight: 600;
  line-height: 1.35;
}

.connect-account-card__metric-value--time {
  font-size: 0.8125rem !important;
}

.connect-account-card__details {
  margin-top: var(--sd-space-sm);
}

.connect-account-card__detail-value {
  overflow-wrap: anywhere;
}

.connect-account-card__error {
  margin-top: var(--sd-space-sm);
}

.connect-account-card__footer {
  justify-content: space-between;
}

.connect-account-card__id {
  overflow-wrap: anywhere;
}

.connect-account-card__actions {
  justify-content: flex-end;
}

.connect-account-card__readonly-note {
  display: block;
  margin-top: var(--sd-space-xs);
}

@media (max-width: 639.9px) {
  .connect-account-card :deep(.n-card__header) {
    flex-wrap: wrap;
    gap: var(--sd-space-xs);
  }

  .connect-account-card :deep(.n-card__header-extra) {
    max-width: none;
    margin-left: 0;
  }

  .connect-account-card__metrics {
    gap: 0.375rem;
  }

  .connect-account-card__metric dd {
    font-size: 0.875rem;
  }

  .connect-account-card__footer,
  .connect-account-card__actions {
    align-items: stretch;
    justify-content: flex-start;
  }

  .connect-account-card__actions {
    width: 100%;
  }

  .connect-account-card__actions :deep(.n-button) {
    min-height: 2.75rem;
  }
}
</style>
