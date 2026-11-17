<template>
  <n-modal
    v-model:show="show"
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
    :closable="!createMutation.isPending.value"
    :close-on-esc="!createMutation.isPending.value"
    @after-leave="!show && resetConnectWizard(wizardState)"
  >
    <TipBox v-if="createMutation.isError.value" type="error">
      {{ getErrorMessage(createMutation.error.value, '添加账号失败') }}。请检查配置后重试。
    </TipBox>
    <n-space vertical size="large">
      <n-steps class="wizard-steps" :current="wizardStep" size="small">
        <n-step v-for="step in wizardSteps" :key="step" :title="step" />
      </n-steps>

      <div class="wizard-progress" role="status" aria-live="polite">
        <div class="wizard-progress__current">
          <span>第 {{ wizardStep }} / {{ wizardSteps.length }} 步</span>
          <strong>{{ currentWizardStepTitle }}</strong>
        </div>
        <span v-if="remainingWizardStepTitles.length" class="wizard-progress__remaining">
          接下来：{{ remainingWizardStepTitles.join('、') }}
        </span>
        <span v-else class="wizard-progress__remaining">完成当前信息后即可添加账号</span>
      </div>

      <div v-if="wizardStep === 1" class="wizard-step-panel">
        <div class="split-layout">
          <div class="split-left">
            <button
              v-for="platform in protocols"
              :key="platform.id"
              type="button"
              :class="[
                'split-item',
                { 'split-item--selected': wizardPlatform?.id === platform.id },
              ]"
              :aria-pressed="wizardPlatform?.id === platform.id"
              @click="selectConnectPlatform(wizardState, platform)"
            >
              <span class="split-item-name">{{ platform.name }}</span>
            </button>
          </div>
          <div class="split-right">
            <n-empty v-if="!wizardPlatform" description="请选择一个平台" />
            <template v-else>
              <h3 class="split-detail-title">{{ wizardPlatform.name }}</h3>
              <p class="split-detail-desc">{{ wizardPlatform.description }}</p>
            </template>
          </div>
        </div>
      </div>

      <div v-if="wizardStep === 2" class="wizard-step-panel">
        <div class="split-layout">
          <div class="split-left">
            <button
              v-for="method in wizardPlatform?.methods"
              :key="method.id"
              type="button"
              :class="['split-item', { 'split-item--selected': wizardMethod?.id === method.id }]"
              :aria-pressed="wizardMethod?.id === method.id"
              @click="selectConnectMethod(wizardState, method)"
            >
              <span class="split-item-name">{{ method.name }}</span>
            </button>
          </div>
          <div class="split-right">
            <n-empty v-if="!wizardMethod" description="请选择一种方式" />
            <template v-else>
              <h3 class="split-detail-title">{{ wizardMethod.name }}</h3>
              <p class="split-detail-desc">{{ wizardMethod.description }}</p>
            </template>
          </div>
        </div>
      </div>

      <div v-if="wizardStep === 3" class="wizard-step-panel">
        <div class="split-layout">
          <div class="split-left">
            <button
              v-for="protocol in wizardMethod?.protocols"
              :key="protocol.key"
              type="button"
              :class="[
                'split-item',
                { 'split-item--selected': wizardProtocol?.key === protocol.key },
                { 'split-item--disabled': !protocol.available },
              ]"
              :aria-pressed="wizardProtocol?.key === protocol.key"
              :disabled="!protocol.available"
              @click="selectConnectProtocol(wizardState, protocol)"
            >
              <span class="split-item-name">{{ protocol.name }}</span>
              <n-tag v-if="protocol.deprecated" type="warning" size="small">已废弃</n-tag>
              <n-tag v-else-if="!protocol.available" type="error" size="small">不可用</n-tag>
            </button>
          </div>
          <div class="split-right">
            <n-empty v-if="!wizardProtocol" description="请选择一个协议" />
            <template v-else>
              <h3 class="split-detail-title">
                {{ wizardProtocol.name }}
                <n-tag v-if="wizardProtocol.deprecated" type="warning" size="small">已废弃</n-tag>
              </h3>
              <p class="split-detail-desc">{{ wizardProtocol.description }}</p>
              <TipBox
                v-if="!wizardProtocol.available && wizardProtocol.disabledReason"
                type="warning"
              >
                {{ wizardProtocol.disabledReason }}
              </TipBox>
            </template>
          </div>
        </div>
      </div>

      <div v-if="wizardStep === 4" class="wizard-step-panel">
        <div class="sd-section-flow">
          <TipBox v-if="wizardProtocol?.key === 'officialqq'" type="info">
            <n-space vertical size="small">
              <n-radio-group v-model:value="officialQQMode">
                <n-radio-button value="manual">手动填写机器人 ID</n-radio-button>
                <n-radio-button value="qrcode">扫码登录</n-radio-button>
              </n-radio-group>
              <n-text depth="3">
                {{
                  officialQQMode === 'qrcode'
                    ? '扫码模式会在创建后生成二维码，自动获取机器人 ID 和密钥。'
                    : '手动模式需要填写机器人 ID 和密钥，添加前会先测试凭据。'
                }}
              </n-text>
            </n-space>
          </TipBox>

          <TipBox v-if="wizardProtocol && !wizardProtocol.available" type="warning">
            {{ wizardProtocol.disabledReason }}
          </TipBox>

          <TipBox v-if="schemasError" type="error"> 配置项读取失败，请稍后重试。 </TipBox>

          <DynamicForm
            :model-value="formModel"
            :schema="visibleSchema"
            :disabled="createMutation.isPending.value || isTestMode"
            :label-placement="isMobile ? 'top' : 'left'"
            :label-width="isMobile ? undefined : 108"
            @update:model-value="formModel = $event"
          >
            <template #field="{ fieldKey, value, setValue }">
              <LagrangeSignInfoField
                v-if="
                  wizardProtocol?.key === 'lagrange' &&
                  (fieldKey === 'signServerVersion' || fieldKey === 'signServerName')
                "
                :field-key="fieldKey"
                :value="value"
                :sign-info-state="signInfoState"
                :sign-info-error-message="signInfoErrorMessage"
                :sign-version-options="signVersionOptions"
                :sign-servers="signServers"
                :disabled="createMutation.isPending.value || isTestMode"
                @retry="signQuery.refetch()"
                @update:value="setValue"
              />
            </template>
          </DynamicForm>
        </div>
      </div>
    </n-space>

    <div class="wizard-actions">
      <n-button :disabled="createMutation.isPending.value" @click="show = false"> 取消 </n-button>
      <n-button
        :disabled="createMutation.isPending.value"
        v-if="wizardStep > 1"
        @click="wizardStep--"
      >
        上一步
      </n-button>
      <n-button
        v-if="wizardStep < 4"
        :type="canSubmit && !isTestMode ? 'primary' : 'default'"
        :disabled="!canSubmit || isTestMode"
        @click="advanceConnectWizard(wizardState, selectedSchema)"
      >
        下一步
      </n-button>
      <n-button
        v-if="wizardStep === 4"
        :type="canSubmit && !isTestMode ? 'primary' : 'default'"
        :loading="createMutation.isPending.value"
        :disabled="!canSubmit || isTestMode"
        @click="submit()"
      >
        添加
      </n-button>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, reactive, toRefs, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { storeToRefs } from 'pinia';
