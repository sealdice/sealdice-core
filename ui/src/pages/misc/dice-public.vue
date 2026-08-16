<template>
  <main class="public-dice-page sd-page-flow">
    <PageHeader title="公骰设置">
      <n-flex align="center" size="small">
        <n-text depth="3">启用公骰</n-text>
        <n-switch
          :value="draft?.config.publicDiceEnable ?? false"
          :disabled="!draft || saving"
          :loading="enableMutation.isPending.value"
          aria-label="启用公骰"
          @update:value="handleEnableUpdate"
        />
      </n-flex>
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

    <TipBox v-if="queryErrorText" type="error" class="public-dice-alert">
      {{ queryErrorText }}
    </TipBox>

    <n-spin :show="loadingInitial">
      <div v-if="draft" class="public-dice-groups">
        <!-- 只读原因紧跟被禁用的表单，而不是放在页面底部。 -->
        <TipBox v-if="!draft.config.publicDiceEnable" type="info">
          公骰已关闭，配置保持只读；启用公骰后可继续编辑。
        </TipBox>

        <SettingCategoryBox title="公骰资料" padded>
          <div class="public-dice-profile">
            <aside class="public-dice-profile__seal" aria-hidden="true">
              <img :src="imgSeal" alt="" />
            </aside>
            <PublicDiceProfileForm
              v-model:config="draft.config"
              class="public-dice-profile__form"
              :disabled="contentDisabled"
            />
          </div>
        </SettingCategoryBox>

        <SettingCategoryBox title="上报终端" padded>
          <PublicDiceEndpointSelector
            v-model:checked-row-keys="checkedRowKeys"
            :rows="endpointRows"
            :disabled="contentDisabled"
            :loading="publicDiceQuery.isFetching.value && endpointRows.length === 0"
          />
        </SettingCategoryBox>
      </div>
    </n-spin>
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
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import TipBox from '@/components/shared/TipBox.vue';
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
    if (draft.value && initialDraft.value && dirty.value) return;
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

.public-dice-groups {
  display: grid;
  gap: var(--sd-space-2xs);
}

.public-dice-profile {
  display: grid;
  grid-template-columns: minmax(180px, 240px) minmax(0, 1fr);
  gap: 28px;
  align-items: stretch;
}

.public-dice-profile__seal {
  display: grid;
  place-items: center;
  min-height: 248px;
  border: 1px solid var(--sd-border);
  border-radius: var(--sd-radius-md);
  background: var(--sd-bg-elevated-soft);
  overflow: hidden;
}

.public-dice-profile__seal img {
  display: block;
  width: min(76%, 190px);
  height: auto;
  transition: filter 0.2s ease;
}

.public-dice-profile__form {
  min-width: 0;
}

@media (max-width: 860px) {
  .public-dice-profile {
    grid-template-columns: 1fr;
    gap: 18px;
  }

  .public-dice-profile__seal {
    min-height: 180px;
  }

  .public-dice-profile__seal img {
    width: min(52%, 150px);
  }
}
</style>
