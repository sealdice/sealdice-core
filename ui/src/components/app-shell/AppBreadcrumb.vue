<template>
  <n-page-header class="sd-breadcrumb-bar">
    <template #title>
      <div class="sd-breadcrumb-title">
        <!-- 桌面收起态用主题主色提示侧栏状态，移动端仍保持普通菜单按钮。 -->
        <n-button
          class="sd-sidebar-toggle"
          :class="{ 'sd-sidebar-toggle--collapsed': props.collapsed && !props.mobileMode }"
          size="small"
          quaternary
          circle
          :type="props.collapsed && !props.mobileMode ? 'primary' : 'default'"
          @click="emit('toggleSidebar')"
        >
          <template #icon>
            <n-icon size="1.2rem">
              <i-ep-menu v-if="props.mobileMode" />
              <i-ep-expand v-else-if="props.collapsed" />
              <i-ep-fold v-else />
            </n-icon>
          </template>
        </n-button>

        <n-breadcrumb>
          <n-breadcrumb-item
            v-for="(item, index) in visibleBreadcrumbItems"
            :key="`${index}-${item.label}`"
          >
            <RouterLink v-if="item.to" :to="item.to">
              {{ item.label }}
            </RouterLink>
            <span v-else>{{ item.label }}</span>
          </n-breadcrumb-item>
        </n-breadcrumb>
      </div>
    </template>

    <template #extra>
      <div class="sd-page-actions">
        <AppThemeSwitch />
        <AppInstallButton v-if="!props.mobileMode" />
        <n-button tag="a" secondary class="legacy-entry" :href="oldUIUrl">
          回退老 UI
        </n-button>

        <button type="button" class="search-entry" @click="emit('openSearch')">
          <span class="search-label">
            <n-icon size="1.1rem">
              <i-ep-search />
            </n-icon>
            <span>搜索</span>
          </span>
          <span class="search-shortcut">Ctrl k</span>
        </button>

        <div class="version-summary">
          <n-tag
            :bordered="false"
            :type="isStable ? 'success' : 'default'"
            size="small"
            class="version-channel"
          >
            {{ isStable ? '正式版' : '测试版' }}
          </n-tag>
          <n-tooltip placement="bottom">
            <template #trigger>
              <span class="version-text">{{ overview?.version.simple ?? '-' }}</span>
            </template>
            {{ overview?.version.value ?? '-' }}
          </n-tooltip>
          <span v-if="hasNewVersion" class="new-version">
            新版本 {{ overview?.version.latest }}
          </span>
        </div>
      </div>
    </template>
  </n-page-header>

</template>

<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import { useBaseOverview } from '@/features/base/useBaseOverview';
import { resolveOldUIUrlFromLocation } from '@/api/config';
import { appNavigation } from '@/router/navigation';
import { buildBreadcrumbItems } from '@/router/navigationModel';
import AppInstallButton from './AppInstallButton.vue';
import AppThemeSwitch from './AppThemeSwitch.vue';

const props = defineProps<{
  collapsed: boolean;
  mobileMode: boolean;
}>();

const emit = defineEmits<{
  toggleSidebar: [];
  openSearch: [];
}>();

const route = useRoute();
const { overview, isStable, hasNewVersion } = useBaseOverview();
const oldUIUrl =
  typeof window !== 'undefined' ? resolveOldUIUrlFromLocation(window.location) : '/old-ui/';

const breadcrumbItems = computed(() =>
  buildBreadcrumbItems(appNavigation, route.path, String(route.meta.title ?? '当前页面')),
);
const visibleBreadcrumbItems = computed(() =>
  props.mobileMode ? breadcrumbItems.value.slice(0, -1) : breadcrumbItems.value,
);
</script>

<style scoped>
.sd-breadcrumb-bar {
  position: sticky;
  top: 0;
  z-index: 10;
  border-bottom: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated);
  padding: 0.65rem 1rem;
}

.sd-breadcrumb-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.5rem;
}

.sd-sidebar-toggle {
  flex: 0 0 auto;
}

.sd-sidebar-toggle--collapsed {
  background: var(--sd-bg-selected);
  color: var(--sd-primary);
}

.sd-sidebar-toggle--collapsed:hover {
  background: var(--sd-bg-selected-strong);
  color: var(--sd-primary);
}

.version-summary {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.35rem;
  margin-left: 0.25rem;
}

.version-channel {
  font-size: 0.75rem;
}

.version-text {
  color: var(--sd-text-secondary);
  cursor: default;
  font-size: 0.82rem;
  line-height: 1;
}

.new-version {
  color: var(--sd-accent-strong);
  font-size: 0.78rem;
  line-height: 1;
  white-space: nowrap;
}

:deep(.n-page-header__title) {
  width: 100%;
}

.sd-page-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.legacy-entry {
  white-space: nowrap;
}

.search-entry {
  display: inline-flex;
  height: 34px;
  align-items: center;
  border: 0;
  border-radius: 6px;
  background: var(--sd-bg-control);
  color: var(--sd-text-secondary);
  cursor: pointer;
  font: inherit;
  gap: 0.55rem;
  line-height: 1;
  padding: 0 0.5rem 0 0.7rem;
}

.search-entry:hover {
  background: var(--sd-bg-control-hover);
}

.search-label {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.search-shortcut {
  display: inline-flex;
  height: 22px;
  align-items: center;
  border-radius: 6px;
  background: var(--sd-bg-elevated);
  color: var(--sd-text-muted);
  font-size: 0.75rem;
  line-height: 1;
  padding: 0 0.45rem;
  white-space: nowrap;
}

.feed-content {
  margin-top: 10px;
  margin-bottom: 20px;
  text-align: left;
}

:deep(.n-breadcrumb) {
  min-width: 0;
}

:deep(.n-breadcrumb-item__link) {
  color: inherit;
  text-decoration: none;
}

@media screen and (max-width: 639.9px) {
  :deep(.n-page-header__main) {
    flex: 1 1 auto;
    min-width: 0;
  }

  :deep(.n-page-header__title),
  :deep(.n-page-header__extra) {
    min-width: 0;
  }

  .sd-breadcrumb-title {
    min-width: 0;
    overflow: hidden;
  }

  :deep(.n-breadcrumb) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sd-page-actions {
    flex-shrink: 0;
    flex-wrap: wrap;
    gap: 0.375rem;
  }

  .search-entry {
    flex: 0 0 34px;
    width: 34px;
    justify-content: center;
    padding: 0;
  }

  .search-label span,
  .search-shortcut {
    display: none;
  }

  .version-summary {
    display: flex;
    margin-left: auto;
  }

  .version-channel,
  .new-version {
    display: none;
  }
}
</style>
