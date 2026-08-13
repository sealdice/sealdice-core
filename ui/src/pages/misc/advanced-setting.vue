<template>
  <main class="advanced-page">
    <PageHeader title="高级设置" unsaved-scope="advanced-setting">
      <n-button secondary :disabled="!modified" @click="reload">
        <template #icon>
          <n-icon><i-tabler-arrow-back-up /></n-icon>
        </template>
        放弃改动
      </n-button>
      <n-button type="primary" :loading="saving" :disabled="saving || !modified" @click="save">
        <template #icon>
          <n-icon><i-tabler-device-floppy /></n-icon>
        </template>
        保存设置
      </n-button>
    </PageHeader>
    <TipBox type="warning">
      <p>
        <strong>除非你知道自己在做什么，否则不要修改此处的任何设置项。</strong>
      </p>
      <p>这些设置可能对海豹核心的功能造成重大影响，且不保证稳定提供，在未来版本随时可能被移除。</p>
    </TipBox>

    <TipBox type="info">
      如需恢复默认，请手动删除
      <n-text code class="sd-code-text">data/default/advanced.yaml</n-text>
      文件。
    </TipBox>

    <n-spin :show="pageBusy">
      <div class="advanced-groups">
        <SettingCategoryBox title="页面与功能" padded>
          <n-form :label-placement="isMobile ? 'top' : 'left'" label-width="auto">
            <n-form-item label="显示高级设置页">
              <template #label>
                <span>显示高级设置页</span>
                <n-tooltip>
                  <template #trigger>
                    <n-icon><i-tabler-help-circle /></n-icon>
                  </template>
                  设置是否显示高级设置页，只影响展示
                </n-tooltip>
              </template>
              <n-switch v-model:value="config.show" />
            </n-form-item>

            <n-form-item label="启用高级设置">
              <template #label>
                <span>启用高级设置</span>
                <n-tooltip>
                  <template #trigger>
                    <n-icon><i-tabler-help-circle /></n-icon>
                  </template>
                  设置是否启用高级设置，关闭时下列设置无效
                </n-tooltip>
              </template>
              <n-switch v-model:value="config.enable" />
            </n-form-item>
          </n-form>
        </SettingCategoryBox>

        <SettingCategoryBox title="前端调试" padded>
          <n-form :label-placement="isMobile ? 'top' : 'left'" label-width="auto">
            <n-form-item label="启用 Eruda 调试面板">
              <template #label>
                <span>Eruda 调试面板</span>
                <n-tooltip>
                  <template #trigger>
                    <n-icon><i-tabler-help-circle /></n-icon>
                  </template>
                  仅对当前浏览器生效，状态保存在本机，不会同步到后端。
                </n-tooltip>
              </template>
              <n-switch
                :value="erudaEnabled"
                :loading="erudaPending"
                @update:value="handleErudaToggle"
              />
            </n-form-item>
          </n-form>
        </SettingCategoryBox>

        <SettingCategoryBox title="自定义回复" padded>
          <n-form :label-placement="isMobile ? 'top' : 'left'" label-width="auto">
            <n-form-item label="开启回复调试日志">
              <template #label>
                <span>回复调试日志</span>
                <n-tooltip>
                  <template #trigger>
                    <n-icon><i-tabler-help-circle /></n-icon>
                  </template>
                  开启自定义回复调试日志，打印字符细节
                </n-tooltip>
              </template>
              <n-checkbox v-model:checked="replyDebugMode">开启</n-checkbox>
            </n-form-item>
          </n-form>
        </SettingCategoryBox>

        <SettingCategoryBox title="跑团日志" padded>
          <n-form :label-placement="isMobile ? 'top' : 'left'" label-width="auto">
            <n-form-item label="自定义后端 URL">
              <n-input
                v-model:value="config.storyLogBackendUrl"
                class="advanced-input advanced-input--long"
              />
            </n-form-item>

            <n-form-item label="API 版本">
              <n-input
                v-model:value="config.storyLogApiVersion"
                class="advanced-input advanced-input--short"
              />
            </n-form-item>

            <n-form-item label="Token">
              <n-input
                v-model:value="config.storyLogBackendToken"
                class="advanced-input advanced-input--long"
              />
            </n-form-item>
          </n-form>
        </SettingCategoryBox>
      </div>
    </n-spin>
  </main>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { breakpointsTailwind, useBreakpoints } from '@vueuse/core';
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useMessage } from 'naive-ui';
import {
  getSdApiV2ConfigAdvanced,
  getSdApiV2ConfigAdvancedQueryKey,
  getSdApiV2CustomReplyDebugModeOptions,
  getSdApiV2CustomReplyDebugModeQueryKey,
  putSdApiV2ConfigAdvanced,
  putSdApiV2CustomReplyDebugMode,
  type AdvancedConfig,
} from '@/api';
import TipBox from '@/components/shared/TipBox.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
import { hasAccessToken } from '@/features/auth/state';
import {
  normalizeAdvancedConfig,
  setAdvancedSettingsVisible,
} from '@/features/config/advancedSettings';
import { isErudaEnabled, setErudaEnabled } from '@/features/debug/eruda';
import { useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const queryClient = useQueryClient();
const breakpoints = useBreakpoints(breakpointsTailwind);
const isMobile = breakpoints.smaller('md');

const advancedConfigQuery = useQuery({
  queryKey: getSdApiV2ConfigAdvancedQueryKey(),
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2ConfigAdvanced({
      throwOnError: true,
    });
    return normalizeAdvancedConfig(data.item);
  },
});
const debugModeQuery = useQuery({
  ...getSdApiV2CustomReplyDebugModeOptions(),
  enabled: hasAccessToken,
});

