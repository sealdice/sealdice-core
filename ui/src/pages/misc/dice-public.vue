<template>
  <main class="public-dice-page">
    <PageHeader title="公骰设置" description="配置公共骰服务的启用状态、账号信息和可用端点。">
      <n-switch
        :value="draft?.config.publicDiceEnable ?? false"
        :disabled="!draft || saving"
        :loading="enableMutation.isPending.value"
        @update:value="handleEnableUpdate"
      >
        <template #checked>启用</template>
        <template #unchecked>关闭</template>
      </n-switch>
      <n-button
        type="primary"
        :disabled="!canSave"
        :loading="saveMutation.isPending.value"
        @click="saveDraft"
      >
        <template #icon>
          <n-icon>
            <i-tabler-device-floppy />
          </n-icon>
        </template>
        保存
      </n-button>
    </PageHeader>

    <n-card class="public-dice-card" :bordered="false">
      <n-alert v-if="queryErrorText" class="public-dice-card__alert" type="error">
        {{ queryErrorText }}
      </n-alert>

      <n-spin :show="loadingInitial">
        <div v-if="draft" class="public-dice-card__body">
          <aside class="public-dice-card__seal" aria-hidden="true">
            <img :src="imgSeal" alt="" />
          </aside>
          <PublicDiceProfileForm
            v-model:config="draft.config"
            class="public-dice-card__form"
            :disabled="contentDisabled"
          />
        </div>
        <n-alert
          v-if="draft && !draft.config.publicDiceEnable"
          type="info"
          :bordered="false"
          class="public-dice-card__disabled-note"
        >
          公骰已关闭，配置保持只读；启用公骰后可继续编辑。
        </n-alert>
      </n-spin>

      <template #footer>
        <div v-if="draft" class="public-dice-card__footer">
          <PublicDiceEndpointSelector
            v-model:checked-row-keys="checkedRowKeys"
            :rows="endpointRows"
            :disabled="contentDisabled"
            :loading="publicDiceQuery.isFetching.value && endpointRows.length === 0"
          />
        </div>
      </template>
    </n-card>
  </main>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useMutation, useQuery } from '@tanstack/vue-query';
import type { DataTableRowKey } from 'naive-ui';
import {
  getSdApiV2ConfigPublicDice,
  getSdApiV2ConfigPublicDiceQueryKey,
  putSdApiV2ConfigPublicDice,
  putSdApiV2ConfigPublicDiceEnable,
  type PublicDiceInfoResp,
  type PublicDiceUpdateBodyWritable,
} from '@/api';
import imgSeal from '@/assets/seal.png';
import PublicDiceEndpointSelector from '@/components/public-dice/PublicDiceEndpointSelector.vue';
import PublicDiceProfileForm from '@/components/public-dice/PublicDiceProfileForm.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import { getErrorMessage } from '@/features/auth/error';
import { hasAccessToken } from '@/features/auth/state';
import { useUnsavedChanges } from '@/features/unsavedChanges';
import {
  buildPublicDicePayload,
  createPublicDiceDraft,
  getPublicDiceEndpointRows,
  isPublicDiceDirty,
  type PublicDiceDraft,
} from '@/features/publicDice/viewModel';

const message = useMessage();

const draft = ref<PublicDiceDraft | null>(null);
const initialDraft = ref<PublicDiceDraft | null>(null);

const publicDiceQuery = useQuery({
  queryKey: getSdApiV2ConfigPublicDiceQueryKey(),
  enabled: hasAccessToken,
  queryFn: async () => {
    const { data } = await getSdApiV2ConfigPublicDice({
      throwOnError: true,
    });
    return data.item;
  },
});

const endpointRows = computed(() =>
  getPublicDiceEndpointRows(publicDiceQuery.data.value?.endpoints)
);
const loadingInitial = computed(() => publicDiceQuery.isLoading.value && !draft.value);
const saving = computed(() => saveMutation.isPending.value || enableMutation.isPending.value);
const contentDisabled = computed(() => !draft.value?.config.publicDiceEnable || saving.value);
const dirty = computed(() => isPublicDiceDirty(draft.value, initialDraft.value));
const canSave = computed(
  () => Boolean(draft.value?.config.publicDiceEnable) && dirty.value && !saving.value
);
const queryErrorText = computed(() =>
  publicDiceQuery.isError.value
    ? getErrorMessage(publicDiceQuery.error.value, '读取公骰设置失败')
    : ''
);

