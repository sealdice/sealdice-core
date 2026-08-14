<template>
  <main class="connect-page sd-page-flow">
    <PageHeader title="账号设置">
      <n-button type="primary" :disabled="isTestMode" @click="openCreateDialog">
        <template #icon
          ><n-icon><i-tabler-plus /></n-icon
        ></template>
        添加账号
      </n-button>
    </PageHeader>

    <TipBox v-if="connectionsErrorText" type="error">
      <div class="connect-alert-content">
        <span>{{ connectionsErrorText }}</span>
        <n-button size="small" secondary @click="retryConnections">
          <template #icon
            ><n-icon><i-tabler-refresh /></n-icon
          ></template>
          重试
        </n-button>
      </div>
    </TipBox>

    <ListEmptyState v-if="connections.length === 0 && connectionsReady" description="暂无接入账号">
      <n-button type="primary" :disabled="isTestMode" @click="openCreateDialog">
        添加账号
      </n-button>
    </ListEmptyState>

    <ConnectAccountGrid
      v-else
      :connections="connections"
      :workflows="realtimeConnections.workflows.value"
      :qr-codes="realtimeConnections.qrCodes.value"
      :loading="connectionsLoading"
      :is-test-mode="isTestMode"
      :pending-action="pendingEndpointAction"
      :operation-errors="endpointOperationErrors"
      @show-q-r-code="openQRCode"
      @edit="openEditDialog"
      @toggle-enable="confirmEnable"
      @delete="confirmDelete"
    />

    <n-modal
      v-model:show="dialogVisible"
      preset="dialog"
      title="添加账号"
      class="account-dialog wizard-dialog"
      style="
        width: min(860px, calc(100vw - 24px));
        max-width: 860px;
        border-radius: var(--sd-radius-md);
      "
      :show-icon="false"
      :mask-closable="false"
      @after-leave="resetWizard"
    >
      <div class="connect-dialog-content">
        <TipBox v-if="createSubmitError" type="error">
          {{ createSubmitError }}。请检查配置后重试。
        </TipBox>
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
      </div>
    </n-modal>

    <ConnectEditDialog
      :visible="editDialogVisible"
      :config="editingConfig"
      :form-model="editFormModel"
      :schema="editSchema"
      :loading="editConfigQuery.isFetching.value"
      :error-message="editConfigErrorText"
      :submit-error="editSubmitError"
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
  fieldKeyOf,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import ConnectAccountGrid from '@/components/connect/ConnectAccountGrid.vue';
import { updateEndpointOperationErrors } from '@/features/connect/endpointActionState';
import { getEndpointTargetLabel } from '@/features/connect/endpointDisplay';
import PageHeader from '@/components/shared/PageHeader.vue';
import ListEmptyState from '@/components/shared/ListEmptyState.vue';
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
import TipBox from '@/components/shared/TipBox.vue';

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
const pendingEndpointAction = ref<{ endpointId: string; action: 'enable' | 'delete' } | null>(null);
const endpointOperationErrors = ref<Record<string, string>>({});

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
const editSchema = computed<FormConfigItem[]>(() => {
  const schema = editingConfig.value?.schema ?? [];
  if (editingEndpoint.value?.protocolType !== 'official' || editFormModel.value.useWebhook) {
    return schema;
  }

  return schema.filter(item => {
    const fieldKey = fieldKeyOf(item);
    return fieldKey !== 'webhookPath' && fieldKey !== 'webhookPort';
  });
});

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
const activeQRWorkflow = computed(
  () => realtimeConnections.workflows.value[qrDialogEndpointId.value] ?? null
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
  [qrDialogVisible, activeQRWorkflow],
  ([visible, workflow]) => {
    if (!visible || workflow?.state !== 'success') return;
    qrDialogVisible.value = false;
    qrDialogEndpointId.value = '';
  },
  { immediate: true }
);

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
  editConfigQuery.error.value
    ? getErrorMessage(editConfigQuery.error.value, '账号配置读取失败')
    : ''
);

