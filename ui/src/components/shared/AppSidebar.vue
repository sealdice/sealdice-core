<script setup lang="tsx">
import { computed, h, ref, watch } from 'vue';
import { RouterLink, useRoute } from 'vue-router';
import type { MenuOption } from 'naive-ui';
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

function renderIcon(name?: string) {
  switch (name) {
    case 'home':
      return <i-ep-house />;
    case 'connection':
      return <i-ep-connection />;
    case 'edit':
      return <i-ep-edit-pen />;
    case 'reply':
      return <i-ep-chat-line-round />;
    case 'deck':
      return <i-ep-collection />;
    case 'story':
      return <i-ep-notebook />;
    case 'js':
      return <i-ep-cpu />;
    case 'helpdoc':
      return <i-ep-document />;
    case 'censor':
      return <i-ep-warning />;
    case 'operation':
      return <i-ep-operation />;
    case 'setting':
      return <i-ep-setting />;
    case 'base-setting':
      return <i-ep-setting />;
    case 'group':
      return <i-ep-user-filled />;
    case 'ban':
      return <i-ep-circle-close-filled />;
    case 'dice':
      return <i-ep-grid />;
    case 'backup':
      return <i-ep-folder-opened />;
    case 'advanced-setting':
      return <i-ep-tools />;
    case 'tools':
      return <i-ep-tools />;
    case 'test':
      return <i-ep-magic-stick />;
    case 'resource':
      return <i-ep-folder />;
    case 'star':
      return <i-ep-star-filled />;
    default:
      return <i-ep-menu />;
  }
}

function icon(name?: string) {
  if (!name) return undefined;
  return () => <n-icon>{renderIcon(name)}</n-icon>;
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
      :collapsed-icon-size="22"
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
}

.sd-sidebar-menu {
  flex: 1 1 auto;
  min-height: 0;
  background: transparent;
}

:deep(.sd-menu-link) {
  color: inherit;
  text-decoration: none;
}
</style>
