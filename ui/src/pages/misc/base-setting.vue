<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue';
import { useQueryClient } from '@tanstack/vue-query';
import BaseSettingFieldRenderer from '@/components/base-setting/BaseSettingFieldRenderer.vue';
import BaseSettingSearchBar from '@/components/base-setting/BaseSettingSearchBar.vue';
import TipBox from '@/components/shared/TipBox.vue';
import { useBaseOverview } from '@/features/base/useBaseOverview';
import { useBaseSettingDraft } from '@/features/baseSetting/draft';
import {
  prepareBaseSettingSavePayload,
  useBaseSettingMutations,
} from '@/features/baseSetting/mutations';
import {
  useBaseSettingSchemaQuery,
  useBaseSettingValueQuery,
} from '@/features/baseSetting/queries';
import {
  buildBaseSettingPatch,
  buildBaseSettingSearchIndex,
  searchBaseSettingFields,
  type BaseSettingSearchEntry,
} from '@/features/baseSetting/viewModel';
import { useUnsavedChanges } from '@/features/unsavedChanges';

const message = useMessage();
const queryClient = useQueryClient();
const { overview } = useBaseOverview();

const activeTab = ref('master-notice');
const searchKeyword = ref('');
const expandedGroups = reactive<Record<string, boolean>>({});
const highlightedFieldId = ref('');
const busyActionId = ref<string | null>(null);

const schemaQuery = useBaseSettingSchemaQuery();
const valueQuery = useBaseSettingValueQuery();
const draft = useBaseSettingDraft();

watch(
  () => schemaQuery.data.value,
  schema => {
    if (!schema) return;
    for (const tab of schema.tabs) {
      for (const group of tab.groups) {
        expandedGroups[group.id] = group.collapsible ? Boolean(group.defaultExpanded) : true;
      }
    }
  },
  { immediate: true },
);

watch(
  () => valueQuery.data.value,
  value => {
    if (!value) return;
    draft.syncRemote(value);
  },
  { immediate: true },
);

const tabs = computed(() => schemaQuery.data.value?.tabs ?? []);
const searchIndex = computed(() => (schemaQuery.data.value ? buildBaseSettingSearchIndex(schemaQuery.data.value) : []));
const searchResults = computed(() => searchBaseSettingFields(searchIndex.value, searchKeyword.value));
const currentValue = computed(() => draft.currentValue.value);
const pageBusy = computed(() => schemaQuery.isFetching.value || valueQuery.isFetching.value);
const isContainerMode = computed(() => overview.value?.runtime.containerMode === true);

const { saveMutation, mailTestMutation, upgradeMutation } = useBaseSettingMutations({
  queryClient,
  message,
  onSaved: () => {},
});

useUnsavedChanges('base-setting', {
  label: '基本设置',
  dirty: computed(() => draft.dirty.value),
  save: saveChanges,
  saving: computed(() => saveMutation.isPending.value),
  canSave: computed(() => draft.dirty.value),
  confirmMessage: '基本设置还有修改，确定要忽略？',
});

async function saveChanges() {
  if (!draft.currentValue.value || !draft.initialValue.value) return;
  const payload = await prepareBaseSettingSavePayload(
    draft.currentValue.value,
    draft.initialValue.value,
    buildBaseSettingPatch,
  );
  if (Object.keys(payload).length === 0) {
    message.info('没有可保存的改动');
    return;
  }
  await saveMutation.mutateAsync(payload);
  const refreshed = await valueQuery.refetch();
  if (refreshed.data) {
    draft.syncRemote(refreshed.data, true);
  }
}

function resetChanges() {
  draft.resetToRemote();
}

