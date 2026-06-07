<template>
  <div class="ext-defaults-field">
    <div class="ext-defaults-toolbar">
      <n-input
        v-model:value="keyword"
        clearable
        placeholder="搜索扩展或指令"
        class="ext-defaults-search"
      >
        <template #prefix>
          <i-carbon-search />
        </template>
      </n-input>

      <n-select
        v-model:value="sortKey"
        :options="sortOptions"
        class="ext-defaults-sort"
      />

      <n-radio-group v-model:value="filterMode" size="small" class="ext-defaults-filter">
        <n-radio-button
          v-for="option in filterOptions"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </n-radio-button>
      </n-radio-group>
    </div>

    <div class="ext-defaults-summary">
      <n-text depth="3">共 {{ viewItems.length }} 项，已修改 {{ modifiedCount }} 项</n-text>
    </div>

    <div v-if="pagedItems.items.length > 0" class="ext-defaults-list">
      <section
        v-for="entry in pagedItems.items"
        :key="entry.item.name"
        :class="['ext-default-row', { 'ext-default-row-dirty': entry.dirty }]"
      >
        <div class="ext-default-row-head">
          <div class="ext-default-row-title">
            <span class="ext-default-row-name">{{ entry.item.name }}</span>
            <n-tag v-if="entry.dirty" size="small" type="warning" round :bordered="false">
              已修改
            </n-tag>
          </div>

          <div :class="['ext-default-row-switch', { 'ext-default-row-switch-dirty': entry.autoActiveDirty }]">
            <span class="ext-default-row-switch-label">入群自动开启</span>
            <n-switch
              :value="entry.item.autoActive"
              size="small"
              @update:value="updateAutoActive(entry.item.name, $event)"
            />
          </div>
        </div>

        <div class="ext-default-row-meta">
          <n-text depth="3">禁用 {{ entry.disabledCount }} / {{ entry.commandCount }} 条指令</n-text>
          <n-text v-if="entry.changedCommands.length" depth="3">
            变更 {{ entry.changedCommands.length }} 项
          </n-text>
        </div>

        <div class="ext-default-row-commands">
          <button
            v-for="[command, disabled] in getCommandEntries(entry.item)"
            :key="command"
            type="button"
            :class="[
              'ext-default-command-chip',
              {
                'ext-default-command-chip-disabled': disabled,
                'ext-default-command-chip-dirty': entry.changedCommands.includes(command),
              },
            ]"
            @click="toggleDisabledCommand(entry.item.name, command)"
          >
            {{ command }}
          </button>
        </div>
      </section>
    </div>

    <n-empty v-else :description="emptyDescription" size="small" class="ext-defaults-empty" />

    <div v-if="viewItems.length > 0" class="ext-defaults-footer">
      <n-text depth="3">第 {{ pagedItems.page }} / {{ pagedItems.pageCount }} 页</n-text>
      <n-pagination
        v-model:page="page"
        v-model:page-size="pageSize"
        show-size-picker
        show-quick-jumper
        :page-sizes="[10, 20, 50]"
        :page-slot="5"
        :item-count="pagedItems.total"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { BaseSettingExtDefaultSettingItem } from '@/api';
import {
  buildExtDefaultSettingsView,
  filterExtDefaultSettingsView,
  getExtDefaultSettingModifiedCount,
  getExtDefaultSettingPage,
  searchExtDefaultSettingsView,
  sortExtDefaultSettingsView,
  type ExtDefaultSettingsFilterMode,
  type ExtDefaultSettingsSortKey,
} from '@/features/baseSetting/viewModel';

const props = withDefaults(defineProps<{
  initialItems?: BaseSettingExtDefaultSettingItem[];
}>(), {
  initialItems: () => [],
});

const model = defineModel<BaseSettingExtDefaultSettingItem[]>({ required: true });

const keyword = ref('');
const filterMode = ref<ExtDefaultSettingsFilterMode>('all');
const sortKey = ref<ExtDefaultSettingsSortKey>('source');
const page = ref(1);
const pageSize = ref(10);

const viewItems = computed(() => buildExtDefaultSettingsView(model.value, props.initialItems));
const modifiedCount = computed(() => getExtDefaultSettingModifiedCount(viewItems.value));
const filteredByKeyword = computed(() => searchExtDefaultSettingsView(viewItems.value, keyword.value));
const filteredItems = computed(() => filterExtDefaultSettingsView(filteredByKeyword.value, filterMode.value));
const sortedItems = computed(() => sortExtDefaultSettingsView(filteredItems.value, sortKey.value));
const pagedItems = computed(() => getExtDefaultSettingPage(sortedItems.value, page.value, pageSize.value));

const filterOptions = computed(() => [
  { label: '全部', value: 'all' },
  { label: `已修改 ${modifiedCount.value}`, value: 'modified' },
]);

const sortOptions: Array<{ label: string; value: ExtDefaultSettingsSortKey }> = [
  { label: '原始顺序', value: 'source' },
  { label: '修改优先', value: 'modified' },
  { label: '扩展名', value: 'name' },
  { label: '自动开启优先', value: 'auto-active' },
  { label: '禁用指令数', value: 'disabled-count' },
];

const emptyDescription = computed(() => {
  if (viewItems.value.length === 0) return '暂无扩展默认设置';
  if (filterMode.value === 'modified') return '当前没有已修改的扩展';
  if (keyword.value.trim()) return '没有匹配的扩展或指令';
  return '当前没有可显示的扩展';
});