const config = ref<AdvancedConfig>({
  show: false,
  enable: false,
  storyLogBackendUrl: '',
  storyLogApiVersion: '',
  storyLogBackendToken: '',
});
const erudaEnabled = ref(isErudaEnabled());
const erudaPending = ref(false);
const replyDebugMode = ref(false);
const initialConfig = ref<AdvancedConfig | null>(null);
const initialReplyDebugMode = ref(false);

const pageBusy = computed(() => {
  return advancedConfigQuery.isFetching.value || debugModeQuery.isFetching.value;
});

watch(
  () => advancedConfigQuery.data.value,
  value => {
    if (!value) return;
    const next = normalizeAdvancedConfig(value);
    config.value = next;
    initialConfig.value = normalizeAdvancedConfig(next);
  },
  { immediate: true }
);

watch(
  () => debugModeQuery.data.value?.item.value,
  value => {
    replyDebugMode.value = value ?? false;
    initialReplyDebugMode.value = value ?? false;
  },
  { immediate: true }
);

const modified = computed(() => {
  if (!initialConfig.value) return false;
  return (
    JSON.stringify(config.value) !== JSON.stringify(initialConfig.value) ||
    replyDebugMode.value !== initialReplyDebugMode.value
  );
});

const advancedSaveMutation = useMutation({
  mutationFn: async () => {
    const { data } = await putSdApiV2ConfigAdvanced({
      body: config.value,
      throwOnError: true,
    });
    return data.item;
  },
});

const debugModeSaveMutation = useMutation({
  mutationFn: async () => {
    const { data } = await putSdApiV2CustomReplyDebugMode({
      body: { value: replyDebugMode.value },
      throwOnError: true,
    });
    return data.item;
  },
});

const saving = computed(
  () => advancedSaveMutation.isPending.value || debugModeSaveMutation.isPending.value
);

useUnsavedChanges('advanced-setting', {
  label: '高级设置',
  dirty: modified,
  save,
  saving,
  confirmMessage: '高级设置还有修改，确定要忽略？',
});

async function save() {
  if (saving.value) return;

  const advancedChanged =
    Boolean(initialConfig.value) &&
    JSON.stringify(config.value) !== JSON.stringify(initialConfig.value);
  const debugModeChanged = replyDebugMode.value !== initialReplyDebugMode.value;
  const tasks: Array<{ kind: 'advanced' | 'debug'; promise: Promise<unknown> }> = [];

  if (advancedChanged) {
    tasks.push({ kind: 'advanced', promise: advancedSaveMutation.mutateAsync() });
  }
  if (debugModeChanged) {
    tasks.push({ kind: 'debug', promise: debugModeSaveMutation.mutateAsync() });
  }
  if (!tasks.length) return;

  const results = await Promise.allSettled(tasks.map(task => task.promise));
  const succeeded = new Set(
    results
      .map((result, index) => (result.status === 'fulfilled' ? tasks[index]?.kind : null))
      .filter((kind): kind is 'advanced' | 'debug' => Boolean(kind))
  );
  const failed = results
    .map((result, index) => (result.status === 'rejected' ? tasks[index]?.kind : null))
    .filter((kind): kind is 'advanced' | 'debug' => Boolean(kind))
    .map(kind => (kind === 'advanced' ? '高级配置' : '回复调试日志'));

  if (succeeded.has('advanced')) {
    await queryClient.invalidateQueries({ queryKey: getSdApiV2ConfigAdvancedQueryKey() });
    setAdvancedSettingsVisible(config.value.show);
    initialConfig.value = normalizeAdvancedConfig(config.value);
  }
  if (succeeded.has('debug')) {
    await queryClient.invalidateQueries({ queryKey: getSdApiV2CustomReplyDebugModeQueryKey() });
    initialReplyDebugMode.value = replyDebugMode.value;
  }

  if (failed.length) {
    const succeededLabels = Array.from(succeeded).map(kind =>
      kind === 'advanced' ? '高级配置' : '回复调试日志'
    );
    const succeededText = succeededLabels.length ? `已保存${succeededLabels.join('、')}；` : '';
    message.error(`${succeededText}保存失败：${failed.join('、')}`);
  } else {
    message.success('已保存');
  }
}

async function reload() {
  await queryClient.invalidateQueries({ queryKey: getSdApiV2ConfigAdvancedQueryKey() });
  await queryClient.invalidateQueries({ queryKey: getSdApiV2CustomReplyDebugModeQueryKey() });
}

async function handleErudaToggle(value: boolean) {
  erudaPending.value = true;
  try {
    await setErudaEnabled(value);
    erudaEnabled.value = value;
    message.success(value ? '已开启 Eruda 调试面板' : '已关闭 Eruda 调试面板');
  } catch {
    erudaEnabled.value = isErudaEnabled();
    message.error('Eruda 调试面板切换失败');
  } finally {
    erudaPending.value = false;
  }
}
</script>

<style scoped>
.advanced-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.advanced-groups {
  display: grid;
  gap: var(--sd-space-2xs);
}

.advanced-input {
  width: 100%;
}

.advanced-input--long {
  max-width: 30rem;
}

.advanced-input--short {
  max-width: 10rem;
}

@media screen and (max-width: 767.9px) {
  .advanced-input--short,
  .advanced-input--long {
    max-width: none;
  }
}
</style>
