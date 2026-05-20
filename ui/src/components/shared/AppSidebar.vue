<script setup lang="tsx">
import { computed, h, ref, watch } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import type { MenuOption } from 'naive-ui';
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
  },
);

const emit = defineEmits<{
  enableAdvancedConfig: [];
}>();

const route = useRoute();
const { navigationTree } = useAppNavigation(() => props.advancedConfigCounter);
const expandedKeys = ref<string[]>([]);

function link(path: string, label: string) {
  return () => h(RouterLink, { to: path, class: 'sd-menu-link' }, () => label);
}

function icon(name?: string) {
  if (!name) return undefined;
  return () => <n-icon><AppNavigationIcon name={name} /></n-icon>;
}

function expandIcon() {
  return (
    <n-icon>
      <i-carbon-caret-right />
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
  { immediate: true },
);
</script>

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
  </div>
</template>

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
  background: transparent;
  padding: 0;
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
  border-radius: 10px;
}

:deep(.sd-sidebar-menu .n-menu-item-content__icon .n-icon) {
  font-size: 18px;
  line-height: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transform: translateY(-0.5px);
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
