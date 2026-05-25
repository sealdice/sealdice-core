<template>
  <n-space vertical size="large">
    <n-steps :current="wizardStep" size="small">
      <n-step title="选择平台" />
      <n-step title="选择方式" />
      <n-step title="选择协议" />
      <n-step title="填写信息" />
    </n-steps>

    <div v-if="wizardStep === 1" class="wizard-step-panel">
      <div class="split-layout">
        <div class="split-left">
          <div
            v-for="platform in protocols"
            :key="platform.id"
            :class="['split-item', { 'split-item--selected': wizardPlatform?.id === platform.id }]"
            @click="wizardPlatform = platform"
          >
            <span class="split-item-name">{{ platform.name }}</span>
          </div>
        </div>
        <div class="split-right">
          <n-empty v-if="!wizardPlatform" description="请在左侧选择一个平台" />
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
          <div
            v-for="method in wizardPlatform?.methods"
            :key="method.id"
            :class="['split-item', { 'split-item--selected': wizardMethod?.id === method.id }]"
            @click="wizardMethod = method"
          >
            <span class="split-item-name">{{ method.name }}</span>
          </div>
        </div>
        <div class="split-right">
          <n-empty v-if="!wizardMethod" description="请在左侧选择一种方式" />
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
          <div
            v-for="protocol in wizardMethod?.protocols"
            :key="protocol.key"
            :class="[
              'split-item',
              { 'split-item--selected': wizardProtocol?.key === protocol.key },
              { 'split-item--disabled': !protocol.available },
            ]"
            @click="protocol.available ? (wizardProtocol = protocol) : null"
          >
            <span class="split-item-name">{{ protocol.name }}</span>
            <n-tag v-if="protocol.deprecated" type="warning" size="small">已废弃</n-tag>
            <n-tag v-else-if="!protocol.available" type="error" size="small">不可用</n-tag>
          </div>
        </div>
        <div class="split-right">
          <n-empty v-if="!wizardProtocol" description="请在左侧选择一个协议" />
          <template v-else>
            <h3 class="split-detail-title">
              {{ wizardProtocol.name }}
              <n-tag v-if="wizardProtocol.deprecated" type="warning" size="small">已废弃</n-tag>
            </h3>
            <p class="split-detail-desc">{{ wizardProtocol.description }}</p>
            <n-alert
              v-if="!wizardProtocol.available && wizardProtocol.disabledReason"
              type="warning"
              :show-icon="false"
              class="mt-2"
            >
              {{ wizardProtocol.disabledReason }}
            </n-alert>
          </template>
        </div>
      </div>
    </div>

    <div v-if="wizardStep === 4" class="wizard-step-panel">
      <n-alert v-if="selectedProtocol && !selectedProtocol.available" type="warning" :show-icon="false">
        {{ selectedProtocol.disabledReason }}
      </n-alert>

      <n-alert v-if="schemasError" type="error" :show-icon="false">
        配置项读取失败，请稍后重试。
      </n-alert>

      <DynamicForm
        v-model="formModel"
        :schema="selectedSchema"
        :disabled="submitting"
        :label-placement="isMobile ? 'top' : 'left'"
        :label-width="isMobile ? undefined : 108"
      >
        <template #field="{ item, fieldKey, value, setValue }">
          <AsyncFieldSection
            v-if="selectedProtocolKey === 'lagrange' && fieldKey === 'signServerVersion'"
            :loading="signInfoState.mode === 'loading'"
            :message="signInfoState.message"
            :error="signInfoErrorMessage"
            @retry="emit('retrySignInfo')"
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
            :message="signInfoState.mode === 'manual-fallback' ? '' : signInfoState.message"
            :error="fieldKey === 'signServerName' ? signInfoErrorMessage : ''"
            @retry="emit('retrySignInfo')"
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
    </div>
  </n-space>

  <div class="wizard-actions">
    <n-button @click="emit('cancel')">
      取消
    </n-button>
    <n-button v-if="wizardStep > 1" @click="emit('previous')">
      上一步
    </n-button>
    <n-button
      v-if="wizardStep < 4"
      type="primary"
      :disabled="!canSubmit"
      @click="emit('next')"
    >
      下一步
    </n-button>
    <n-button
      v-if="wizardStep === 4"
      type="primary"
      :loading="submitting"
      :disabled="!canSubmit"
      @click="emit('submit')"
    >
      添加
    </n-button>
  </div>
</template>

<script setup lang="ts">
import type { SelectOption } from 'naive-ui';
import type { FormConfigItem, MethodTreeNode, PlatformTreeNode, ProtocolDefinition } from '@/api';
import AsyncFieldSection from '@/components/shared/AsyncFieldSection.vue';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import type { DynamicFormModel } from '@/components/shared/dynamicFormModel';
import type { SignInfoState } from '@/features/connect/signInfoState';

defineProps<{
  protocols: PlatformTreeNode[];
  schemasError: boolean;
  selectedProtocol: ProtocolDefinition | null;
  selectedProtocolKey: string;
  selectedSchema: FormConfigItem[];
  signInfoState: SignInfoState;
  signInfoErrorMessage: string;
  signVersionOptions: SelectOption[];
  signServers: SelectOption[];
  isMobile: boolean;
  canSubmit: boolean;
  submitting: boolean;
}>();

const formModel = defineModel<DynamicFormModel>('formModel', { required: true });
const wizardStep = defineModel<number>('wizardStep', { required: true });
const wizardPlatform = defineModel<PlatformTreeNode | null>('wizardPlatform', { required: true });
const wizardMethod = defineModel<MethodTreeNode | null>('wizardMethod', { required: true });
const wizardProtocol = defineModel<ProtocolDefinition | null>('wizardProtocol', { required: true });

const emit = defineEmits<{
  cancel: [];
  next: [];
  previous: [];
  submit: [];
  retrySignInfo: [];
}>();
</script>

<style scoped>
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
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
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
  margin-top: 1.5rem;
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
</style>
