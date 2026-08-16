<template>
  <n-modal
    :show="visible"
    preset="dialog"
    title="修改账号配置"
    class="account-dialog"
    style="
      width: min(720px, calc(100vw - 24px));
      max-width: 720px;
      border-radius: var(--sd-radius-md);
    "
    :show-icon="false"
    :mask-closable="false"
    @update:show="emit('update:visible', $event)"
  >
    <TipBox v-if="errorMessage" type="error">
      {{ errorMessage }}
    </TipBox>
    <TipBox v-if="submitError" type="error" class="connect-edit-dialog__submit-error">
      {{ submitError }}。请检查配置后重试。
    </TipBox>
    <n-spin :show="loading && !config">
      <n-space vertical size="large">
        <TipBox v-if="config?.restartRequired" type="warning">
          保存后会重新连接此账号。Token、密码等敏感字段留空时保持原值不变。
        </TipBox>
        <DynamicForm
          :model-value="formModel"
          :schema="schema"
          :disabled="saving || disabled"
          :label-placement="isMobile ? 'top' : 'left'"
          :label-width="isMobile ? undefined : 108"
          @update:model-value="emit('update:formModel', $event)"
        />
      </n-space>
    </n-spin>

    <template #action>
      <n-button @click="emit('update:visible', false)"> 取消 </n-button>
      <n-button v-if="errorMessage" :loading="loading" @click="emit('retry')">重试</n-button>
      <n-button
        type="primary"
        :loading="saving"
        :disabled="!config || !canSubmit || disabled"
        @click="emit('submit')"
      >
        保存
      </n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import type { EditableConfigResp, FormConfigItem } from '@/api';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import type { DynamicFormModel } from '@/components/shared/dynamicFormModel';
import TipBox from '@/components/shared/TipBox.vue';

defineProps<{
  visible: boolean;
  config: EditableConfigResp | null;
  formModel: DynamicFormModel;
  schema: FormConfigItem[];
  isMobile: boolean;
  saving: boolean;
  loading: boolean;
  errorMessage: string;
  submitError: string;
  disabled: boolean;
  canSubmit: boolean;
}>();

const emit = defineEmits<{
  'update:visible': [value: boolean];
  'update:formModel': [value: DynamicFormModel];
  retry: [];
  submit: [];
}>();
</script>

<style scoped>
.connect-edit-dialog__submit-error {
  margin-top: var(--sd-space-sm);
}
</style>
