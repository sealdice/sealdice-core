<template>
  <main class="connect-page">
    <div class="page-head">
      <h4>账号设置</h4>
      <n-button type="primary" :disabled="isTestMode" @click="openCreateDialog">
        添加账号
      </n-button>
    </div>

    <n-alert v-if="realtimeErrorText" type="error" class="mb-4">
      {{ realtimeErrorText }}
    </n-alert>

    <n-alert v-if="connectionsErrorText" type="error" class="mb-4">
      {{ connectionsErrorText }}
    </n-alert>

    <n-empty v-if="connections.length === 0 && connectionsReady" description="似乎还没有账号">
      <template #extra>
        <n-button type="primary" :disabled="isTestMode" @click="openCreateDialog">
          添加账号
        </n-button>
      </template>
    </n-empty>

    <n-data-table
      v-else
      :columns="columns"
      :data="connections"
      :loading="connectionsLoading"
      :bordered="false"
      :scroll-x="780"
      size="small"
    />

    <n-modal
      v-model:show="dialogVisible"
      preset="dialog"
      title="添加账号"
      class="account-dialog wizard-dialog"
      :show-icon="false"
      :mask-closable="false"
      @after-leave="resetWizard"
    >
      <ConnectCreateWizard
        v-model:form-model="formModel"
        v-model:wizard-step="wizardStep"
        v-model:wizard-platform="wizardPlatform"
        v-model:wizard-method="wizardMethod"
        v-model:wizard-protocol="wizardProtocol"
        :protocols="protocols"
        :schemas-error="Boolean(schemasQuery.error.value)"
        :selected-protocol="selectedProtocol"
        :selected-schema="selectedSchema"
        :sign-info-state="signInfoState"
        :sign-info-error-message="signInfoErrorMessage"
        :sign-version-options="signVersionOptions"
        :sign-servers="signServers"
        :is-mobile="isMobile"
        :can-submit="wizardCanNext"
        :submitting="createMutation.isPending.value"
        :official-qq-mode="officialQQMode"
        :test-mode-disabled="isTestMode"
        @cancel="dialogVisible = false"
        @select-platform="selectPlatform"
        @select-method="selectMethod"
        @select-protocol="selectProtocol"
        @previous="goPrev"
        @next="goNext"
        @submit="submit"
        @update:official-qq-mode="officialQQMode = $event"
        @retry-sign-info="retrySignInfo"
      />
    </n-modal>

    <ConnectEditDialog
      :visible="editDialogVisible"
      :config="editingConfig"
      :form-model="editFormModel"
      :schema="editSchema"
      :loading="editConfigQuery.isFetching.value"
      :error-message="editConfigErrorText"
      :is-mobile="isMobile"
      :saving="updateMutation.isPending.value"
      :disabled="isTestMode"
      :can-submit="canSubmitEdit"
      @update:visible="editDialogVisible = $event"
      @update:form-model="editFormModel = $event"
      @retry="retryEditConfig"
      @submit="submitEdit"
    />

    <n-modal
      v-model:show="qrDialogVisible"
      preset="dialog"
      title="登录二维码"
      class="qrcode-dialog"
      :show-icon="false"
    >
      <n-space vertical align="center" size="large">
        <n-image v-if="activeQRCode" :src="activeQRCode" width="280" preview-disabled />
        <n-empty v-else description="当前没有可用二维码" />
        <n-button size="small" secondary @click="realtimeConnections.reconnect">
          刷新连接
        </n-button>
      </n-space>
    </n-modal>
  </main>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useDialog, useMessage } from 'naive-ui';