const applyEditConfig = (data: EditableConfigResp | undefined) => {
  if (!data || !editingEndpoint.value) return;
  editingConfig.value = data;
  editFormModel.value = {
    ...buildDynamicFormInitialModel(data.schema ?? []),
    ...data.config,
  };
};

watch([editConfigQuery.data, editDialogVisible], ([data, visible]) => {
  if (!visible) return;
  applyEditConfig(data);
});

watch(editConfigQuery.error, error => {
  if (!error || !editDialogVisible.value) return;
  message.error('账号配置读取失败');
});

const retryEditConfig = () => {
  void editConfigQuery.refetch();
};

const retryConnections = () => {
  void connectionsQuery.refetch();
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

const createSubmitError = computed(() =>
  createMutation.isError.value ? getErrorMessage(createMutation.error.value, '添加账号失败') : ''
);
const editSubmitError = computed(() =>
  updateMutation.isError.value
    ? getErrorMessage(updateMutation.error.value, '账号配置更新失败')
    : ''
);
const runEndpointOperation = async (
  endpoint: EndPointInfo,
  action: 'enable' | 'delete',
  operation: () => Promise<unknown>,
  fallbackError: string
) => {
  if (pendingEndpointAction.value) {
    message.warning('正在处理另一个账号操作，请稍候');
    return;
  }

  pendingEndpointAction.value = { endpointId: endpoint.id, action };
  endpointOperationErrors.value = updateEndpointOperationErrors(
    endpointOperationErrors.value,
    endpoint.id
  );
  try {
    await operation();
  } catch (error) {
    endpointOperationErrors.value = updateEndpointOperationErrors(
      endpointOperationErrors.value,
      endpoint.id,
      getErrorMessage(error, fallbackError)
    );
  } finally {
    pendingEndpointAction.value = null;
  }
};

const openQRCode = (endpoint: EndPointInfo) => {
  qrDialogEndpointId.value = endpoint.id;
  qrDialogVisible.value = true;
};

const confirmDelete = (endpoint: EndPointInfo) => {
  if (isTestMode.value || pendingEndpointAction.value) {
    if (pendingEndpointAction.value) message.warning('正在处理另一个账号操作，请稍候');
    if (isTestMode.value) message.warning('展示模式不支持该操作');
    return;
  }
  dialog.warning({
    title: '删除账号',
    content: `确认删除账号「${getEndpointTargetLabel(endpoint)}」吗？删除账号不会影响人物卡和 logs 等数据。`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () =>
      runEndpointOperation(
        endpoint,
        'delete',
        () => deleteMutation.mutateAsync(endpoint.id),
        '账号删除失败'
      ),
  });
};

const confirmEnable = (endpoint: EndPointInfo, enable: boolean) => {
  if (isTestMode.value || pendingEndpointAction.value) {
    if (pendingEndpointAction.value) message.warning('正在处理另一个账号操作，请稍候');
    if (isTestMode.value) message.warning('展示模式不支持该操作');
    return;
  }
  dialog.warning({
    title: '修改账号状态',
    content: `确认${enable ? '启用' : '禁用'}账号「${getEndpointTargetLabel(endpoint)}」吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () =>
      runEndpointOperation(
        endpoint,
        'enable',
        () => enableMutation.mutateAsync({ id: endpoint.id, enable }),
        '账号状态更新失败'
      ),
  });
};

const openEditDialog = (endpoint: EndPointInfo) => {
  updateMutation.reset();
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
  createMutation.reset();
  resetWizard();
  dialogVisible.value = true;
};

const retrySignInfo = () => {
  void signInfo.retry();
};

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
  text-align: left;
}

.connect-alert-content {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--sd-space-sm);
}

.connect-dialog-content {
  display: flex;
  flex-direction: column;
  gap: var(--sd-space-md);
}

@media screen and (max-width: 639.9px) {
  .account-dialog {
    width: calc(100vw - 24px);
  }

  :global(.wizard-dialog .n-step-content-header) {
    font-size: 0.78rem;
  }
}
</style>