import {
  getSdApiV2ImconnectionProtocolsOptions,
  getSdApiV2ImconnectionSchemasOptions,
  getSdApiV2ImconnectionSignInfoOptions,
  getSdApiV2ImconnectionQueryKey,
  postSdApiV2Imconnection,
  type EndPointInfo,
} from '@/api';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import { fieldKeyOf } from '@/components/shared/dynamicFormModel';
import LagrangeSignInfoField from './protocol/LagrangeSignInfoField.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { useAuthStore } from '@/features/auth/store';
import { getErrorMessage } from '@/features/auth/error';
import { useTestMode } from '@/features/testMode/state';
import { notifyError } from '@/features/feedback/notifyError';
import {
  advanceConnectWizard,
  createConnectWizardState,
  resetConnectWizard,
  selectConnectPlatform,
  selectConnectMethod,
  selectConnectProtocol,
  validateConnectWizardForm,
} from '@/features/connect/createWizard';
import { buildOfficialQQCreateConfig } from '@/features/connect/officialQQ';
import { buildSignInfoState } from '@/features/connect/signInfoState';

const show = defineModel<boolean>('show', { required: true });
const emit = defineEmits<{ created: [endpoint: EndPointInfo, showQRCode: boolean] }>();
const message = useMessage();
const queryClient = useQueryClient();
const { hasAccessToken } = storeToRefs(useAuthStore());
const { isTestMode } = useTestMode();
const isMobile = useBreakpoints(breakpointsTailwind).smaller('md');
const wizardState = reactive(createConnectWizardState());
const {
  step: wizardStep,
  platform: wizardPlatform,
  method: wizardMethod,
  protocol: wizardProtocol,
  formModel,
  officialQQMode,
} = toRefs(wizardState);
const enabled = computed(() => show.value && hasAccessToken.value);
const protocolsQuery = useQuery({ ...getSdApiV2ImconnectionProtocolsOptions(), enabled });
const schemasQuery = useQuery({ ...getSdApiV2ImconnectionSchemasOptions(), enabled });
const protocols = computed(() => protocolsQuery.data.value?.item.items ?? []);
const schemasError = computed(() => schemasQuery.isError.value);
const selectedSchema = computed(
  () => schemasQuery.data.value?.item[wizardProtocol.value?.schemaKey ?? ''] ?? []
);
const visibleSchema = computed(() => {
  if (wizardProtocol.value?.key !== 'officialqq') return selectedSchema.value;
  return selectedSchema.value.filter(item => {
    const key = fieldKeyOf(item);
    if (!formModel.value.useWebhook && (key === 'webhookPath' || key === 'webhookPort'))
      return false;
    return officialQQMode.value !== 'qrcode' || (key !== 'appID' && key !== 'appSecret');
  });
});
const signEnabled = computed(
  () => enabled.value && wizardProtocol.value?.key === 'lagrange' && wizardStep.value === 4
);
const signQuery = useQuery({ ...getSdApiV2ImconnectionSignInfoOptions(), enabled: signEnabled });
const signItems = computed(
  () => signQuery.data.value?.item.items?.filter(item => !item.ignored) ?? []
);
const signInfoState = computed(() =>
  buildSignInfoState({
    enabled: signEnabled.value,
    isLoading: signQuery.isLoading.value,
    isFetching: signQuery.isFetching.value,
    isError: signQuery.isError.value,
    hasData: signItems.value.length > 0,
    signServerVersion: String(formModel.value.signServerVersion ?? ''),
  })
);
const signInfoErrorMessage = computed(() =>
  signInfoState.value.mode === 'manual-fallback' ? signInfoState.value.message : ''
);
const signVersionOptions = computed(() => [
  ...(signInfoState.value.mode === 'manual-fallback'
    ? []
    : signItems.value.map(item => ({
        label: item.selected ? `${item.version} 最新` : item.version,
        value: item.version,
      }))),
  { label: '自定义', value: '自定义' },
]);
const signServers = computed(() =>
  (signItems.value.find(item => item.version === formModel.value.signServerVersion)?.servers ?? [])
    .filter(item => !item.ignored)
    .map(item => ({
      label: item.latency > 0 ? `${item.name} (${item.latency}ms)` : item.name,
      value: item.name,
    }))
);
watch(
  [signItems, () => formModel.value.signServerVersion, signEnabled],
  ([items, version, active], [, previousVersion]) => {
    if (!active || !items.length || version === '自定义') return;
    const selected =
      items.find(item => item.version === version) ??
      (!version ? (items.find(item => item.selected) ?? items[0]) : undefined);
    if (!selected) return;
    const servers = selected.servers?.filter(item => !item.ignored) ?? [];
    const server = servers.find(item => item.selected) ?? servers[0];
    const versionChanged = !!previousVersion && previousVersion !== version;
    // 首次进入与重开向导时补默认值；后台刷新不能重置用户选择或自定义地址。
    if (!version) formModel.value = { ...formModel.value, signServerVersion: selected.version };
    if (server && (!formModel.value.signServerName || versionChanged)) {
      formModel.value = { ...formModel.value, signServerName: server.name };
    }
  }
);
const wizardSteps = ['选择平台', '选择方式', '选择协议', '填写信息'];
const currentWizardStepTitle = computed(() => wizardSteps[wizardStep.value - 1] ?? wizardSteps[0]);
const remainingWizardStepTitles = computed(() => wizardSteps.slice(wizardStep.value));
const canSubmit = computed(() => {
  switch (wizardStep.value) {
    case 1:
      return !!wizardPlatform.value;
    case 2:
      return !!wizardMethod.value;
    case 3:
      return !!wizardProtocol.value?.available && !wizardProtocol.value.deprecated;
    case 4:
      return (
        schemasQuery.isSuccess.value &&
        validateConnectWizardForm(wizardState, visibleSchema.value).valid
      );
    default:
      return false;
  }
});
const createMutation = useMutation({
  mutationFn: async () => {
    const platform = wizardProtocol.value!.key;
    const mode = officialQQMode.value;
    const config =
      platform === 'officialqq'
        ? buildOfficialQQCreateConfig(formModel.value, mode)
        : { ...formModel.value };
    // 创建接口负责探测凭据与重复账号校验，避免先 test 再 create 探测两次。
    const { data } = await postSdApiV2Imconnection({
      body: { platform, config },
      throwOnError: true,
    });
    return { endpoint: data.item, showQRCode: platform === 'officialqq' && mode === 'qrcode' };
  },
  onSuccess: async result => {
    message.success('账号已添加');
    show.value = false;
    emit('created', result.endpoint, result.showQRCode);
    await queryClient.invalidateQueries({ queryKey: getSdApiV2ImconnectionQueryKey() });
  },
  onError: error => notifyError(error, message, '添加账号失败'),
});
watch(show, visible => {
  if (visible) {
    resetConnectWizard(wizardState);
    createMutation.reset();
  }
});
function submit() {
  if (
    isTestMode.value ||
    createMutation.isPending.value ||
    wizardStep.value !== 4 ||
    !canSubmit.value
  )
    return;
  createMutation.mutate();
}
</script>