const checkedRowKeys = computed<DataTableRowKey[]>({
  get: () => draft.value?.selectedEndpointIds ?? [],
  set: keys => {
    if (!draft.value) return;
    draft.value.selectedEndpointIds = keys.map(String);
  },
});

const saveMutation = useMutation({
  mutationFn: async (payload: PublicDiceUpdateBodyWritable) => {
    const { data } = await putSdApiV2ConfigPublicDice({
      body: payload,
      throwOnError: true,
    });
    return data.item;
  },
});

const enableMutation = useMutation({
  mutationFn: async (publicDiceEnable: boolean) => {
    const { data } = await putSdApiV2ConfigPublicDiceEnable({
      body: { publicDiceEnable },
      throwOnError: true,
    });
    return data.item;
  },
});

function syncDraft(info: PublicDiceInfoResp) {
  const next = createPublicDiceDraft(info);
  draft.value = structuredClone(next);
  initialDraft.value = structuredClone(next);
}

watch(
  () => publicDiceQuery.data.value,
  value => {
    if (!value) return;
    syncDraft(value);
  },
  { immediate: true }
);

async function submitCurrentDraft(successText: string) {
  if (!draft.value) return;
  const item = await saveMutation.mutateAsync(
    buildPublicDicePayload(draft.value.config, draft.value.selectedEndpointIds)
  );
  syncDraft(item);
  message.success(successText);
}

async function handleEnableUpdate(value: boolean) {
  if (!draft.value) return;
  const previous = draft.value.config.publicDiceEnable;
  draft.value.config.publicDiceEnable = value;
  try {
    const item = await enableMutation.mutateAsync(value);
    if (draft.value) {
      draft.value.config.publicDiceEnable = item.config.publicDiceEnable;
    }
    if (initialDraft.value) {
      initialDraft.value.config.publicDiceEnable = item.config.publicDiceEnable;
    }
    message.success(value ? '公骰已启用' : '公骰已关闭');
  } catch (error) {
    if (draft.value) {
      draft.value.config.publicDiceEnable = previous;
    }
    message.error(getErrorMessage(error, '保存公骰设置失败'));
  }
}

async function saveDraft() {
  try {
    await submitCurrentDraft('已保存');
  } catch (error) {
    message.error(getErrorMessage(error, '保存公骰设置失败'));
  }
}

useUnsavedChanges('public-dice', {
  label: '公骰设置',
  dirty,
  save: saveDraft,
  saving,
  canSave,
  confirmMessage: '公骰设置还有修改，确定要忽略？',
});
</script>

<style scoped>
.public-dice-page {
  min-width: 0;
}

.public-dice-card {
  --public-dice-gap: 28px;
}

.public-dice-card__alert {
  margin-bottom: 16px;
}

.public-dice-card__body {
  display: grid;
  grid-template-columns: minmax(180px, 240px) minmax(0, 1fr);
  gap: var(--public-dice-gap);
  align-items: stretch;
}

.public-dice-card__disabled-note {
  margin-top: 16px;
}

.public-dice-card__body--disabled .public-dice-card__seal img {
  filter: grayscale(1);
}

.public-dice-card__seal {
  display: grid;
  place-items: center;
  min-height: 248px;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated-soft);
  overflow: hidden;
}

.public-dice-card__seal img {
  display: block;
  width: min(76%, 190px);
  height: auto;
  transition: filter 0.2s ease;
}

.public-dice-card__form {
  min-width: 0;
}

.public-dice-card__footer {
  padding-top: 4px;
}

@media (max-width: 860px) {
  .public-dice-card__body {
    grid-template-columns: 1fr;
    gap: 18px;
  }

  .public-dice-card__seal {
    min-height: 180px;
  }

  .public-dice-card__seal img {
    width: min(52%, 150px);
  }
}

@media (max-width: 560px) {
  .public-dice-card :deep(.n-card-header) {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }

  .public-dice-card :deep(.n-card-header__extra) {
    width: 100%;
  }

  .public-dice-card :deep(.n-card-header__extra .n-space) {
    justify-content: space-between;
    width: 100%;
  }
}
</style>
