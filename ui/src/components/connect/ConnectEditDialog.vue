<template>
  <n-modal
    :show="!!endpoint"
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
    :closable="!saving"
    :close-on-esc="!saving"
    @update:show="!$event && emit('close')"
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
        <TipBox v-if="config?.restartRequired" type="warning">
          保存后会重新连接此账号。Token、密码等敏感字段留空时保持原值不变。
        </TipBox>
        <DynamicForm
          :model-value="formModel"
          :schema="schema"
          :disabled="saving || disabled"
          :label-placement="isMobile ? 'top' : 'left'"
          :label-width="isMobile ? undefined : 108"
          @update:model-value="formModel = $event"
        />
      </n-space>
    </n-spin>

    <template #action>
      <n-button :disabled="saving" @click="emit('close')"> 取消 </n-button>
      <n-button v-if="errorMessage" :loading="loading" @click="configQuery.refetch()"
        >重试</n-button
      >
      <n-button
        type="primary"
        :loading="saving"
        :disabled="!config || !canSubmit || disabled || saving"
        @click="submit()"
      >
        保存
      </n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { storeToRefs } from 'pinia';
import {
  getSdApiV2ImconnectionByIdConfigOptions,
  getSdApiV2ImconnectionByIdConfigQueryKey,
  getSdApiV2ImconnectionQueryKey,
  putSdApiV2ImconnectionById,
  type EndPointInfo,
} from '@/api';
import DynamicForm from '@/components/shared/DynamicForm.vue';
import {
  buildDynamicFormInitialModel,
  fieldKeyOf,
  validateDynamicFormModel,
  type DynamicFormModel,
} from '@/components/shared/dynamicFormModel';
import TipBox from '@/components/shared/TipBox.vue';
import { useAuthStore } from '@/features/auth/store';
import { getErrorMessage } from '@/features/auth/error';
import { useTestMode } from '@/features/testMode/state';
import { notifyError } from '@/features/feedback/notifyError';

const props = defineProps<{ endpoint: EndPointInfo | null }>();
const emit = defineEmits<{ close: [] }>();
const message = useMessage();
const queryClient = useQueryClient();
const { hasAccessToken } = storeToRefs(useAuthStore());
const { isTestMode: disabled } = useTestMode();
const isMobile = useBreakpoints(breakpointsTailwind).smaller('md');
const formModel = ref<DynamicFormModel>({});
let loadedId = '';
let initialForm = '';
const configQuery = useQuery(
  computed(() => ({
    ...getSdApiV2ImconnectionByIdConfigOptions({ path: { id: props.endpoint?.id ?? '' } }),
    enabled: hasAccessToken.value && !!props.endpoint,
  }))
);
const config = computed(() => configQuery.data.value?.item ?? null);
const loading = computed(() => configQuery.isFetching.value);
const schema = computed(() => {
  const fields = config.value?.schema ?? [];
  if (config.value?.protocolKey !== 'officialqq' || formModel.value.useWebhook) return fields;
  return fields.filter(item => !['webhookPath', 'webhookPort'].includes(fieldKeyOf(item)));
});
const canSubmit = computed(() => validateDynamicFormModel(schema.value, formModel.value).valid);
const errorMessage = computed(() =>
  configQuery.error.value ? getErrorMessage(configQuery.error.value, '账号配置读取失败') : ''
);
const updateMutation = useMutation({
  mutationFn: async () => {
    const id = props.endpoint!.id;
    await putSdApiV2ImconnectionById({
      path: { id },
      body: { ...formModel.value },
      throwOnError: true,
    });
    return id;
  },
  onSuccess: async id => {
    message.success('账号配置已更新');
    if (props.endpoint?.id === id) emit('close');
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: getSdApiV2ImconnectionQueryKey() }),
      queryClient.invalidateQueries({
        queryKey: getSdApiV2ImconnectionByIdConfigQueryKey({ path: { id } }),
      }),
    ]);
  },
  onError: error => notifyError(error, message, '账号配置更新失败'),
});
const saving = computed(() => updateMutation.isPending.value);
const submitError = computed(() =>
  updateMutation.error.value ? getErrorMessage(updateMutation.error.value, '账号配置更新失败') : ''
);
watch(
  () => props.endpoint?.id,
  () => {
    loadedId = '';
    formModel.value = {};
    updateMutation.reset();
  },
  { flush: 'sync' }
);
watch(
  [config, () => props.endpoint?.id],
  ([value, id]) => {
    // 未编辑时接受最新配置；焦点刷新或后台重连不能覆盖正在填写的内容。
    if (!value || !id || (loadedId === id && JSON.stringify(formModel.value) !== initialForm))
      return;
    formModel.value = { ...buildDynamicFormInitialModel(value.schema ?? []), ...value.config };
    initialForm = JSON.stringify(formModel.value);
    loadedId = id;
  },
  { immediate: true }
);
function submit() {
  if (!props.endpoint || !config.value || !canSubmit.value || disabled.value || saving.value)
    return;
  updateMutation.mutate();
}
</script>

<style scoped>
.connect-edit-dialog__submit-error {
  margin-top: var(--sd-space-sm);
}
</style>
