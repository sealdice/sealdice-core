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
    <n-alert v-if="errorMessage" type="error" :show-icon="false">
      {{ errorMessage }}
    </n-alert>
    <n-alert
      v-if="submitError"
      type="error"
      :show-icon="false"
      class="connect-edit-dialog__submit-error"
    >
      {{ submitError }}。请检查配置后重试。
    </n-alert>
    <n-spin :show="loading && !config">
      <n-space vertical size="large">
        <n-alert v-if="config?.restartRequired" type="warning" :show-icon="false">
          保存后会重新连接此账号。Token、密码等敏感字段留空时保持原值不变。
        </n-alert>
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
