<template>
  <n-page-header class="sd-breadcrumb-bar" :class="{ 'sd-breadcrumb-bar--compact': isCompactMode }">
    <template #title>
      <div class="sd-breadcrumb-title">
        <!-- 桌面收起态用主题主色提示侧栏状态，移动端仍保持普通菜单按钮。 -->
        <n-button
          class="sd-sidebar-toggle"
          :class="{ 'sd-sidebar-toggle--collapsed': props.collapsed && !isMobileMode }"
          size="small"
          quaternary
          circle
          :aria-label="props.collapsed && !isMobileMode ? '展开侧栏' : '收起侧栏'"
          :type="props.collapsed && !isMobileMode ? 'primary' : 'default'"
          @click="emit('toggleSidebar')"
        >
          <template #icon>
            <n-icon size="1.2rem">
              <i-tabler-menu-2 v-if="isMobileMode" />
              <i-tabler-layout-sidebar-left-expand v-else-if="props.collapsed" />
              <i-tabler-layout-sidebar-left-collapse v-else />
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
        <button type="button" class="search-entry" aria-label="搜索" @click="emit('openSearch')">
          <span class="search-label">
            <n-icon size="1.1rem">
              <i-tabler-search />
            </n-icon>
            <span>搜索</span>
          </span>
          <span class="search-shortcut">Ctrl k</span>
        </button>

        <AppThemeSwitch />

        <!-- 窄屏下 badge 兼作版本信息入口，桌面端版本信息已经常驻，badge 只表达渠道。 -->
        <template v-if="isCompactMode">
          <AppVersionPopover />
        </template>
        <template v-else>
          <div class="version-summary">
            <AppChannelTag />
            <AppVersionDetails />
          </div>
        </template>
      </div>
    </template>
  </n-page-header>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import { appNavigation } from '@/router/navigation';
import { buildBreadcrumbItems } from '@/router/navigationModel';
import AppChannelTag from './AppChannelTag.vue';
import AppThemeSwitch from './AppThemeSwitch.vue';
import AppVersionDetails from './AppVersionDetails.vue';
import AppVersionPopover from './AppVersionPopover.vue';
import type { AppShellViewportMode } from './appShellLayout';

const props = defineProps<{
  collapsed: boolean;
  viewportMode: AppShellViewportMode;
}>();

const emit = defineEmits<{
  toggleSidebar: [];
  openSearch: [];
}>();

const route = useRoute();

const breadcrumbItems = computed(() =>
  buildBreadcrumbItems(appNavigation, route.path, String(route.meta.title ?? '当前页面'))
);
const isCompactMode = computed(() => props.viewportMode !== 'desktop');
const isMobileMode = computed(() => props.viewportMode === 'mobile');
const visibleBreadcrumbItems = computed(() =>
  isCompactMode.value ? breadcrumbItems.value.slice(0, -1) : breadcrumbItems.value
);
</script>

<style scoped>
.sd-breadcrumb-bar {
  min-height: 56px;
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
  transition:
    background-color var(--sd-transition-fast),
    color var(--sd-transition-fast);
}

.sd-sidebar-toggle--collapsed {
  background: var(--sd-bg-selected);
  color: var(--sd-primary);
}

.sd-sidebar-toggle--collapsed:hover {
  background: var(--sd-bg-selected-strong);
  color: var(--sd-primary);
}

/* 渠道 badge 与版本信息是一组，badge 与两行文字整体居中对齐。 */
.version-summary {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.4rem;
  margin-left: 0.25rem;
}

:deep(.n-page-header__title) {
  width: 100%;
}

.sd-page-actions {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.search-entry {
  display: inline-flex;
  height: 34px;
  align-items: center;
  border: 0;
  border-radius: var(--sd-radius-sm);
  background: var(--sd-bg-control);
  color: var(--sd-text-secondary);
  cursor: pointer;
  font: inherit;
  gap: 0.55rem;
  line-height: 1;
  padding: 0 0.5rem 0 0.7rem;
  transition:
    background-color var(--sd-transition-fast),
    color var(--sd-transition-fast);
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
  border-radius: var(--sd-radius-sm);
  background: var(--sd-bg-elevated);
  color: var(--sd-text-muted);
  font-size: 0.75rem;
  line-height: 1;
  padding: 0 0.45rem;
  white-space: nowrap;
}

:deep(.n-breadcrumb-item:last-child .n-breadcrumb-item__link) {
  color: var(--sd-text-primary);
  font-weight: 600;
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

.sd-breadcrumb-bar--compact :deep(.n-page-header__main) {
  flex: 1 1 auto;
  min-width: 0;
}

.sd-breadcrumb-bar--compact :deep(.n-page-header__title),
.sd-breadcrumb-bar--compact :deep(.n-page-header__extra) {
  min-width: 0;
}

.sd-breadcrumb-bar--compact .sd-breadcrumb-title {
  min-width: 0;
  overflow: hidden;
}

.sd-breadcrumb-bar--compact :deep(.n-breadcrumb) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sd-breadcrumb-bar--compact .sd-page-actions {
  flex-shrink: 0;
  gap: 0.25rem;
}

.sd-breadcrumb-bar--compact .search-entry {
  flex: 0 0 34px;
  width: 34px;
  justify-content: center;
  padding: 0;
}

.sd-breadcrumb-bar--compact .search-label span,
.sd-breadcrumb-bar--compact .search-shortcut {
  display: none;
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

  .sd-breadcrumb-bar {
    padding-inline: 0.75rem;
  }
}
</style>
