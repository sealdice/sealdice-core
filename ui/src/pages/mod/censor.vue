<template>
  <main class="censor-page">
    <PageHeader title="拦截管理" :unsaved-scope="['censor-config', 'censor-reload']">
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
      <!-- 待重载时由待处理状态条承载重载操作，此处不再重复。 -->
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
      <n-tabs v-model:value="tab">
        <n-tab-pane tab="拦截设置" name="setting">
          <n-spin :show="configQuery.isFetching.value">
            <CensorConfigView v-model:config="configDraft.currentConfig.value" />
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
import { computed, reactive, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';
import { getSdApiV2CensorFilesTemplateToml, getSdApiV2CensorFilesTemplateTxt } from '@/api';
import { downloadApiFile } from '@/api/download';
import CensorConfigView from '@/components/censor/CensorConfigView.vue';
import CensorFilesView from '@/components/censor/CensorFilesView.vue';
import CensorLogView from '@/components/censor/CensorLogView.vue';
import CensorWordsView from '@/components/censor/CensorWordsView.vue';
import CensorWordTip from '@/components/censor/CensorWordTip.vue';
import FeatureDisabledState from '@/components/shared/FeatureDisabledState.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import { useCensorConfigDraft } from '@/features/censor/configDraft';
import { useCensorMutations } from '@/features/censor/mutations';
import {
  useCensorConfigQuery,
  useCensorFilesQuery,
  useCensorLogsQuery,
  useCensorStatusQuery,
  useCensorWordsQuery,
} from '@/features/censor/queries';
import { createDefaultCensorLogQuery } from '@/features/censor/viewModel';
import { usePendingReload, useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const queryClient = useQueryClient();

const tab = ref<'setting' | 'word' | 'log'>('setting');
const needReload = ref(false);
const censorEnable = ref(false);
const logQuery = reactive(createDefaultCensorLogQuery());

const statusQuery = useCensorStatusQuery();

watch(
  () => statusQuery.data.value,
  item => {
    censorEnable.value = Boolean(item?.enable);
  },
  { immediate: true }
);

const enabledForContent = computed(() => censorEnable.value);
const configDraft = useCensorConfigDraft();

const configQuery = useCensorConfigQuery(
  computed(() => enabledForContent.value && tab.value === 'setting')
);
const filesQuery = useCensorFilesQuery(
  computed(() => enabledForContent.value && tab.value === 'word')
);
const wordsQuery = useCensorWordsQuery(
  computed(() => enabledForContent.value && tab.value === 'word')
);
const logsQuery = useCensorLogsQuery(
  logQuery,
  computed(() => enabledForContent.value && tab.value === 'log')
);

watch(
  () => configQuery.data.value,
  value => {
    if (!value) return;
    configDraft.syncRemote(value);
  },
  { immediate: true }
);

const files = computed(() => filesQuery.data.value ?? []);
const words = computed(() => wordsQuery.data.value ?? []);
const logs = computed(() => logsQuery.data.value?.data ?? []);
const logTotal = computed(() => Number(logsQuery.data.value?.total ?? 0));
const statusBusy = computed(() => restartMutation.isPending.value || stopMutation.isPending.value);

const { restartMutation, stopMutation, saveConfigMutation, uploadFileMutation } =
  useCensorMutations({
    queryClient,
    message,
    getConfigPayload: () => configDraft.currentConfig.value,
    onReloaded: () => {
      needReload.value = false;
      censorEnable.value = true;
    },
    onStopped: () => {
      needReload.value = false;
      censorEnable.value = false;
    },
    onConfigSaved: () => {
      configDraft.commitSaved();
      needReload.value = true;
    },
    onFilesChanged: () => {
      needReload.value = true;
    },
  });

useUnsavedChanges('censor-config', {
  label: '拦截设置',
  dirty: computed(() => enabledForContent.value && configDraft.dirty.value),
  save: saveConfig,
  saving: computed(() => saveConfigMutation.isPending.value),
  canSave: computed(() => enabledForContent.value && configDraft.dirty.value),
  discard: () => configDraft.resetToRemote(),
  confirmMessage: '拦截设置还有修改，确定要忽略？',
});

usePendingReload('censor-reload', {
  label: '拦截配置',
  pending: computed(() => enabledForContent.value && needReload.value),
  reload: restartCensor,
  reloading: computed(() => restartMutation.isPending.value),
  canReload: computed(() => !restartMutation.isPending.value),
  actionText: '重载拦截',
});

watch(
  () => [logQuery.pageNum, logQuery.pageSize] as const,
  () => {
    if (tab.value === 'log' && enabledForContent.value) {
      void logsQuery.refetch();
    }
  }
);

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
.censor-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.censor-tabs :deep(.n-tabs-nav-scroll-content) {
  min-width: max-content;
}

@media screen and (max-width: 639.9px) {
  .censor-tabs :deep(.n-tabs-nav-scroll-content) {
    justify-content: flex-start !important;
  }
}
</style>
