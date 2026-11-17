<template>
  <main class="base-setting-page sd-page-flow">
    <PageHeader title="基本设置">
      <n-button
        secondary
        :disabled="!draft.dirty.value || saveMutation.isPending.value"
        @click="resetChanges"
      >
        <template #icon>
          <n-icon><i-tabler-arrow-back-up /></n-icon>
        </template>
        放弃改动
      </n-button>
      <n-button
        type="primary"
        :loading="saveMutation.isPending.value"
        :disabled="!draft.dirty.value || saveMutation.isPending.value"
        @click="saveChanges"
      >
        <template #icon>
          <n-icon><i-tabler-device-floppy /></n-icon>
        </template>
        保存设置
      </n-button>
    </PageHeader>

    <BaseSettingSearchBar
      v-model:keyword="searchKeyword"
      :results="searchResults"
      @select="jumpToField"
    />

    <n-spin :show="pageBusy">
      <ResponsiveTabs
        v-model:value="activeTab"
        class="setting-tabs"
        :compact-at="760"
        :options="tabOptions"
      >
        <template #panel="{ option }">
          <div class="setting-groups">
            <SettingCategoryBox
              v-for="group in findTab(option.value)?.groups ?? []"
              :key="group.id"
              :title="group.title"
              :collapsible="group.collapsible"
              :expanded="expandedGroups[group.id]"
              :wide="isBaseSettingGroupWide(group.id)"
              @toggle="toggleGroup(group.id)"
            >
              <template #notes>
                <TipBox
                  v-for="(note, noteIndex) in group.notes"
                  :key="`${group.id}-${noteIndex}`"
                  :type="note.tone === 'warning' ? 'warning' : 'info'"
                >
                  <div v-for="(line, lineIndex) in note.lines" :key="lineIndex">
                    {{ line }}
                  </div>
                </TipBox>
              </template>

              <template v-if="currentValue">
                <BaseSettingFieldRenderer
                  v-for="field in group.fields"
                  :key="field.id"
                  :field="field"
                  :model="currentValue"
                  :initial-model="initialValue"
                  :is-container-mode="isContainerMode"
                  :busy-action-id="busyActionId"
                  :highlighted="highlightedFieldId === field.id"
                  :run-action="runAction"
                  @update-field="updateField"
                />
              </template>
            </SettingCategoryBox>
          </div>
        </template>
      </ResponsiveTabs>
    </n-spin>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { getSdApiV2BaseSettingValueOptions } from '@/api';
