<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue';
import { useClipboard, useLocalStorage } from '@vueuse/core';
import { useMessage } from 'naive-ui';
import { useRouter } from 'vue-router';
import {
  addSearchHistory,
  matchesNavigationSearch,
  removeSearchHistoryItem,
} from '@/router/navigationModel';
import type { NavigationSearchItem } from '@/router/types';
import { useAppNavigation } from '@/router/useAppNavigation';
import AppNavigationIcon from './AppNavigationIcon.vue';

const props = withDefaults(
  defineProps<{
    advancedConfigCounter?: number;
  }>(),
  {
    advancedConfigCounter: 0,
  },
);

const show = ref(false);
const keyword = ref('');
const selectedIndex = ref(-1);
const inputRef = ref<{ focus: () => void } | null>(null);
const history = useLocalStorage<NavigationSearchItem[]>('search-history', []);
const router = useRouter();
const message = useMessage();
const { copy } = useClipboard();
const { searchItems } = useAppNavigation(() => props.advancedConfigCounter);

const trimmedKeyword = computed(() => keyword.value.trim());
const results = computed(() => {
  if (!trimmedKeyword.value) return [];
  return searchItems.value.filter(item => matchesNavigationSearch(item, trimmedKeyword.value));
});
const visibleItems = computed(() => (trimmedKeyword.value ? results.value : history.value));
const hasList = computed(() => visibleItems.value.length > 0);

watch(visibleItems, items => {
  selectedIndex.value = items.length ? Math.min(Math.max(selectedIndex.value, 0), items.length - 1) : -1;
});

function open() {
  show.value = true;
  selectedIndex.value = visibleItems.value.length ? 0 : -1;
  nextTick(() => inputRef.value?.focus());
}

function close() {
  show.value = false;
  keyword.value = '';
  selectedIndex.value = -1;
}

function resolvedHref(path: string) {
  return new URL(router.resolve(path).href, window.location.href).toString();
}

async function copyLink(item: NavigationSearchItem, event?: MouseEvent) {
  event?.stopPropagation();
  await copy(resolvedHref(item.path));
  message.success('复制成功');
}

function openInNewWindow(item: NavigationSearchItem, event?: MouseEvent) {
  event?.stopPropagation();
  window.open(resolvedHref(item.path), '_blank', 'noopener,noreferrer');
}

function removeHistory(item: NavigationSearchItem, event?: MouseEvent) {
  event?.stopPropagation();
  history.value = removeSearchHistoryItem(history.value, item.path);
}

async function selectItem(item = visibleItems.value[selectedIndex.value]) {
  if (!item) return;
  history.value = addSearchHistory(history.value, item);
  close();
  await router.push(item.path);
}

function moveSelection(delta: number) {
  if (!visibleItems.value.length) return;
  selectedIndex.value =
    (selectedIndex.value + delta + visibleItems.value.length) % visibleItems.value.length;
}

function onMouseEnter(index: number) {
  selectedIndex.value = index;
}

defineExpose({ open });
</script>

