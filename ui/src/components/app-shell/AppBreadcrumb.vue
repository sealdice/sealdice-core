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
              <i-carbon-menu v-if="props.mobileMode" />
              <i-carbon-side-panel-open v-else-if="props.collapsed" />
              <i-carbon-side-panel-close v-else />
            </n-icon>
          </template>
        </n-button>

        <n-breadcrumb>
          <n-breadcrumb-item
            v-for="(item, index) in breadcrumbItems"
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
        <AppThemePaletteButton />
        <AppInstallButton />

        <button type="button" class="search-entry" @click="emit('openSearch')">
          <span class="search-label">
            <n-icon size="1.1rem">
              <i-carbon-search />
            </n-icon>
            <span>搜索</span>
          </span>
          <span class="search-shortcut">Ctrl k</span>
        </button>

        <n-badge :show="!newsChecked" value="new">
          <n-tooltip>
            <template #trigger>
              <n-button
                quaternary
                circle
                :type="newsChecked ? 'default' : 'error'"
                @click="dialogFeed = true"
              >
                <template #icon>
                  <n-icon size="1.3rem">
                    <i-carbon-notification />
                  </n-icon>
                </template>
              </n-button>
            </template>
            海豹新闻
          </n-tooltip>
        </n-badge>

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

  <n-modal
    v-model:show="dialogFeed"
    :closable="false"
    :mask-closable="false"
    class="feed-modal"
    preset="card"
    title="海豹新闻"
  >
    <template #header-extra>
      <n-button type="primary" @click="dialogFeed = false">
        确认已读
      </n-button>
    </template>
    <div class="feed-content" v-safe-html="newsData"></div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import { useBaseOverview } from '@/features/base/useBaseOverview';
import { appNavigation } from '@/router/navigation';
import { buildBreadcrumbItems } from '@/router/navigationModel';
import AppInstallButton from './AppInstallButton.vue';
import AppThemePaletteButton from './AppThemePaletteButton.vue';
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
const dialogFeed = ref(false);
const newsChecked = ref(true);
const newsData = ref('<div>暂无内容</div>');
const { overview, isStable, hasNewVersion } = useBaseOverview();

const breadcrumbItems = computed(() =>
  buildBreadcrumbItems(appNavigation, route.path, String(route.meta.title ?? '当前页面')),
);
</script>

<style scoped>
.sd-breadcrumb-bar {
  position: relative;
  z-index: 10;
  border-bottom: 1px solid var(--sd-border-soft);
  background: var(--sd-bg-elevated-tint);
  padding: 0.65rem 1rem;
}

@supports (color: color-mix(in srgb, white, black)) {
  .sd-breadcrumb-bar {
    background: color-mix(in srgb, var(--sd-bg-elevated), transparent 12%);
  }
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
  font-size: 0.72rem;
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
  font-size: 0.76rem;
  line-height: 1;
  padding: 0 0.45rem;
  white-space: nowrap;
}

.feed-content {
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
  .search-entry {
    width: 34px;
    justify-content: center;
    padding: 0;
  }

  .search-label span,
  .search-shortcut {
    display: none;
  }

  .version-summary {
    display: none;
  }
}
</style>
