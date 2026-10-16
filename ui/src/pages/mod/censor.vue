<template>
  <main class="censor-page sd-page-flow">
    <PageHeader title="拦截管理">
      <n-flex align="center" size="small">
        <n-text depth="3">启用拦截</n-text>
        <n-switch
          v-model:value="censorEnable"
          :loading="statusBusy"
          :disabled="statusBusy || statusQuery.isFetching.value"
          aria-label="启用拦截"
          @update:value="enableChange"
        />
      </n-flex>
      <n-button
        v-show="censorEnable && !needReload"
        type="primary"
        :loading="restartMutation.isPending.value"
        :disabled="restartMutation.isPending.value"
        @click="restartCensor"
      >
        <template #icon>
          <i-tabler-refresh />
        </template>
        重载拦截
      </n-button>
    </PageHeader>
    <template v-if="censorEnable">
      <n-tabs v-model:value="tab" class="censor-tabs sd-scrollable-tabs">
        <n-tab-pane tab="拦截设置" name="setting">
          <n-spin :show="configQuery.isFetching.value">
            <CensorConfigView
              v-model:config="currentConfig"
              :saving="saveConfigMutation.isPending.value"
              :modified="configDirty"
              @save="saveConfig"
            />
          </n-spin>
        </n-tab-pane>

        <n-tab-pane tab="敏感词管理" name="word">
          <n-spin
            :show="
              filesQuery.isFetching.value ||
              wordsQuery.isFetching.value ||
              uploadFileMutation.isPending.value
            "
          >
            <CensorWordTip />
            <CensorFilesView
              :files="files"
              :upload-file="uploadFile"
              :download-toml-template="downloadTomlTemplate"
              :download-txt-template="downloadTxtTemplate"
            />
            <CensorWordsView :words="words" />
          </n-spin>
        </n-tab-pane>

        <n-tab-pane tab="拦截日志" name="log">
          <CensorLogView
            v-model:query="logQuery"
            :logs="logs"
            :total="logTotal"
            :loading="logsQuery.isFetching.value"
            @refresh="refreshLogs"
          />
        </n-tab-pane>
      </n-tabs>
    </template>
    <template v-else>
      <FeatureDisabledState
        title="拦截已关闭"
        description="启用后可管理拦截设置、敏感词和拦截日志。"
      />
    </template>
  </main>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import {
  getSdApiV2CensorFilesTemplateToml,
  getSdApiV2CensorFilesTemplateTxt,
  getSdApiV2CensorStatusOptions,
  getSdApiV2CensorStatusQueryKey,
  getSdApiV2CensorConfigOptions,
  getSdApiV2CensorConfigQueryKey,
  getSdApiV2CensorFilesOptions,
  getSdApiV2CensorFilesQueryKey,
  getSdApiV2CensorWordsOptions,
  getSdApiV2CensorWordsQueryKey,
  getSdApiV2CensorLogsPageOptions,
  getSdApiV2CensorLogsPageQueryKey,
  postSdApiV2CensorConfig,
  postSdApiV2CensorRestart,
  postSdApiV2CensorStop,
  type GetSdApiV2CensorLogsPageResponse,
} from '@/api';
import { downloadApiFile } from '@/api/download';
import CensorConfigView from '@/components/censor/CensorConfigView.vue';
import CensorFilesView from '@/components/censor/CensorFilesView.vue';
import CensorLogView from '@/components/censor/CensorLogView.vue';
import CensorWordsView from '@/components/censor/CensorWordsView.vue';
import CensorWordTip from '@/components/censor/CensorWordTip.vue';
import FeatureDisabledState from '@/components/shared/FeatureDisabledState.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import {
  createDefaultCensorLogQuery,
  createDefaultCensorConfig,
  cloneCensorConfig,
  isCensorConfigDirty,
} from '@/features/censor/viewModel';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';
import { useUploadStore } from '@/features/upload/store';
import {
  getTestModeBlockMessage,
  isTestModeApiError,
  isTestModeResponse,
} from '@/features/testMode/state';
import { notifyError } from '@/features/feedback/notifyError';
import { usePendingReload, useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const queryClient = useQueryClient();
const { hasAccessToken } = storeToRefs(useAuthStore());
const uploads = useUploadStore();

const tab = ref<'setting' | 'word' | 'log'>('setting');
const needReload = ref(false);
const censorEnable = ref(false);
const logQuery = ref(createDefaultCensorLogQuery());

const statusQuery = useQuery({
  ...getSdApiV2CensorStatusOptions(),
  enabled: hasAccessToken,
  select: data => data.item,
});

watch(
  () => statusQuery.data.value,
  item => {
    censorEnable.value = Boolean(item?.enable);
  },
  { immediate: true }
);

const currentConfig = ref(createDefaultCensorConfig());
const initialConfig = ref(createDefaultCensorConfig());
const configDirty = computed(() => isCensorConfigDirty(currentConfig.value, initialConfig.value));
let configReady = false;
const configQuery = useQuery({
  ...getSdApiV2CensorConfigOptions(),
  enabled: computed(() => hasAccessToken.value && censorEnable.value && tab.value === 'setting'),
  select: data => data.item,
});
const filesQuery = useQuery({
  ...getSdApiV2CensorFilesOptions(),
  enabled: computed(() => hasAccessToken.value && censorEnable.value && tab.value === 'word'),
  select: data => data.item.data ?? [],
});
const wordsQuery = useQuery({
  ...getSdApiV2CensorWordsOptions(),
  enabled: computed(() => hasAccessToken.value && censorEnable.value && tab.value === 'word'),
  select: data => data.item.data ?? [],
});
const logsQuery = useQuery(
  computed(() => ({
    ...getSdApiV2CensorLogsPageOptions({ query: { ...logQuery.value } }),
    enabled: hasAccessToken.value && censorEnable.value && tab.value === 'log',
    select: (data: GetSdApiV2CensorLogsPageResponse) => data.item,
  }))
);

watch(
  configQuery.data,
  value => {
    if (!value || (configReady && configDirty.value)) return;
    currentConfig.value = cloneCensorConfig(value);
    initialConfig.value = cloneCensorConfig(value);
    configReady = true;
  },
  { immediate: true }
);

const files = computed(() => filesQuery.data.value ?? []);
const words = computed(() => wordsQuery.data.value ?? []);
const logs = computed(() => logsQuery.data.value?.data ?? []);
const logTotal = computed(() => Number(logsQuery.data.value?.total ?? 0));
const statusBusy = computed(() => restartMutation.isPending.value || stopMutation.isPending.value);

function refreshCensor() {
  return Promise.all(
    [
      getSdApiV2CensorStatusQueryKey(),
      getSdApiV2CensorConfigQueryKey(),
      getSdApiV2CensorFilesQueryKey(),
      getSdApiV2CensorWordsQueryKey(),
      getSdApiV2CensorLogsPageQueryKey(),
    ].map(queryKey => queryClient.invalidateQueries({ queryKey }))
  );
}

const restartMutation = useMutation({
  mutationFn: async (intent: 'enable' | 'reload' = 'reload') => {
    const { data } = await postSdApiV2CensorRestart({ throwOnError: true });
    return { item: data.item, intent };
  },
  onSuccess: async ({ item, intent }) => {
    if (isTestModeResponse(item)) {
      message.warning('展示模式无法启用拦截');
      return;
    }
    message.success(intent === 'enable' ? '拦截已启用' : '重载拦截成功');
    needReload.value = false;
    censorEnable.value = true;
    await refreshCensor();
  },
  onError: error => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    notifyError(error, message, '重载拦截失败');
  },
});
const stopMutation = useMutation({
  mutationFn: () => postSdApiV2CensorStop({ throwOnError: true }),
  onSuccess: async () => {
    message.success('拦截已关闭');
    needReload.value = false;
    censorEnable.value = false;
    await refreshCensor();
  },
  onError: error => notifyError(error, message, '关闭拦截失败'),
});
const saveConfigMutation = useMutation({
  mutationFn: async () => {
    const snapshot = cloneCensorConfig(currentConfig.value);
    await postSdApiV2CensorConfig({ body: snapshot, throwOnError: true });
    return snapshot;
  },
  onSuccess: async snapshot => {
    initialConfig.value = cloneCensorConfig(snapshot);
    needReload.value = true;
    message.success('保存设置成功');
    await queryClient.invalidateQueries({ queryKey: getSdApiV2CensorConfigQueryKey() });
  },
  onError: error => notifyError(error, message, '保存设置失败'),
});
const uploadFileMutation = useMutation({
  mutationFn: (file: File) => uploads.upload(file, 'censor'),
  onSuccess: async () => {
    message.success('上传完成，请在全部操作完成后，手动重载拦截');
    needReload.value = true;
    await queryClient.invalidateQueries({ queryKey: getSdApiV2CensorFilesQueryKey() });
  },
  onError: error => notifyError(error, message, '上传失败'),
});

