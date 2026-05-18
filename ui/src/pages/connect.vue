<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { useMutation, useQuery } from '@tanstack/vue-query';
import dayjs from 'dayjs';
import { useDialog, useMessage, type DataTableColumns } from 'naive-ui';
import {
  deleteSdApiV2ImconnectionById,
  getSdApiV2ImconnectionByIdConfig,
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
  type WorkflowResp,
} from '@/api';
import AsyncFieldSection from '@/components/shared/AsyncFieldSection.vue';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import {
  buildDynamicFormInitialModel,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import { hasAccessToken } from '@/features/auth/state';
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
const protocols = computed(() => protocolsQuery.data.value?.item.items ?? []);
const schemas = computed(() => schemasQuery.data.value?.item ?? {});
const connections = realtimeConnections.connections;
const selectedProtocol = computed(
  () => protocols.value.find(item => item.key === selectedProtocolKey.value) ?? null
);
const selectedSchema = computed<FormConfigItem[]>(() => {
  const protocol = selectedProtocol.value;
  if (!protocol) return [];
  return schemas.value[protocol.schemaKey] ?? [];
});
const editSchema = computed<FormConfigItem[]>(() => editingConfig.value?.schema ?? []);

const protocolOptions = computed(() =>
  protocols.value
    .filter(item => !item.deprecated)
    .map(item => ({
      label: item.name,
      value: item.key,
      disabled: !item.available,
    }))
);

const canSubmit = computed(() => {
  if (!selectedProtocol.value?.available) return false;
  return validateDynamicFormModel(selectedSchema.value, formModel.value).valid;
});
const canSubmitEdit = computed(() => validateDynamicFormModel(editSchema.value, editFormModel.value).valid);
const activeQRCode = computed(() => realtimeConnections.qrCodes.value[qrDialogEndpointId.value] ?? '');
const realtimeErrorText = computed(() =>
  realtimeConnections.lastError.value ? '实时连接异常，账号状态可能延迟。' : ''
);
const connectionsLoading = computed(() =>
  hasAccessToken.value && !realtimeConnections.ready.value,
);

watch(
  protocols,
  items => {
    if (!selectedProtocolKey.value) {
      selectedProtocolKey.value = items.find(item => !item.deprecated && item.available)?.key ?? '';
    }
  },
  { immediate: true }
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
      body: { body: editFormModel.value },
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
      body: { body: { enable } },
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

const stateTag = (state: number) => {
  switch (state) {
    case 1:
      return { type: 'success' as const, text: '已连接' };
    case 2:
      return { type: 'warning' as const, text: '连接中' };
    case 3:
      return { type: 'error' as const, text: '失败' };
    default:
      return { type: 'error' as const, text: '断开' };
  }
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

const protocolLabel = (endpoint: EndPointInfo) => {
  const adapter = adapterOf(endpoint);
  if (endpoint.protocolType === 'onebot' && adapter.builtinMode === 'lagrange') return 'QQ(内置客户端)';
  if (endpoint.protocolType === 'milky' && adapter.built_in_mode) return 'QQ(内置Milky)';
  if (endpoint.protocolType === 'milky') return 'QQ(Milky)';
  if (endpoint.protocolType === 'pureonebot' && adapter.reverseAddr) return 'QQ(onebot11反向WS)';
  if (endpoint.protocolType === 'pureonebot') return 'QQ(onebot11正向WS)';
  if (endpoint.protocolType === 'satori') return 'Satori';
  return endpoint.platform;
};

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
  dialogVisible.value = true;
};

const retrySignInfo = () => {
  void signInfoQuery.refetch();
};

const columns: DataTableColumns<EndPointInfo> = [
  {
    title: '账号',
    key: 'account',
    render: row => {
      const tag = stateTag(row.state);
      const loginTag = workflowTag(row);
      return (
        <div class='account-cell'>
          <div class='account-title'>
            <span>{row.nickname || row.userId || row.id}</span>
            <n-tag size='small' type={tag.type} bordered={false}>
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
    render: row => (
      <n-descriptions size='small' label-placement='left' column={2}>
        {detailRows(row).map(([label, value]) => (
          <n-descriptions-item key={label} label={label}>
            {value}
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
];

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

    <n-empty v-if="connections.length === 0 && realtimeConnections.ready.value" description="似乎还没有账号">
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
      size="small"
    />

    <n-modal
      v-model:show="dialogVisible"
      preset="dialog"
      title="帐号登录"
      class="account-dialog"
      :show-icon="false"
      :mask-closable="false"
    >
      <n-space vertical size="large">
        <n-alert
          v-if="selectedProtocol && !selectedProtocol.available"
          type="warning"
          :show-icon="false"
        >
          {{ selectedProtocol.disabledReason }}
        </n-alert>

        <n-form label-placement="left" :label-width="108">
          <n-form-item label="账号类型">
            <n-select
              v-model:value="selectedProtocolKey"
              :options="protocolOptions"
              :loading="protocolsQuery.isLoading.value"
            />
          </n-form-item>
        </n-form>

        <n-alert
          v-if="protocolsQuery.error.value"
          type="error"
          :show-icon="false"
        >
          账号类型读取失败，请关闭弹窗后重试。
        </n-alert>

        <n-alert
          v-if="selectedProtocol && !selectedSchema.length && schemasQuery.isFetching.value"
          type="info"
          :show-icon="false"
        >
          正在加载当前账号类型的配置项…
        </n-alert>

        <n-alert
          v-if="selectedProtocol && !selectedSchema.length && schemasQuery.error.value"
          type="error"
          :show-icon="false"
        >
          当前账号类型的配置项读取失败，请稍后重试。
        </n-alert>

        <DynamicForm
          v-model="formModel"
          :schema="selectedSchema"
          :disabled="createMutation.isPending.value"
        >
          <template #field="{ item, fieldKey, value, setValue }">
            <AsyncFieldSection
              v-if="selectedProtocolKey === 'lagrange' && fieldKey === 'signServerVersion'"
              :loading="signInfoState.mode === 'loading'"
              :message="signInfoState.message"
              :error="signInfoErrorMessage"
              @retry="retrySignInfo"
            >
              <n-select
                :value="value as string"
                :options="signVersionOptions"
                :disabled="!signInfoState.canSelectVersion"
                placeholder="请选择签名版本"
                @update:value="setValue"
              />
            </AsyncFieldSection>
            <AsyncFieldSection
              v-else-if="selectedProtocolKey === 'lagrange' && fieldKey === 'signServerName'"
              :loading="signInfoState.mode === 'loading'"
              :message="
                signInfoState.mode === 'manual-fallback' ? '' : signInfoState.message
              "
              :error="fieldKey === 'signServerName' ? signInfoErrorMessage : ''"
              @retry="retrySignInfo"
            >
              <n-select
                v-if="!signInfoState.showCustomServerInput"
                :value="value as string"
                :options="signServers"
                :disabled="!signInfoState.canSelectServer"
                placeholder="请选择签名服务"
                @update:value="setValue"
              />
              <n-input
                v-else
                :value="value as string"
                placeholder="请输入自定义签名地址"
                @update:value="setValue"
              />
            </AsyncFieldSection>
            <n-input
              v-else-if="item.input_type === 0"
              :value="value as string"
              :type="item.sensitive ? 'password' : 'text'"
              :placeholder="item.placeholder"
              show-password-on="mousedown"
              @update:value="setValue"
            />
          </template>
        </DynamicForm>
      </n-space>

      <template #action>
        <n-button @click="dialogVisible = false">
          取消
        </n-button>
        <n-button
          type="primary"
          :loading="createMutation.isPending.value"
          :disabled="!canSubmit"
          @click="submit"
        >
          下一步
        </n-button>
      </template>
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
  color: #111827;
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
  color: #111827;
}

.account-subtitle {
  margin-top: 0.25rem;
  color: #6b7280;
  font-size: 0.82rem;
}

.account-dialog {
  width: min(720px, calc(100vw - 32px));
}
</style>