watch([keyword, filterMode, sortKey, pageSize], () => {
  page.value = 1;
});

watch(
  () => pagedItems.value.page,
  nextPage => {
    if (page.value !== nextPage) {
      page.value = nextPage;
    }
  },
);

function updateItem(name: string, updater: (item: BaseSettingExtDefaultSettingItem) => void) {
  const targetIndex = model.value.findIndex(item => item.name === name);
  if (targetIndex < 0) return;
  const next = structuredClone(model.value);
  updater(next[targetIndex]!);
  model.value = next;
}

function toggleDisabledCommand(name: string, command: string) {
  updateItem(name, item => {
    item.disabledCommand[command] = !item.disabledCommand[command];
  });
}

function updateAutoActive(name: string, value: boolean) {
  updateItem(name, item => {
    item.autoActive = value;
  });
}

function getCommandEntries(item: BaseSettingExtDefaultSettingItem) {
  return Object.entries(item.disabledCommand ?? {}).sort(([left], [right]) => left.localeCompare(right));
}
</script>

<style scoped>
.ext-defaults-field {
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}

.ext-defaults-toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 12rem auto;
  gap: 0.65rem;
  align-items: center;
}

.ext-defaults-search,
.ext-defaults-sort {
  min-width: 0;
}

.ext-defaults-filter {
  justify-self: start;
}

.ext-defaults-summary,
.ext-defaults-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.ext-defaults-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.ext-default-row {
  display: flex;
  flex-direction: column;
  gap: 0.7rem;
  padding: 0.85rem 0.95rem;
  border: 1px solid var(--sd-border-soft);
  border: 1px solid color-mix(in srgb, var(--sd-border-color), transparent 10%);
  border-radius: 8px;
  background: var(--sd-bg-elevated-soft);
  background: color-mix(in srgb, var(--sd-bg-page), var(--sd-bg-elevated) 72%);
  transition: border-color 0.16s ease, background-color 0.16s ease, box-shadow 0.16s ease;
}

.ext-default-row:hover {
  border-color: var(--sd-primary);
  border-color: color-mix(in srgb, var(--sd-primary-color), var(--sd-border-color) 70%);
}

.ext-default-row-dirty {
  border-color: var(--n-warning-color);
  border-color: color-mix(in srgb, var(--n-warning-color), var(--sd-border-color) 55%);
  box-shadow: inset 0 0 0 1px rgba(202, 138, 4, 0.18);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--n-warning-color), transparent 82%);
}

.ext-default-row-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.9rem;
  flex-wrap: wrap;
}

.ext-default-row-title {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  flex-wrap: wrap;
  min-width: 0;
}

.ext-default-row-name {
  color: var(--sd-text-primary);
  font-size: 0.94rem;
  font-weight: 600;
  line-height: 1.35;
}

.ext-default-row-switch {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  min-height: 1.75rem;
  padding: 0.18rem 0.22rem 0.18rem 0.55rem;
  border-radius: 999px;
  background: var(--sd-bg-hover);
  background: color-mix(in srgb, var(--sd-bg-hover), transparent 24%);
}

.ext-default-row-switch-dirty {
  background: rgba(202, 138, 4, 0.12);
  background: color-mix(in srgb, var(--n-warning-color), transparent 88%);
}

.ext-default-row-switch-label {
  color: var(--sd-text-secondary);
  font-size: 0.78rem;
  line-height: 1.2;
  white-space: nowrap;
}

.ext-default-row-meta {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  flex-wrap: wrap;
}

.ext-default-row-commands {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.ext-default-command-chip {
  appearance: none;
  cursor: pointer;
  padding: 0.36rem 0.66rem;
  border: 1px solid var(--sd-border-soft);
  border: 1px solid color-mix(in srgb, var(--sd-border-color), transparent 8%);
  border-radius: 999px;
  background: var(--sd-bg-elevated);
  color: var(--sd-text-secondary);
  font: inherit;
  font-size: 0.78rem;
  line-height: 1.25;
  transition: border-color 0.16s ease, background-color 0.16s ease, color 0.16s ease;
}

.ext-default-command-chip:hover {
  border-color: var(--sd-primary);
  border-color: color-mix(in srgb, var(--sd-primary-color), var(--sd-border-color) 55%);
  color: var(--sd-text-primary);
}

.ext-default-command-chip-disabled {
  border-color: var(--n-error-color);
  border-color: color-mix(in srgb, var(--n-error-color), transparent 52%);
  background: rgba(220, 38, 38, 0.08);
  background: color-mix(in srgb, var(--n-error-color), transparent 92%);
  color: var(--n-error-color);
}

.ext-default-command-chip-dirty {
  box-shadow: inset 0 0 0 1px rgba(202, 138, 4, 0.28);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--n-warning-color), transparent 72%);
}

.ext-defaults-empty {
  padding: 1rem 0;
}

@media (max-width: 860px) {
  .ext-defaults-toolbar {
    grid-template-columns: minmax(0, 1fr);
  }

  .ext-defaults-filter {
    justify-self: stretch;
  }
}

@media (max-width: 640px) {
  .ext-defaults-field {
    gap: 0.6rem;
  }

  .ext-default-row {
    padding: 0.75rem;
  }

  .ext-default-row-head {
    gap: 0.6rem;
  }

  .ext-default-row-switch {
    width: 100%;
    justify-content: space-between;
  }

  .ext-defaults-footer :deep(.n-pagination) {
    width: 100%;
  }
}
</style>