useUnsavedChanges('censor-config', {
  label: '拦截设置',
  dirty: computed(() => censorEnable.value && configDirty.value),
  save: saveConfig,
  saving: computed(() => saveConfigMutation.isPending.value),
  canSave: computed(() => censorEnable.value && configDirty.value),
  discard: () => {
    currentConfig.value = cloneCensorConfig(initialConfig.value);
  },
  confirmMessage: '拦截设置还有修改，确定要忽略？',
});

usePendingReload('censor-reload', {
  label: '拦截配置',
  pending: computed(() => censorEnable.value && needReload.value),
  reload: restartCensor,
  reloading: computed(() => restartMutation.isPending.value),
  canReload: computed(() => !restartMutation.isPending.value),
  actionText: '重载拦截',
});

async function restartCensor() {
  await restartMutation.mutateAsync('reload');
}

async function stopCensor() {
  await stopMutation.mutateAsync();
}

async function enableChange(value: boolean | number | string) {
  const next = value === true;
  try {
    if (next) {
      await restartMutation.mutateAsync('enable');
    } else {
      await stopCensor();
    }
  } catch {
    censorEnable.value = !next;
  }
}

async function saveConfig() {
  await saveConfigMutation.mutateAsync();
}

async function uploadFile(file: File) {
  await uploadFileMutation.mutateAsync(file);
}

async function downloadTomlTemplate() {
  await downloadApiFile(
    getSdApiV2CensorFilesTemplateToml({
      responseType: 'blob',
      throwOnError: true,
    }),
    '词库模板.toml'
  );
}

async function downloadTxtTemplate() {
  await downloadApiFile(
    getSdApiV2CensorFilesTemplateTxt({
      responseType: 'blob',
      throwOnError: true,
    }),
    '词库模板.txt'
  );
}

function refreshLogs() {
  void logsQuery.refetch();
}
</script>

<style scoped>
.censor-tabs :deep(.n-tabs-nav-scroll-content) {
  min-width: max-content;
}

@media screen and (max-width: 639.9px) {
  .censor-tabs :deep(.n-tabs-nav-scroll-content) {
    justify-content: flex-start !important;
  }
}
</style>