import { storeToRefs } from 'pinia';
import { useAuthStore } from '@/features/auth/store';
import BaseSettingFieldRenderer from '@/components/base-setting/BaseSettingFieldRenderer.vue';
import BaseSettingSearchBar from '@/components/base-setting/BaseSettingSearchBar.vue';
import SettingCategoryBox from '@/components/settings-panel/SettingCategoryBox.vue';
import PageHeader from '@/components/shared/PageHeader.vue';
import ResponsiveTabs from '@/components/shared/ResponsiveTabs.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { useBaseOverview } from '@/features/base/useBaseOverview';
import { useBaseSettingDraft } from '@/features/baseSetting/draft';
import { useBaseSettingMutations } from '@/features/baseSetting/mutations';
import {
  buildBaseSettingPatch,
  buildBaseSettingSearchIndex,
  cloneBaseSettingValue,
  isBaseSettingGroupWide,
  searchBaseSettingFields,
  validateBaseSettingPatchFormats,
  type BaseSettingSearchEntry,
} from '@/features/baseSetting/viewModel';
import { baseSettingSchema } from '@/features/baseSetting/schema';
import { useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const { hasAccessToken } = storeToRefs(useAuthStore());
const { overview } = useBaseOverview();

const activeTab = ref('master-notice');
const searchKeyword = ref('');
const expandedGroups = reactive<Record<string, boolean>>({});
const highlightedFieldId = ref('');
const busyActionId = ref<string | null>(null);

const schema = baseSettingSchema;
const valueQuery = useQuery({
  ...getSdApiV2BaseSettingValueOptions(),
  enabled: hasAccessToken,
  select: data => data.item,
});
const draft = useBaseSettingDraft();

for (const tab of schema.tabs) {
  for (const group of tab.groups) {
    expandedGroups[group.id] = group.collapsible ? Boolean(group.defaultExpanded) : true;
  }
}

watch(
  () => valueQuery.data.value,
  value => {
    if (!value) return;
    draft.syncRemote(value);
  },
  { immediate: true }
);

const tabs = computed(() => schema.tabs);
const tabOptions = computed(() =>
  tabs.value.map(tab => ({
    label: tab.title,
    value: tab.id,
  }))
);
const searchIndex = computed(() => buildBaseSettingSearchIndex(schema));
const searchResults = computed(() =>
  searchBaseSettingFields(searchIndex.value, searchKeyword.value)
);
const currentValue = computed(() => draft.currentValue.value);
const initialValue = computed(() => draft.initialValue.value);
const pageBusy = computed(() => valueQuery.isFetching.value);
const isContainerMode = computed(() => overview.value?.runtime.containerMode === true);

const { saveMutation, mailTestMutation, upgradeMutation } = useBaseSettingMutations(message);

useUnsavedChanges('base-setting', {
  label: '基本设置',
  dirty: draft.dirty,
  save: saveChanges,
  saving: computed(() => saveMutation.isPending.value),
  canSave: draft.dirty,
  discard: resetChanges,
  confirmMessage: '基本设置还有修改，确定要忽略？',
});

async function saveChanges() {
  if (saveMutation.isPending.value || !draft.currentValue.value || !draft.initialValue.value)
    return;
  // 在任何异步工作之前固定请求和基线，保存期间的后续编辑继续保持未保存状态。
  const submittedSnapshot = cloneBaseSettingValue(draft.currentValue.value);
  const payload = buildBaseSettingPatch(submittedSnapshot, draft.initialValue.value);
  if (Object.keys(payload).length === 0) {
    message.info('没有可保存的改动');
    return;
  }
  const formatErrors = validateBaseSettingPatchFormats(payload);
  if (formatErrors.length > 0) {
    const error = formatErrors[0]!;
    message.error(error.message);
    const entry = searchIndex.value.find(item => item.fieldKey === error.key);
    if (entry) {
      jumpToField(entry);
    }
    return;
  }
  await saveMutation.mutateAsync(payload);
  draft.commitSaved(submittedSnapshot);
  const refreshed = await valueQuery.refetch();
  if (refreshed.isSuccess && refreshed.data) {
    draft.syncRemote(refreshed.data);
  }
}

function resetChanges() {
  draft.resetToRemote();
}

function updateField(key: string, value: unknown) {
  if (!draft.currentValue.value) return;
  const next = {
    ...draft.currentValue.value,
    [key]: value,
  };
  if (key === 'officialQQUseMarkdown' && value === false) {
    next.officialQQUseNameCard = false;
  }
  draft.currentValue.value = next;
}

async function runAction(fieldId: string, payload?: unknown) {
  busyActionId.value = fieldId;
  try {
    if (fieldId === 'mail-test') {
      await mailTestMutation.mutateAsync();
      return;
    }
    if (fieldId === 'upgrade-package') {
      await upgradeMutation.mutateAsync(payload as File);
    }
  } finally {
    busyActionId.value = null;
  }
}

function toggleGroup(groupId: string) {
  expandedGroups[groupId] = !expandedGroups[groupId];
}

function findTab(tabId: string) {
  return tabs.value.find(tab => tab.id === tabId);
}

async function jumpToField(entry: BaseSettingSearchEntry) {
  activeTab.value = entry.tabId;
  expandedGroups[entry.groupId] = true;
  highlightedFieldId.value = entry.fieldId;
  await nextTick();
  const element = document.querySelector(`[data-field-id="${entry.fieldId}"]`);
  if (element instanceof HTMLElement) {
    element.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }
  window.setTimeout(() => {
    if (highlightedFieldId.value === entry.fieldId) {
      highlightedFieldId.value = '';
    }
  }, 1600);
}
</script>

<style scoped>
.base-setting-page {
  width: 100%;
  min-width: 0;
}

.setting-groups {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--sd-space-2xs);
  padding-bottom: var(--sd-space-xl);
}

.group-note {
  margin: var(--sd-space-sm) var(--sd-space-md) 0.35rem;
}

@media (max-width: 768px) {
  .setting-groups {
    gap: 0.15rem;
  }
}

@media (max-width: 639.9px) {
  .group-note {
    margin: var(--sd-space-xs) var(--sd-space-sm) 0.3rem;
  }
}
</style>