import type {
  EditableConfigResp,
  EndPointInfo,
  FormConfigItem,
  MethodTreeNode,
  PlatformTreeNode,
  ProtocolDefinition,
} from '@/api';
import ConnectCreateWizard from '@/components/connect/ConnectCreateWizard.vue';
import ConnectEditDialog from '@/components/connect/ConnectEditDialog.vue';
import {
  buildDynamicFormInitialModel,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import { createConnectTableColumns } from '@/components/connect/ConnectTableColumns';
import { getEndpointTargetLabel } from '@/features/connect/endpointDisplay';
import {
  advanceConnectWizard,
  buildConnectCreatePayload,
  createConnectWizardState,
  resetConnectWizard,
  selectConnectMethod,
  selectConnectPlatform,
  selectConnectProtocol,
  validateConnectWizardForm,
} from '@/features/connect/createWizard';
import {
  useConnectEndpointConfigQuery,
  useConnectConnectionsQuery,
  useConnectProtocolsQuery,
  useConnectSchemasQuery,
} from '@/features/connect/queries';
import { useConnectMutations } from '@/features/connect/mutations';
import { useConnectSignInfo } from '@/features/connect/signInfo';
import { useRealtimeConnections } from '@/features/connect/realtime';
import type { OfficialQQMode } from '@/features/connect/officialQQ';
import { useTestMode } from '@/features/testMode/state';

const message = useMessage();
const dialog = useDialog();
const realtimeConnections = useRealtimeConnections();
const { isTestMode } = useTestMode();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

const dialogVisible = ref(false);
const editDialogVisible = ref(false);
const qrDialogVisible = ref(false);
const qrDialogEndpointId = ref('');
const editingEndpoint = ref<EndPointInfo | null>(null);
const editingConfig = ref<EditableConfigResp | null>(null);
const editFormModel = ref<DynamicFormModel>({});

const wizardState = reactive(createConnectWizardState());
const wizardStep = computed({
  get: () => wizardState.step,
  set: value => {
    wizardState.step = value;
  },
});
const wizardPlatform = computed<PlatformTreeNode | null>({
  get: () => wizardState.platform,
  set: value => {
    wizardState.platform = value;
  },
});
const wizardMethod = computed<MethodTreeNode | null>({
  get: () => wizardState.method,
  set: value => {
    wizardState.method = value;
  },
});
const wizardProtocol = computed<ProtocolDefinition | null>({
  get: () => wizardState.protocol,
  set: value => {
    wizardState.protocol = value;
  },
});
const formModel = computed<DynamicFormModel>({
  get: () => wizardState.formModel,
  set: value => {
    wizardState.formModel = value;
  },
});
const officialQQMode = computed<OfficialQQMode>({
  get: () => wizardState.officialQQMode,
  set: value => {
    wizardState.officialQQMode = value;
  },
});

const connectionsQuery = useConnectConnectionsQuery();
const protocolsQuery = useConnectProtocolsQuery();
const schemasQuery = useConnectSchemasQuery();
const protocols = computed<PlatformTreeNode[]>(() => protocolsQuery.data.value?.item.items ?? []);
const schemas = computed<Record<string, FormConfigItem[]>>(() => {
  const raw = schemasQuery.data.value?.item ?? {};
  return Object.fromEntries(
    Object.entries(raw).map(([key, value]) => [key, value ?? []])
  ) as Record<string, FormConfigItem[]>;
});
const connections = realtimeConnections.connections;

const selectedProtocol = computed(() => wizardState.protocol);
const selectedSchema = computed<FormConfigItem[]>(() => {
  const schemaKey = selectedProtocol.value?.schemaKey;
  return schemaKey ? (schemas.value[schemaKey] ?? []) : [];
});
const editSchema = computed<FormConfigItem[]>(() => editingConfig.value?.schema ?? []);

const signInfo = useConnectSignInfo(selectedProtocol, formModel);
const signInfoState = signInfo.state;
const signInfoErrorMessage = signInfo.errorMessage;
const signVersionOptions = signInfo.versionOptions;
const signServers = signInfo.servers;

const canSubmit = computed(
  () => validateConnectWizardForm(wizardState, selectedSchema.value).valid
);
const canSubmitEdit = computed(
  () => validateDynamicFormModel(editSchema.value, editFormModel.value).valid
);
const activeQRCode = computed(
  () => realtimeConnections.qrCodes.value[qrDialogEndpointId.value] ?? ''
);
const realtimeErrorText = computed(() =>
  realtimeConnections.lastError.value ? '实时连接异常，账号状态可能延迟。' : ''
);
const connectionsReady = computed(
  () =>
    realtimeConnections.ready.value ||
    connectionsQuery.isSuccess.value ||
    connectionsQuery.isError.value
);
const connectionsErrorText = computed(() =>
  connectionsQuery.isError.value && !realtimeConnections.ready.value
    ? getErrorMessage(connectionsQuery.error.value, '账号列表读取失败')
    : ''
);
const connectionsLoading = computed(() => hasAccessToken.value && !connectionsReady.value);

watch(
  () => connectionsQuery.data.value,
  data => {
    if (data) realtimeConnections.applyInitialSnapshot(data.item.items ?? null);
  },
  { immediate: true }
);

const editConfigQuery = useConnectEndpointConfigQuery(
  computed(() => editingEndpoint.value?.id ?? ''),
  editDialogVisible
);

const editConfigErrorText = computed(() =>
  editConfigQuery.error.value ? getErrorMessage(editConfigQuery.error.value, '账号配置读取失败') : ''
);

watch(editConfigQuery.data, data => {
  if (!data || !editingEndpoint.value) return;
  editingConfig.value = data;
  editFormModel.value = {
    ...buildDynamicFormInitialModel(data.schema ?? []),
    ...data.config,
  };
});

watch(editConfigQuery.error, error => {
  if (!error || !editDialogVisible.value) return;
  message.error('账号配置读取失败');
});

const retryEditConfig = () => {
  void editConfigQuery.refetch();
};

const { createMutation, updateMutation, enableMutation, deleteMutation } = useConnectMutations({
  message,
  onCreated: ({ endpoint, platform, officialQQMode: mode }) => {
    message.success('账号已添加');
    dialogVisible.value = false;
    if (platform === 'officialqq' && mode === 'qrcode') {
      qrDialogEndpointId.value = endpoint.id;
      qrDialogVisible.value = true;
    }
    resetWizard();
  },
  onUpdated: () => {
    message.success('账号配置已更新');
    editDialogVisible.value = false;
    editingEndpoint.value = null;
    editingConfig.value = null;
    editFormModel.value = {};
  },
  onEnabled: () => undefined,
  onDeleted: () => undefined,
});

const workflowOf = (endpointId: string) => realtimeConnections.workflows.value[endpointId] ?? null;

const openQRCode = (endpoint: EndPointInfo) => {
  qrDialogEndpointId.value = endpoint.id;
  qrDialogVisible.value = true;
};

const confirmDelete = (endpoint: EndPointInfo) => {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  dialog.warning({
    title: '删除账号',
    content: `确认删除账号「${getEndpointTargetLabel(endpoint)}」吗？删除账号不会影响人物卡和 logs 等数据。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => deleteMutation.mutate(endpoint.id),
  });
};

const confirmEnable = (endpoint: EndPointInfo, enable: boolean) => {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  dialog.warning({
    title: '修改账号状态',
    content: `确认${enable ? '启用' : '禁用'}账号「${getEndpointTargetLabel(endpoint)}」吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => enableMutation.mutate({ id: endpoint.id, enable }),
  });
};

const openEditDialog = (endpoint: EndPointInfo) => {
  editingEndpoint.value = endpoint;
  editingConfig.value = null;
  editFormModel.value = {};
  editDialogVisible.value = true;
};

const openCreateDialog = () => {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  resetWizard();
  dialogVisible.value = true;
};

const retrySignInfo = () => {
  void signInfo.retry();
};

const columns = computed(() =>
  createConnectTableColumns({
    isMobile,
    qrCodes: realtimeConnections.qrCodes,
    isTestMode,
    workflowOf,
    openQRCode,
    openEditDialog,
    confirmEnable,
    confirmDelete,
  })
);

const wizardCanNext = computed(() => {
  switch (wizardStep.value) {
    case 1:
      return !!wizardPlatform.value;
    case 2:
      return !!wizardMethod.value;
    case 3: {
      const protocol = wizardProtocol.value;
      return !!protocol && protocol.available && !protocol.deprecated;
    }
    case 4:
      return canSubmit.value;
    default:
      return false;
  }
});

const selectPlatform = (platform: PlatformTreeNode) => selectConnectPlatform(wizardState, platform);
const selectMethod = (method: MethodTreeNode) => selectConnectMethod(wizardState, method);
const selectProtocol = (protocol: ProtocolDefinition) =>
  selectConnectProtocol(wizardState, protocol, []);

const goNext = () => {
  advanceConnectWizard(wizardState, selectedSchema.value);
};

const goPrev = () => {
  if (wizardState.step > 1) wizardState.step -= 1;
};

const resetWizard = () => resetConnectWizard(wizardState);

const submit = () => {
  if (isTestMode.value) {
    message.warning('展示模式不支持该操作');
    return;
  }
  if (!canSubmit.value) return;
  const payload = buildConnectCreatePayload(wizardState);
  createMutation.mutate({
    platform: payload.platform,
    config: payload.config,
    officialQQMode: officialQQMode.value,
  });
};

const submitEdit = () => {
  if (isTestMode.value || !editingEndpoint.value || !canSubmitEdit.value) return;
  updateMutation.mutate({ id: editingEndpoint.value.id, config: editFormModel.value });
};
</script>

<style scoped>
.connect-page {
  max-width: 1180px;
  margin: 0 auto;
  text-align: left;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

h4 {
  margin: 0;
  color: var(--sd-text-primary);
  font-size: 1rem;
  font-weight: 700;
}

.account-cell {
  min-width: 180px;
}

.account-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  font-weight: 700;
  color: var(--sd-text-primary);
}

.account-subtitle {
  margin-top: 0.25rem;
  color: var(--sd-text-muted);
  font-size: 0.82rem;
}

:deep(.account-detail-value) {
  overflow-wrap: anywhere;
}

.account-dialog {
  width: min(720px, calc(100vw - 32px));
}

.wizard-dialog {
  max-width: 720px;
}

@media screen and (max-width: 639.9px) {
  .page-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .account-dialog {
    width: calc(100vw - 24px);
  }

  .wizard-dialog :deep(.n-step-content-header) {
    font-size: 0.78rem;
  }
}
</style>
