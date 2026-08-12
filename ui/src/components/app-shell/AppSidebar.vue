<template>
  <div class="sd-sidebar-content">
    <AppSidebarBrand
      :collapsed="props.collapsed"
      @enable-advanced-config="emit('enableAdvancedConfig')"
    />
    <n-menu
      class="sd-sidebar-menu"
      :collapsed="props.collapsed"
      :collapsed-width="64"
      :icon-size="20"
      :collapsed-icon-size="20"
      :options="options"
      :value="activeValue"
      :expanded-keys="expandedKeys"
      :expand-icon="expandIcon"
      accordion
      @update:expanded-keys="expandedKeys = $event"
    />
    <footer class="sd-sidebar-footer" :class="{ 'sd-sidebar-footer--collapsed': props.collapsed }">
      <n-tooltip :disabled="!props.collapsed" placement="right">
        <template #trigger>
          <n-button
            tag="a"
            quaternary
            class="sd-sidebar-footer-action"
            aria-label="旧版 UI"
            :href="oldUIUrl"
          >
            <template #icon>
              <n-icon><i-tabler-history /></n-icon>
            </template>
            <span v-if="!props.collapsed">旧版 UI</span>
          </n-button>
        </template>
        旧版 UI
      </n-tooltip>
    </footer>
  </div>
</template>

<script setup lang="tsx">
import { computed, h, ref, watch } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import type { MenuOption } from 'naive-ui';
import { resolveOldUIUrlFromLocation } from '@/api/config';
import AppNavigationIcon from './AppNavigationIcon.vue';
import AppSidebarBrand from './AppSidebarBrand.vue';
import { getNavigationExpandedKeys } from '@/router/navigationModel';
import type { NavigationItem } from '@/router/types';
import { useAppNavigation } from '@/router/useAppNavigation';

const props = withDefaults(
  defineProps<{
    collapsed?: boolean;
    advancedConfigCounter?: number;
  }>(),
  {
    collapsed: false,
    advancedConfigCounter: 0,
  }
);

const emit = defineEmits<{
  enableAdvancedConfig: [];
}>();

const route = useRoute();
const oldUIUrl =
  typeof window !== 'undefined' ? resolveOldUIUrlFromLocation(window.location) : '/old-ui/';
const { navigationTree } = useAppNavigation(() => props.advancedConfigCounter);
const expandedKeys = ref<string[]>([]);

function link(path: string, label: string) {
  return () => h(RouterLink, { to: path, class: 'sd-menu-link' }, () => label);
}

function icon(name?: string) {
  if (!name) return undefined;
  return () => (
    <n-icon>
      <AppNavigationIcon name={name} />
    </n-icon>
  );
}

function expandIcon() {
  return (
    <n-icon>
      <i-tabler-chevron-right />
    </n-icon>
  );
}

function toMenuOption(item: NavigationItem): MenuOption {
  return {
    key: item.path ?? item.label,
    label: item.path ? link(item.path, item.label) : item.label,
    icon: icon(item.icon),
    children: item.children?.map(toMenuOption),
  };
}

const options = computed<MenuOption[]>(() => navigationTree.value.map(toMenuOption));

function normalizePath(path: string) {
  try {
    return decodeURIComponent(path);
  } catch {
    return path;
  }
}

const activeValue = computed(() => {
  return normalizePath(route.path);
});

watch(
  [navigationTree, () => route.path],
  () => {
    expandedKeys.value = getNavigationExpandedKeys(navigationTree.value, route.path);
  },
  { immediate: true }
);
</script>

<style scoped>
.sd-sidebar-content {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  text-align: left;
}

.sd-sidebar-menu {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  background: transparent;
  padding: 0 0.6rem 1rem;
  scrollbar-width: none;
  transition: padding-inline var(--sd-transition-base);
}

.sd-sidebar-menu::-webkit-scrollbar {
  display: none;
}

.sd-sidebar-footer {
  display: grid;
  flex: 0 0 auto;
  grid-template-columns: minmax(0, 1fr);
  gap: 0.4rem;
  margin-top: auto;
  border-top: 1px solid var(--sd-border-sidebar);
  background: var(--sd-bg-sidebar);
  padding: 0.65rem;
  transition: padding var(--sd-transition-base);
}

.sd-sidebar-footer--collapsed {
  grid-template-columns: 1fr;
  justify-items: center;
  padding: 0.55rem 0;
}

.sd-sidebar-footer-action {
  --n-color-focus: var(--sd-bg-sidebar-hover) !important;
  --n-color-hover: var(--sd-bg-sidebar-hover) !important;
  --n-color-pressed: var(--sd-bg-sidebar-pressed) !important;
  --n-text-color: var(--sd-text-inverse-soft) !important;
  --n-text-color-focus: var(--sd-text-inverse) !important;
  --n-text-color-hover: var(--sd-text-inverse) !important;
  --n-text-color-pressed: var(--sd-text-inverse) !important;
  width: 100%;
  min-width: 0;
  color: var(--sd-text-inverse-soft);
  font-size: 0.78rem;
}

.sd-sidebar-footer-action:hover {
  color: var(--sd-text-inverse);
}

.sd-sidebar-footer--collapsed .sd-sidebar-footer-action {
  width: 40px;
  padding-inline: 0;
}

:deep(.sd-sidebar-menu.n-menu--collapsed) {
  padding-inline: 0;
}

:deep(.sd-menu-link) {
  display: inline-flex;
  align-items: center;
  color: inherit;
  text-decoration: none;
  text-align: left;
  width: 100%;
  line-height: 1.2;
}

:deep(.sd-sidebar-menu .n-menu-item-content) {
  border-radius: var(--sd-radius-sm);
  transition:
    background-color var(--sd-transition-fast),
    color var(--sd-transition-fast),
    box-shadow var(--sd-transition-fast);
}

:deep(.sd-sidebar-menu .n-menu-item-content--selected) {
  box-shadow: none;
}

:deep(.sd-sidebar-menu > .n-menu-item > .n-menu-item-content--selected::before),
:deep(.sd-sidebar-menu > .n-submenu > .n-menu-item > .n-menu-item-content--child-active::before) {
  box-shadow: inset 3px 0 var(--sd-primary);
}

:deep(.sd-sidebar-menu > .n-submenu > .n-menu-item > .n-menu-item-content--child-active::before) {
  background-color: transparent;
}

:deep(.sd-sidebar-menu .n-menu-item-content__icon .n-icon) {
  font-size: 18px;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transform: translateY(-0.5px);
}

:deep(.sd-sidebar-menu.n-menu .n-menu-item-content .n-menu-item-content__arrow) {
  transform: rotate(90deg);
}

:deep(
  .sd-sidebar-menu.n-menu
    .n-menu-item-content.n-menu-item-content--collapsed
    .n-menu-item-content__arrow
) {
  transform: rotate(0deg);
}

:deep(.sd-sidebar-menu .n-menu-item-content-header) {
  display: flex;
  align-items: center;
  min-height: 20px;
  text-align: left;
  line-height: 1.2;
}

:deep(.sd-sidebar-menu .n-menu-item-content-header a) {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  line-height: 20px;
}

:deep(.sd-sidebar-menu .n-menu-item-group-title) {
  height: 22px;
  font-size: 0.72rem;
  opacity: 0.72;
}
</style>