<template>
  <n-modal
    v-model:show="show"
    :auto-focus="false"
    :mask-closable="true"
    class="sd-search-modal"
    transform-origin="center"
  >
    <n-card
        class="sd-search-card"
        :bordered="false"
        content-style="padding: 0"
        header-style="padding: 0"
        footer-style="padding: 0"
        role="dialog"
        aria-modal="true"
        @keydown.esc.prevent="close"
        @keydown.arrow-up.prevent="moveSelection(-1)"
        @keydown.arrow-down.prevent="moveSelection(1)"
        @keydown.enter.prevent="selectItem()"
      >
      <template #header>
        <div class="sd-search-header">
          <n-input
            ref="inputRef"
            v-model:value="keyword"
            class="sd-search-input"
            size="large"
            clearable
            autofocus
            placeholder="输入内容、支持按首字母搜索"
          >
            <template #prefix>
              <n-icon>
                <i-carbon-search />
              </n-icon>
            </template>
            <template #suffix>
              <n-button circle quaternary size="small" @click="close">
                <template #icon>
                  <n-icon>
                    <i-carbon-close />
                  </n-icon>
                </template>
              </n-button>
            </template>
          </n-input>
        </div>
      </template>

      <div class="sd-search-body">
        <n-scrollbar v-if="hasList" class="sd-search-scroll">
          <div v-if="!trimmedKeyword && history.length" class="sd-search-section-title">
            搜索历史
          </div>

          <button
            v-for="(item, index) in visibleItems"
            :key="item.path"
            type="button"
            class="sd-search-item"
            :class="{ selected: index === selectedIndex }"
            @click="selectItem(item)"
            @mouseenter="onMouseEnter(index)"
          >
            <span class="sd-search-item-main">
              <n-icon class="sd-search-item-icon">
                <AppNavigationIcon :name="item.icon" />
              </n-icon>
              <span class="sd-search-item-text">
                <span class="sd-search-item-label">{{ item.label }}</span>
                <span class="sd-search-item-path">{{ item.path }}</span>
              </span>
            </span>

            <span class="sd-search-actions">
              <n-tooltip>
                <template #trigger>
                  <n-button circle quaternary size="small" @click="copyLink(item, $event)">
                    <template #icon>
                      <n-icon>
                        <i-carbon-link />
                      </n-icon>
                    </template>
                  </n-button>
                </template>
                复制链接
              </n-tooltip>

              <n-tooltip>
                <template #trigger>
                  <n-button circle quaternary size="small" @click="openInNewWindow(item, $event)">
                    <template #icon>
                      <n-icon>
                        <i-carbon-launch />
                      </n-icon>
                    </template>
                  </n-button>
                </template>
                新窗口打开
              </n-tooltip>

              <n-tooltip v-if="!trimmedKeyword">
                <template #trigger>
                  <n-button circle quaternary size="small" @click="removeHistory(item, $event)">
                    <template #icon>
                      <n-icon>
                        <i-carbon-close />
                      </n-icon>
                    </template>
                  </n-button>
                </template>
                删除记录
              </n-tooltip>
            </span>
          </button>
        </n-scrollbar>

        <div v-else class="sd-search-empty">
          <n-icon size="38" class="sd-search-empty-icon">
            <i-carbon-search-locate />
          </n-icon>
          <n-text depth="3">
            <template v-if="trimmedKeyword">
              没有找到 <strong>“{{ trimmedKeyword }}”</strong> 的结果
            </template>
            <template v-else>
              输入搜索关键字开始搜索吧~
            </template>
          </n-text>
        </div>
      </div>

      <template #footer>
        <div class="sd-search-footer">
          <span class="sd-key">Enter</span><n-text depth="3">选择</n-text>
          <n-divider vertical />
          <span class="sd-key">↗</span><n-text depth="3">新窗口</n-text>
          <n-divider vertical />
          <span class="sd-key">↑</span><span class="sd-key">↓</span><n-text depth="3">切换</n-text>
          <n-divider vertical />
          <span class="sd-key">Esc</span><n-text depth="3">退出</n-text>
        </div>
      </template>
    </n-card>
  </n-modal>
</template>

<style scoped>
.sd-search-modal {
  width: min(600px, calc(100vw - 32px));
  margin-top: 100px;
}

.sd-search-card {
  overflow: hidden;
}

.sd-search-header {
  padding: 0.75rem 0.75rem 0.5rem;
}

:deep(.sd-search-input .n-input__border),
:deep(.sd-search-input .n-input__state-border) {
  display: none;
}

.sd-search-body {
  min-height: 180px;
  padding: 0 1.25rem;
}

.sd-search-scroll {
  max-height: 420px;
}

.sd-search-section-title {
  color: var(--sd-text-muted);
  font-size: 0.85rem;
  padding: 0.5rem 0.25rem;
}

.sd-search-item {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  border: 1px dashed transparent;
  border-bottom-color: var(--sd-border);
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  gap: 0.75rem;
  margin-bottom: 5px;
  padding: 0.45rem 0.75rem;
  text-align: left;
}

.sd-search-item:hover,
.sd-search-item.selected {
  border-color: var(--sd-primary);
  background: var(--sd-bg-selected);
}

.sd-search-item-main {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.45rem;
}

.sd-search-item-icon {
  flex: 0 0 auto;
  font-size: 18px;
}

.sd-search-item-text {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.sd-search-item-label {
  color: var(--sd-text-primary);
  font-size: 1rem;
  line-height: 1.25;
}

.sd-search-item-path {
  overflow: hidden;
  color: var(--sd-text-muted);
  font-size: 0.8rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sd-search-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.25rem;
}

.sd-search-empty {
  display: flex;
  min-height: 180px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 0;
  text-align: center;
}

.sd-search-empty-icon {
  margin-bottom: 0.5rem;
  opacity: 0.3;
}

.sd-search-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
  padding: 0.75rem 1rem 1rem;
}

.sd-key {
  display: inline-flex;
  min-width: 24px;
  height: 22px;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  background: var(--sd-bg-hover);
  color: var(--sd-text-muted);
  font-size: 0.75rem;
  line-height: 1;
  padding: 0 0.35rem;
}
</style>