function updateField(key: string, value: unknown) {
  if (!draft.currentValue.value) return;
  draft.currentValue.value = {
    ...draft.currentValue.value,
    [key]: value,
  };
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

async function jumpToField(entry: BaseSettingSearchEntry) {
  activeTab.value = entry.tabId;
  expandedGroups[entry.groupId] = true;
  highlightedFieldId.value = entry.fieldId;
  await nextTick();
  const element = document.querySelector(`[data-field-id="${entry.fieldId}"]`);
  if (element instanceof HTMLElement) {
    element.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }
}
</script>

<template>
  <main class="base-setting-page">
    <header class="page-head">
      <div class="page-head-copy">
        <h1>基本设置</h1>
        <p>按业务分栏管理海豹基础配置，并支持跨分栏搜索与定位。</p>
      </div>
      <n-flex>
        <n-button secondary :disabled="!draft.dirty.value" @click="resetChanges">
          放弃改动
        </n-button>
        <n-button type="primary" :loading="saveMutation.isPending.value" :disabled="!draft.dirty.value" @click="saveChanges">
          保存设置
        </n-button>
      </n-flex>
    </header>

    <BaseSettingSearchBar
      v-model:keyword="searchKeyword"
      :results="searchResults"
      @select="jumpToField"
    />

    <n-spin :show="pageBusy">
      <n-tabs v-model:value="activeTab" type="line" animated class="setting-tabs">
        <n-tab-pane
          v-for="tab in tabs"
          :key="tab.id"
          :name="tab.id"
          :tab="tab.title"
        >
          <div class="setting-groups">
            <section
              v-for="group in tab.groups"
              :key="group.id"
              :class="['setting-group-card', { 'setting-group-wide': ['ext-default-settings', 'upgrade', 'rate-limit-main'].includes(group.id) }]"
            >
              <header class="setting-group-head">
                <div>
                  <h3>{{ group.title }}</h3>
                  <p v-if="group.description">{{ group.description }}</p>
                </div>
                <n-button
                  v-if="group.collapsible"
                  text
                  size="small"
                  @click="toggleGroup(group.id)"
                >
                  {{ expandedGroups[group.id] ? '收起' : '展开' }}
                </n-button>
              </header>

              <template v-if="expandedGroups[group.id]">
                <TipBox
                  v-for="(note, noteIndex) in group.notes"
                  :key="`${group.id}-${noteIndex}`"
                  :type="note.tone === 'warning' ? 'warning' : 'info'"
                  class="group-note"
                >
                  <div v-for="(line, lineIndex) in note.lines" :key="lineIndex">
                    {{ line }}
                  </div>
                </TipBox>

                <n-form v-if="currentValue" label-placement="left" label-width="126">
                  <div class="group-fields">
                    <BaseSettingFieldRenderer
                      v-for="field in group.fields"
                      :key="field.id"
                      :field="field"
                      :model="currentValue"
                      :is-container-mode="isContainerMode"
                      :busy-action-id="busyActionId"
                      :run-action="runAction"
                      @update-field="updateField"
                    />
                  </div>
                </n-form>
              </template>
            </section>
          </div>
        </n-tab-pane>
      </n-tabs>
    </n-spin>
  </main>
</template>

<style scoped>
.base-setting-page {
  width: 100%;
}

.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.page-head-copy h1 {
  margin: 0;
  font-size: 1.6rem;
}

.page-head-copy p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
}

.setting-tabs {
  margin-top: 1rem;
}

.setting-groups {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 1rem;
}

.setting-group-card {
  border: 1px solid var(--sd-border);
  border-radius: 18px;
  background: var(--sd-bg-elevated);
  padding: 1rem;
}

.setting-group-wide {
  grid-column: 1 / -1;
}

.setting-group-head {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  margin-bottom: 0.75rem;
}

.setting-group-head h3 {
  margin: 0;
  font-size: 1rem;
}

.setting-group-head p {
  margin: 0.35rem 0 0;
  color: var(--sd-text-muted);
  font-size: 0.85rem;
}

.group-note {
  margin-bottom: 0.75rem;
}

.group-fields {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 0 1rem;
}

@media (max-width: 768px) {
  .setting-groups,
  .group-fields {
    grid-template-columns: 1fr;
  }
}
</style>
