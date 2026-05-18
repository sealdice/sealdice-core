<template>
  <main class="advanced-page">
    <n-affix v-if="modified" :top="120" :trigger-top="60" class="w-full">
      <TipBox type="error">
        <n-text type="error" tag="strong" class="text-lg">内容已修改，不要忘记保存！</n-text>
      </TipBox>
    </n-affix>

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
      <n-form label-placement="left" label-width="auto">
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
          <n-input v-model:value="config.storyLogBackendUrl" style="width: 30rem" />
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
          <n-input v-model:value="config.storyLogApiVersion" style="width: 10rem" />
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
          <n-input v-model:value="config.storyLogBackendToken" style="width: 30rem" />
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
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { useMessage } from 'naive-ui';
import {
  getSdApiV2ConfigAdvancedOptions,
  getSdApiV2ConfigAdvancedQueryKey,
  getSdApiV2CustomReplyDebugModeOptions,
  getSdApiV2CustomReplyDebugModeQueryKey,
  putSdApiV2ConfigAdvanced,
  putSdApiV2CustomReplyDebugMode,
  type AdvancedConfig,
} from '@/api';
import TipBox from '@/components/shared/TipBox.vue';
import { hasAccessToken } from '@/features/auth/state';
import { setAdvancedSettingsVisible } from '@/features/config/advancedSettings';

const message = useMessage();
const queryClient = useQueryClient();

const advancedConfigQuery = useQuery({
  ...getSdApiV2ConfigAdvancedOptions(),
  enabled: hasAccessToken,
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
const replyDebugMode = ref(false);
const modified = ref(false);

const pageBusy = computed(() => {
  return advancedConfigQuery.isFetching.value || debugModeQuery.isFetching.value;
});

watch(
  () => advancedConfigQuery.data.value?.item,
  value => {
    if (!value) return;
    config.value = structuredClone(value);
    modified.value = false;
  },
  { immediate: true },
);

watch(
  () => debugModeQuery.data.value?.item.value,
  value => {
    replyDebugMode.value = value ?? false;
    modified.value = false;
  },
  { immediate: true },
);

watch(config, () => {
  modified.value = true;
}, { deep: true });

watch(replyDebugMode, () => {
  modified.value = true;
});

const saveMutation = useMutation({
  mutationFn: async () => {
    await putSdApiV2ConfigAdvanced({
      body: { body: config.value },
      throwOnError: true,
    });
    await putSdApiV2CustomReplyDebugMode({
      body: { body: { value: replyDebugMode.value } },
      throwOnError: true,
    });
  },
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: getSdApiV2ConfigAdvancedQueryKey() });
    await queryClient.invalidateQueries({ queryKey: getSdApiV2CustomReplyDebugModeQueryKey() });
    setAdvancedSettingsVisible(config.value.show);
    modified.value = false;
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
  modified.value = false;
}
</script>

<style scoped>
.advanced-page {
  max-width: 1180px;
  text-align: left;
}
</style>
