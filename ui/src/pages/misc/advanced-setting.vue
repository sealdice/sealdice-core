<template>
  <main class="advanced-page">
    <h2 class="h-2">高级设置</h2>
    <TipBox type="warning" class="my-4">
      <n-text>
        此处是面向开发者或进阶用户的隐藏设置页，下列的设置项可能会对海豹核心的功能造成重大影响。<br />
        一些尚在测试的不稳定设置项，以及 <strong>普通骰主无需关注</strong> 的设置项会被放在此处。<br />
        此处的设置项不保证稳定提供，在未来版本随时可能会被移除。<br /><br />
        <strong>除非你知道自己在做什么，否则不要修改此处的任何设置项！</strong><br /><br />
        <em>
          如果你误操作修改了此处设置，希望恢复默认，请手动删除
          <n-text code>data/default/advanced.yaml</n-text>
          文件。
        </em>
      </n-text>
    </TipBox>

    <n-spin :show="pageBusy">
      <n-form :label-placement="isMobile ? 'top' : 'left'" label-width="auto">
        <n-form-item label="显示高级设置页">
          <template #label>
            <span>显示高级设置页</span>
            <n-tooltip>
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
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
                <n-icon><i-carbon-help-filled /></n-icon>
              </template>
              设置是否启用高级设置，关闭时下列设置无效
            </n-tooltip>
          </template>
          <n-switch v-model:value="config.enable" />
        </n-form-item>

        <h3>前端调试</h3>
        <n-form-item label="启用 Eruda 调试面板">
          <template #label>
            <span>Eruda 调试面板</span>
            <n-tooltip>
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
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

        <h3>自定义回复</h3>
        <n-form-item label="开启回复调试日志">
          <template #label>
            <span>回复调试日志</span>
            <n-tooltip>
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
              </template>
              开启自定义回复调试日志，打印字符细节
            </n-tooltip>
          </template>
          <n-checkbox v-model:checked="replyDebugMode">开启</n-checkbox>
        </n-form-item>

        <h3>跑团日志</h3>
        <n-form-item label="自定义后端 URL">
          <template #label>
            <span>自定义后端 URL</span>
            <n-tooltip>
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
              </template>
              设置第三方跑团日志后端 URL
            </n-tooltip>
          </template>
          <n-input v-model:value="config.storyLogBackendUrl" class="advanced-input advanced-input--long" />
        </n-form-item>

        <n-form-item label="API 版本">
          <template #label>
            <span>API 版本</span>
            <n-tooltip>
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
              </template>
              指定后端的 API 版本
            </n-tooltip>
          </template>
          <n-input v-model:value="config.storyLogApiVersion" class="advanced-input advanced-input--short" />
        </n-form-item>

        <n-form-item label="Token">
          <template #label>
            <span>Token</span>
            <n-tooltip>
              <template #trigger>
                <n-icon><i-carbon-help-filled /></n-icon>
              </template>
              指定传递给后端的 token
            </n-tooltip>
          </template>
          <n-input v-model:value="config.storyLogBackendToken" class="advanced-input advanced-input--long" />
        </n-form-item>

        <n-form-item v-if="modified" label="" label-width="1rem" class="mt-4">
          <n-flex>
            <n-button type="error" @click="reload">放弃改动</n-button>
            <n-button type="success" :loading="saveMutation.isPending.value" @click="save">
              保存设置
            </n-button>
          </n-flex>
        </n-form-item>
      </n-form>
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
import { hasAccessToken } from '@/features/auth/state';
import { normalizeAdvancedConfig, setAdvancedSettingsVisible } from '@/features/config/advancedSettings';
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
  { immediate: true },
);

watch(
  () => debugModeQuery.data.value?.item.value,
  value => {
    replyDebugMode.value = value ?? false;
    initialReplyDebugMode.value = value ?? false;
  },
  { immediate: true },
);

const modified = computed(() => {
  if (!initialConfig.value) return false;
  return JSON.stringify(config.value) !== JSON.stringify(initialConfig.value)
    || replyDebugMode.value !== initialReplyDebugMode.value;
});

useUnsavedChanges('advanced-setting', {
  label: '高级设置',
  dirty: modified,
  save,
  saving: computed(() => saveMutation.isPending.value),
  confirmMessage: '高级设置还有修改，确定要忽略？',
});

const saveMutation = useMutation({
  mutationFn: async () => {
    await putSdApiV2ConfigAdvanced({
      body: config.value,
      throwOnError: true,
    });
    await putSdApiV2CustomReplyDebugMode({
      body: { value: replyDebugMode.value },
      throwOnError: true,
    });
  },
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: getSdApiV2ConfigAdvancedQueryKey() });
    await queryClient.invalidateQueries({ queryKey: getSdApiV2CustomReplyDebugModeQueryKey() });
    setAdvancedSettingsVisible(config.value.show);
    initialConfig.value = normalizeAdvancedConfig(config.value);
    initialReplyDebugMode.value = replyDebugMode.value;
    message.success('已保存');
  },
  onError: () => {
    message.error('保存失败');
  },
});

async function save() {
  await saveMutation.mutateAsync();
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
  max-width: 1180px;
  text-align: left;
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
