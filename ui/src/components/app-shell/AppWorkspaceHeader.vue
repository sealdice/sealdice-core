<template>
  <header class="workspace-header" :class="{ 'workspace-header--compact': isCompact }">
    <div class="workspace-brand">
      <n-button
        quaternary
        size="small"
        :aria-label="toggleLabel"
        :aria-expanded="props.navigationExpanded"
        :title="toggleLabel"
        @click="emit('toggleSidebar')"
      >
        <template #icon>
          <n-icon size="1.2rem">
            <i-tabler-menu-2 v-if="isMobile" />
            <i-tabler-layout-sidebar-left-collapse v-else-if="props.navigationExpanded" />
            <i-tabler-layout-sidebar-left-expand v-else />
          </n-icon>
        </template>
      </n-button>
      <button
        class="brand-title"
        type="button"
        :aria-label="appName + '，启用高级配置'"
        @click="emit('enableAdvancedConfig')"
      >
        <img :src="sealdiceD20" alt="" />
        <span>{{ appName }}</span>
      </button>
    </div>

    <div class="workspace-actions">
      <button
        class="workspace-search"
        type="button"
        aria-label="搜索页面或功能"
        aria-keyshortcuts="Control+K Meta+K"
        @click="emit('openSearch')"
      >
        <n-icon size="1rem"><i-tabler-search /></n-icon>
        <span>搜索页面</span>
        <kbd>Ctrl K</kbd>
      </button>

      <span
        class="workspace-status"
        :class="{ 'workspace-status--online': hasAccessToken && connected }"
        role="status"
        :title="versionHint"
      >
        <span class="workspace-status__dot" aria-hidden="true" />
        <span class="workspace-status__label">{{ connectionLabel }}</span>
      </span>

      <AppUploadIndicator />
      <AppThemeSwitch />
      <AppInstallButton v-if="!isCompact" />
      <AppHeaderOverflowMenu :show-install="isCompact" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import sealdiceD20 from '@/assets/sealdice-d20.svg';
import { useAuthStore } from '@/features/auth/store';
import { useBaseOverview } from '@/features/base/useBaseOverview';
import { useRealtimeClientStore } from '@/features/realtime/store';
import AppHeaderOverflowMenu from './AppHeaderOverflowMenu.vue';
import AppInstallButton from './AppInstallButton.vue';
import AppThemeSwitch from './AppThemeSwitch.vue';
import AppUploadIndicator from './AppUploadIndicator.vue';
import type { AppShellViewportMode } from './appShellLayout';

const props = defineProps<{
  navigationExpanded: boolean;
  viewportMode: AppShellViewportMode;
}>();

const emit = defineEmits<{
  toggleSidebar: [];
  openSearch: [];
  enableAdvancedConfig: [];
}>();

const { appName, overview, displayVersion } = useBaseOverview();
const { hasAccessToken, isInitializing } = storeToRefs(useAuthStore());
const { connected, connecting } = storeToRefs(useRealtimeClientStore());
const isMobile = computed(() => props.viewportMode === 'mobile');
const isCompact = computed(() => props.viewportMode !== 'desktop');
const toggleLabel = computed(() => {
  if (isMobile.value) return props.navigationExpanded ? '关闭导航' : '打开导航';
  return props.navigationExpanded ? '收起导航' : '展开导航';
});
const connectionLabel = computed(() => {
  if (isInitializing.value) return '正在连接';
  if (!hasAccessToken.value) return '未登录';
  if (connected.value) return '已连接';
  return connecting.value ? '正在连接' : '连接断开';
});
const versionHint = computed(() => {
  if (!overview.value?.version) return '核心版本未知';
  const mode = overview.value.runtime?.containerMode ? '・容器模式' : '';
  return `核心版本 ${displayVersion.value}${mode}`;
});
</script>

<style scoped>
.workspace-header {
  z-index: 10;
  display: flex;
  min-height: var(--sd-workspace-header-height);
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: var(--sd-space-md);
  padding-inline: var(--sd-space-sm) var(--sd-space-lg);
  border-bottom: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated);
}

.workspace-brand,
.workspace-actions {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--sd-space-sm);
}

.workspace-actions {
  flex: 0 0 auto;
  gap: var(--sd-space-xs);
}

.brand-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: var(--sd-space-xs);
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--sd-text-primary);
  font: inherit;
  font-size: 1.125rem;
  font-weight: 650;
  cursor: pointer;
}

.brand-title img {
  width: 26px;
  height: 26px;
  flex: 0 0 auto;
}

.brand-title span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-search {
  display: flex;
  min-height: 32px;
  align-items: center;
  gap: var(--sd-space-xs);
  padding-inline: var(--sd-space-xs);
  border: 1px solid var(--sd-border-soft);
  border-radius: var(--sd-radius-sm);
  background: var(--sd-bg-elevated-soft);
  color: var(--sd-text-secondary);
  font: inherit;
  font-size: 0.8rem;
  cursor: pointer;
}

.workspace-search:hover {
  border-color: var(--sd-border);
  background: var(--sd-bg-control-hover);
  color: var(--sd-text-primary);
}

.workspace-search kbd {
  margin-inline-start: var(--sd-space-md);
  font-size: 0.6875rem;
  white-space: nowrap;
}

.brand-title:focus-visible,
.workspace-search:focus-visible {
  outline: 2px solid var(--sd-primary);
  outline-offset: 2px;
}

.workspace-status {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding-inline: var(--sd-space-xs);
  color: var(--sd-text-secondary);
  font-size: 0.75rem;
  white-space: nowrap;
}

.workspace-status__dot {
  width: 0.375rem;
  height: 0.375rem;
  border-radius: 50%;
  background: var(--sd-text-muted);
}

.workspace-status--online .workspace-status__dot {
  background: var(--sd-success);
}

.workspace-header--compact {
  gap: var(--sd-space-xs);
  padding-inline: var(--sd-space-xs);
}

.workspace-header--compact .workspace-brand,
.workspace-header--compact .workspace-actions {
  gap: var(--sd-space-2xs);
}

.workspace-header--compact .workspace-search {
  width: 32px;
  justify-content: center;
  padding: 0;
}

.workspace-header--compact .workspace-search span,
.workspace-header--compact .workspace-search kbd {
  display: none;
}

@media (max-width: 480px) {
  .brand-title {
    font-size: 1rem;
  }

  .brand-title img {
    display: none;
  }

  .workspace-status__label {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
    white-space: nowrap;
  }
}
</style>