<style scoped>
.wizard-progress {
  display: none;
}

.wizard-step-panel {
  min-height: 240px;
}

.split-layout {
  display: flex;
  gap: 16px;
  height: 280px;
}

.split-left {
  flex: 0 0 180px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  padding-right: 8px;
  border-right: 1px solid var(--sd-border-soft);
}

.split-item {
  width: 100%;
  border: 0;
  font: inherit;
  color: inherit;
  text-align: left;
  background: transparent;
  padding: 10px 12px;
  border-radius: var(--sd-radius-sm);
  cursor: pointer;
  transition:
    background-color var(--sd-transition-fast),
    color var(--sd-transition-fast);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.split-item:hover {
  background-color: var(--sd-bg-hover);
}

.split-item--selected {
  background-color: var(--sd-bg-selected);
  color: var(--sd-primary);
  font-weight: 600;
}

.split-item--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.split-item--disabled:hover {
  background-color: transparent;
}

.split-item:focus-visible {
  outline: 2px solid var(--sd-primary);
  outline-offset: 2px;
}

.split-item-name {
  font-size: 0.9rem;
}

.split-right {
  flex: 1;
  padding: 8px 4px;
  overflow-y: auto;
}

.split-detail-title {
  margin: 0 0 12px 0;
  font-size: 1.1rem;
  font-weight: 600;
  color: var(--sd-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.split-detail-desc {
  margin: 0;
  font-size: 0.9rem;
  color: var(--sd-text-secondary);
  line-height: 1.6;
}

.mt-2 {
  margin-top: 8px;
}

.wizard-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid var(--sd-border-soft);
}

@media screen and (max-width: 639.9px) {
  .wizard-step-panel {
    min-height: 0;
  }

  .split-layout {
    height: auto;
    min-height: 18rem;
    flex-direction: column;
  }

  .split-left {
    flex: 0 0 auto;
    max-height: 11rem;
    border-right: 0;
    border-bottom: 1px solid var(--sd-border-soft);
    padding-right: 0;
    padding-bottom: 8px;
  }

  .split-right {
    min-height: 8rem;
  }

  .wizard-actions {
    flex-wrap: wrap;
  }
}
@media (max-width: 500px) {
  .wizard-steps {
    display: none;
  }

  .wizard-progress {
    display: grid;
    gap: 0.35rem;
    border-left: 3px solid var(--sd-primary);
    background: var(--sd-bg-elevated-soft);
    padding: 0.65rem 0.75rem;
  }

  .wizard-progress__current {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.75rem;
  }

  .wizard-progress__remaining {
    color: var(--sd-text-secondary);
    font-size: 0.82rem;
    line-height: 1.5;
  }
}
</style>
