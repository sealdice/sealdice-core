<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useMutation, useQuery } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import { useDialog, useMessage, type DataTableColumns } from 'naive-ui';
import {
  deleteSdApiV2ImconnectionById,
  getSdApiV2ImconnectionByIdConfig,
  getSdApiV2ImconnectionOptions,
  getSdApiV2ImconnectionProtocolsOptions,
  getSdApiV2ImconnectionSchemasOptions,
  getSdApiV2ImconnectionSignInfoOptions,
  postSdApiV2Imconnection,
  putSdApiV2ImconnectionById,
  putSdApiV2ImconnectionByIdEnable,
  type EndPointInfo,
  type EditableConfigResp,
  type FormConfigItem,
  type ProtocolDefinition,
  type PlatformTreeNode,
  type MethodTreeNode,
  type WorkflowResp,
} from '@/api';
import ConnectCreateWizard from '@/components/connect/ConnectCreateWizard.vue';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import {
  buildDynamicFormInitialModel,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import { getEndpointProtocolLabel, getEndpointStateMeta } from '@/features/connect/endpointDisplay';
import { useRealtimeConnections } from '@/features/connect/realtime';
import { buildSignInfoState } from '@/features/connect/signInfoState';

type AdapterView = {
  connectUrl?: string;
  reverseAddr?: string;
  signServerVer?: string;
  signServerName?: string;
  builtinMode?: string;
  built_in_mode?: string;
  ws_gateway?: string;
  rest_gateway?: string;
  host?: string;
  port?: number;
  platform?: string;
};

const message = useMessage();
const dialog = useDialog();
const realtimeConnections = useRealtimeConnections();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

// 连接管理页的设计：
// - 协议定义和表单 schema 来自后端，用 DynamicForm 渲染；
// - 连接列表、登录工作流、二维码来自实时事件；
// - 创建/编辑/启停/删除仍走标准 HTTP mutation。
// 这样页面既能实时更新，又保持写操作有明确的成功/失败反馈。
const dialogVisible = ref(false);
const editDialogVisible = ref(false);
const qrDialogVisible = ref(false);
const qrDialogEndpointId = ref('');
const selectedProtocolKey = ref('');
const formModel = ref<DynamicFormModel>({});
const editingEndpoint = ref<EndPointInfo | null>(null);
const editingConfig = ref<EditableConfigResp | null>(null);
const editFormModel = ref<DynamicFormModel>({});

// Step wizard state
const wizardStep = ref(1);
const wizardPlatform = ref<PlatformTreeNode | null>(null);
const wizardMethod = ref<MethodTreeNode | null>(null);
const wizardProtocol = ref<ProtocolDefinition | null>(null);

const connectionsQuery = useQuery({
  ...getSdApiV2ImconnectionOptions(),
  enabled: hasAccessToken,
});
const protocolsQuery = useQuery({
  ...getSdApiV2ImconnectionProtocolsOptions(),
  enabled: hasAccessToken,
});
const schemasQuery = useQuery({
  ...getSdApiV2ImconnectionSchemasOptions(),
  enabled: hasAccessToken,
});
const signInfoQuery = useQuery({
  ...getSdApiV2ImconnectionSignInfoOptions(),
  enabled: computed(() => hasAccessToken.value && selectedProtocolKey.value === 'lagrange'),
});

// ProtocolDefinition 是后端对“可创建哪些连接”的声明。
// 前端只过滤 deprecated/available，不在这里硬编码某个平台是否可用。
const protocols = computed<PlatformTreeNode[]>(() => (protocolsQuery.data.value?.item.items ?? []) as PlatformTreeNode[]);
const schemas = computed(() => schemasQuery.data.value?.item ?? {});
const connections = realtimeConnections.connections;

const allProtocols = computed<ProtocolDefinition[]>(() => {
  const result: ProtocolDefinition[] = [];
  for (const platform of protocols.value) {
    for (const method of platform.methods ?? []) {
      for (const protocol of method.protocols ?? []) {
        result.push(protocol);
      }
    }
  }
  return result;
});

const selectedProtocol = computed(
  () => allProtocols.value.find(item => item.key === selectedProtocolKey.value) ?? null
);
const selectedSchema = computed<FormConfigItem[]>(() => {
  const protocol = selectedProtocol.value;
  if (!protocol) return [];
  return schemas.value[protocol.schemaKey] ?? [];
});
const editSchema = computed<FormConfigItem[]>(() => editingConfig.value?.schema ?? []);

const canSubmit = computed(() => {
  if (!selectedProtocol.value?.available) return false;
  return validateDynamicFormModel(selectedSchema.value, formModel.value).valid;
});
const canSubmitEdit = computed(() => validateDynamicFormModel(editSchema.value, editFormModel.value).valid);
const activeQRCode = computed(() => realtimeConnections.qrCodes.value[qrDialogEndpointId.value] ?? '');
const realtimeErrorText = computed(() =>
  realtimeConnections.lastError.value ? '实时连接异常，账号状态可能延迟。' : ''
);
const connectionsReady = computed(() =>
  realtimeConnections.ready.value || connectionsQuery.isSuccess.value || connectionsQuery.isError.value,
);
const connectionsErrorText = computed(() =>
  connectionsQuery.isError.value && !realtimeConnections.ready.value
    ? getErrorMessage(connectionsQuery.error.value, '账号列表读取失败')
    : ''
);
const connectionsLoading = computed(() =>
  hasAccessToken.value && !connectionsReady.value,
);

watch(
  () => connectionsQuery.data.value,
  data => {
    // 实时首帧可能早于页面订阅发出；REST 首屏快照只在 ready 前兜底，后续仍由实时事件增量更新。
    if (!data) return;
    realtimeConnections.applyInitialSnapshot(data.item.items ?? null);
  },
  { immediate: true },
);

watch(selectedSchema, schema => {
  formModel.value = buildDynamicFormInitialModel(schema);
});

watch(signInfoQuery.data, data => {
  // Lagrange 创建表单需要后端推荐的签名服务。这里把推荐值写入动态表单 model，
  // 用户仍可在表单里按需覆盖。
  if (selectedProtocolKey.value !== 'lagrange') return;
  const items = data?.item.items ?? [];
  const selectedVersion = items.find(item => item.selected && !item.ignored) ?? items.find(item => !item.ignored);
  const selectedServer =
    selectedVersion?.servers?.find(item => item.selected && !item.ignored) ??
    selectedVersion?.servers?.find(item => !item.ignored);
  formModel.value = {
    ...formModel.value,
    signServerVersion: selectedVersion?.version ?? '',
    signServerName: selectedServer?.name ?? '',
  };
});

const createMutation = useMutation({
  mutationFn: async () => {
    const { data } = await postSdApiV2Imconnection({
      body: {
        platform: selectedProtocolKey.value,
        config: formModel.value,
      },
      throwOnError: true,
    });
    return data;
  },
  onSuccess: () => {
    message.success('账号已添加');
    dialogVisible.value = false;
    resetWizard();
  },
  onError: () => {
    message.error('添加账号失败');
  },
});

const updateMutation = useMutation({
  mutationFn: async () => {
    if (!editingEndpoint.value) throw new Error('missing endpoint');
    const { data } = await putSdApiV2ImconnectionById({
      path: { id: editingEndpoint.value.id },
      body: editFormModel.value,
      throwOnError: true,
    });
    return data;
  },
  onSuccess: () => {
    message.success('账号配置已更新');
    editDialogVisible.value = false;
    editingEndpoint.value = null;
    editingConfig.value = null;
    editFormModel.value = {};
  },
  onError: () => {
    message.error('账号配置更新失败');
  },
});

const enableMutation = useMutation({
  mutationFn: async ({ endpoint, enable }: { endpoint: EndPointInfo; enable: boolean }) => {
    const { data } = await putSdApiV2ImconnectionByIdEnable({
      path: { id: endpoint.id },
      body: { enable },
      throwOnError: true,
    });
    return data;
  },
  onSuccess: () => {
    message.success('账号状态已更新');
  },
  onError: () => {
    message.error('账号状态更新失败');
  },
});

const deleteMutation = useMutation({
  mutationFn: async (endpoint: EndPointInfo) => {
    const { data } = await deleteSdApiV2ImconnectionById({
      path: { id: endpoint.id },
      throwOnError: true,
    });
    return data;
  },
  onSuccess: () => {
    message.success('账号已删除');
  },
  onError: () => {
    message.error('删除账号失败');
  },
});

const signVersions = computed(() =>
  (signInfoQuery.data.value?.item.items ?? [])
    .filter(item => !item.ignored)
    .map(item => ({
      label: item.selected ? `${item.version} 最新` : item.version,
      value: item.version,
    }))
    .concat({ label: '自定义', value: '自定义' })
);

const signServers = computed(() => {
  const version = formModel.value.signServerVersion;
  const info = (signInfoQuery.data.value?.item.items ?? []).find(item => item.version === version);
  return (info?.servers ?? [])
    .filter(item => !item.ignored)
    .map(item => ({
      label: item.latency > 0 ? `${item.name} (${item.latency}ms)` : item.name,
      value: item.name,
    }));
});

const signInfoState = computed(() =>
  buildSignInfoState({
    selectedProtocolKey: selectedProtocolKey.value,
    isLoading: signInfoQuery.isLoading.value,
    isFetching: signInfoQuery.isFetching.value,
    isError: signInfoQuery.isError.value,
    hasData: (signInfoQuery.data.value?.item.items?.length ?? 0) > 0,
    signServerVersion: String(formModel.value.signServerVersion ?? ''),
  }),
);

const signVersionOptions = computed(() => {
  if (signInfoState.value.mode === 'manual-fallback') {
    return [{ label: '自定义', value: '自定义' }];
  }
  return signVersions.value;
});

const signInfoErrorMessage = computed(() =>
  signInfoState.value.mode === 'manual-fallback' ? signInfoState.value.message : '',
);

watch(
  () => formModel.value.signServerVersion,
  version => {
    if (selectedProtocolKey.value !== 'lagrange' || version === '自定义') return;
    const info = (signInfoQuery.data.value?.item.items ?? []).find(item => item.version === version);
    const server = info?.servers?.find(item => item.selected && !item.ignored) ?? info?.servers?.find(item => !item.ignored);
    if (server && formModel.value.signServerName !== server.name) {
      formModel.value = {
        ...formModel.value,
        signServerName: server.name,
      };
    }
  }
);

const adapterOf = (endpoint: EndPointInfo): AdapterView => {
  if (endpoint.adapter && typeof endpoint.adapter === 'object') {
    return endpoint.adapter as AdapterView;
  }
  return {};
};

const workflowOf = (endpoint: EndPointInfo): WorkflowResp | null =>
  realtimeConnections.workflows.value[endpoint.id] ?? null;

const workflowTag = (endpoint: EndPointInfo) => {
  const workflow = workflowOf(endpoint);
  switch (workflow?.state) {
    case 'qrcode':
      return { type: 'warning' as const, text: '等待扫码' };
    case 'pending':
      return { type: 'info' as const, text: '登录中' };
    case 'failed':
      return { type: 'error' as const, text: '登录失败' };
    default:
      return null;
  }
};

const workflowText = (endpoint: EndPointInfo) => {
  const workflow = workflowOf(endpoint);
  switch (workflow?.state) {
    case 'qrcode':
      return '等待扫码';
    case 'pending':
      return '登录中';
    case 'success':
      return '登录成功';
    case 'failed':
      return workflow.failedReason ? `登录失败：${workflow.failedReason}` : '登录失败';
    default:
      return '';
  }
};

const openQRCode = (endpoint: EndPointInfo) => {
  qrDialogEndpointId.value = endpoint.id;
  qrDialogVisible.value = true;
};

const protocolLabel = (endpoint: EndPointInfo) =>
  getEndpointProtocolLabel({
    platform: endpoint.platform,
    protocolType: endpoint.protocolType,
    adapter: adapterOf(endpoint),
  });

const detailRows = (endpoint: EndPointInfo) => {
  const adapter = adapterOf(endpoint);
  return [
    ['账号', endpoint.userId],
    ['登录流程', workflowText(endpoint)],
    ['群组数量', String(endpoint.groupNum)],
    ['累计响应指令', String(endpoint.cmdExecutedNum)],
    [
      '上次执行指令',
      endpoint.cmdExecutedLastTime > 0
        ? dayjs.unix(endpoint.cmdExecutedLastTime).format('YYYY-MM-DD HH:mm:ss')
        : '尚无记录',
    ],
    ['连接地址', adapter.connectUrl || adapter.ws_gateway || ''],
    ['服务地址', adapter.reverseAddr ? `${adapter.reverseAddr}/ws` : ''],
    ['签名版本', adapter.signServerVer || ''],
    ['签名服务', adapter.signServerName || ''],
    ['协议端', adapter.built_in_mode || adapter.builtinMode || ''],
    ['主机', adapter.host ? `${adapter.host}${adapter.port ? `:${adapter.port}` : ''}` : ''],
  ].filter(([, value]) => value);
};

const confirmDelete = (endpoint: EndPointInfo) => {
  dialog.warning({
    title: '删除账号',
    content: '删除此项帐号，确定吗？删除账号不会影响人物卡和 logs 等数据。',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => deleteMutation.mutate(endpoint),
  });
};

const confirmEnable = (endpoint: EndPointInfo, enable: boolean) => {
  dialog.warning({
    title: '修改账号状态',
    content: '确认修改此账号的在线状态吗？',
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: () => enableMutation.mutate({ endpoint, enable }),
  });
};

const openEditDialog = async (endpoint: EndPointInfo) => {
  editingEndpoint.value = endpoint;
  editDialogVisible.value = true;
  editingConfig.value = null;
  editFormModel.value = {};
  try {
    const { data } = await getSdApiV2ImconnectionByIdConfig({
      path: { id: endpoint.id },
      throwOnError: true,
    });
    const item = data.item;
    editingConfig.value = item;
    editFormModel.value = {
      ...buildDynamicFormInitialModel(item.schema ?? []),
      ...item.config,
    };
  } catch {
    message.error('账号配置读取失败');
    editDialogVisible.value = false;
    editingEndpoint.value = null;
  }
};

const openCreateDialog = () => {
  resetWizard();
  dialogVisible.value = true;
};

const retrySignInfo = () => {
  void signInfoQuery.refetch();
};

const columns = computed<DataTableColumns<EndPointInfo>>(() => [
  {
    title: '账号',
    key: 'account',
    minWidth: 180,
    render: row => {
      const tag = getEndpointStateMeta(row.state);
      const loginTag = workflowTag(row);
      return (
        <div class='account-cell'>
          <div class='account-title'>
            <span>{row.nickname || row.userId || row.id}</span>
            <n-tag size='small' type={tag.tagType} bordered={false}>
              {tag.text}
            </n-tag>
            {!row.enable ? (
              <n-tag size='small' type='warning' bordered={false}>
                已禁用
              </n-tag>
            ) : null}
            {loginTag ? (
              <n-tag size='small' type={loginTag.type} bordered={false}>
                {loginTag.text}
              </n-tag>
            ) : null}
          </div>
          <div class='account-subtitle'>{protocolLabel(row)}</div>
        </div>
      );
    },
  },
  {
    title: '详情',
    key: 'detail',
    minWidth: 320,
    render: row => (
      <n-descriptions size='small' label-placement='left' column={isMobile.value ? 1 : 2}>
        {detailRows(row).map(([label, value]) => (
          <n-descriptions-item key={label} label={label}>
            <span class='account-detail-value'>{value}</span>
          </n-descriptions-item>
        ))}
      </n-descriptions>
    ),
  },
  {
    title: '操作',
    key: 'actions',
    width: 280,
    render: row => (
      <n-space justify='end'>
        {realtimeConnections.qrCodes.value[row.id] ? (
          <n-button size='small' tertiary onClick={() => openQRCode(row)}>
            二维码
          </n-button>
        ) : null}
        <n-button size='small' onClick={() => openEditDialog(row)}>
          修改
        </n-button>
        <n-button size='small' onClick={() => confirmEnable(row, !row.enable)}>
          {row.enable ? '禁用' : '启用'}
        </n-button>
        <n-button size='small' type='error' onClick={() => confirmDelete(row)}>
          删除
        </n-button>
      </n-space>
    ),
  },
]);

const wizardCanNext = computed(() => {
  switch (wizardStep.value) {
    case 1: return !!wizardPlatform.value;
    case 2: return !!wizardMethod.value;
    case 3: {
      const p = wizardProtocol.value;
      return !!p && p.available && !p.deprecated;
    }
    case 4: return canSubmit.value;
  }
  return false;
});

const goNext = () => {
  if (wizardStep.value === 3 && wizardProtocol.value) {
    selectedProtocolKey.value = wizardProtocol.value.key;
    formModel.value = buildDynamicFormInitialModel(selectedSchema.value);
  }
  if (wizardStep.value < 4) {
    wizardStep.value++;
  }
};

const goPrev = () => {
  if (wizardStep.value > 1) {
    wizardStep.value--;
  }
};

const resetWizard = () => {
  wizardStep.value = 1;
  wizardPlatform.value = null;
  wizardMethod.value = null;
  wizardProtocol.value = null;
  selectedProtocolKey.value = '';
  formModel.value = {};
};

const submit = () => {
  if (canSubmit.value) createMutation.mutate();
};

const submitEdit = () => {
  if (canSubmitEdit.value) updateMutation.mutate();
};
</script>

<template>
  <main class="connect-page">
    <div class="page-head">
      <h4>账号设置</h4>
      <n-button type="primary" @click="openCreateDialog">
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
        <n-button type="primary" @click="openCreateDialog">
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
        :selected-protocol-key="selectedProtocolKey"
        :selected-schema="selectedSchema"
        :sign-info-state="signInfoState"
        :sign-info-error-message="signInfoErrorMessage"
        :sign-version-options="signVersionOptions"
        :sign-servers="signServers"
        :is-mobile="isMobile"
        :can-submit="wizardCanNext"
        :submitting="createMutation.isPending.value"
        @cancel="dialogVisible = false"
        @previous="goPrev"
        @next="goNext"
        @submit="submit"
        @retry-sign-info="retrySignInfo"
      />
    </n-modal>

    <n-modal
      v-model:show="editDialogVisible"
      preset="dialog"
      title="修改账号配置"
      class="account-dialog"
      :show-icon="false"
      :mask-closable="false"
    >
      <n-spin :show="!editingConfig">
        <n-space vertical size="large">
          <n-alert v-if="editingConfig?.restartRequired" type="warning" :show-icon="false">
            保存后会重新连接此账号。Token、密码等敏感字段留空时保持原值不变。
          </n-alert>
          <DynamicForm
            v-model="editFormModel"
            :schema="editSchema"
            :disabled="updateMutation.isPending.value"
            :label-placement="isMobile ? 'top' : 'left'"
            :label-width="isMobile ? undefined : 108"
          />
        </n-space>
      </n-spin>

      <template #action>
        <n-button
          @click="editDialogVisible = false"
        >
          取消
        </n-button>
        <n-button
          type="primary"
          :loading="updateMutation.isPending.value"
          :disabled="!editingConfig || !canSubmitEdit"
          @click="submitEdit"
        >
          保存
        </n-button>
      </template>
    </n-modal>

    <n-modal
      v-model:show="qrDialogVisible"
      preset="dialog"
      title="登录二维码"
      class="qrcode-dialog"
      :show-icon="false"
    >
      <n-space vertical align="center" size="large">
        <n-image
          v-if="activeQRCode"
          :src="activeQRCode"
          width="280"
          preview-disabled
        />
        <n-empty v-else description="当前没有可用二维码" />
        <n-button size="small" secondary @click="realtimeConnections.reconnect">
          刷新连接
        </n-button>
      </n-space>
    </n-modal>
  </main>
</template>

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
