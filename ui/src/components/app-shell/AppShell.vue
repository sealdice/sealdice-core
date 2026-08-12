<template>
  <n-layout id="root" class="sd-shell">
    <n-layout class="sd-body" has-sider>
      <n-layout-sider
        class="sd-sidebar no-scrollbar"
        collapse-mode="width"
        :collapsed-width="64"
        :width="224"
        :collapsed="collapsedMenu"
        bordered
        :native-scrollbar="false"
        @collapse="collapsedMenu = true"
        @expand="collapsedMenu = false"
      >
        <AppSidebar
          :collapsed="collapsedMenu"
          :advanced-config-counter="advancedConfigCounter"
          @enable-advanced-config="enableAdvancedConfig"
        />
      </n-layout-sider>

      <n-layout-content
        class="sd-content-pane"
        :native-scrollbar="false"
        content-class="sd-content-inner"
        embedded
      >
        <AppBreadcrumb
          :collapsed="collapsedMenu"
          :viewport-mode="viewportMode"
          @toggle-sidebar="toggleSidebar"
          @open-search="openSearch"
        />
        <div
          class="sd-floating-panel-layer"
          :class="{ 'sd-floating-panel-layer--active': !!activeUnsavedChangesSource }"
        >
          <AppUnsavedChangesPanel />
        </div>
        <main :class="getAppShellContentClass(props.contentMode)">
          <div :class="getAppShellContainerClass(props.containerMode)">
            <slot />
          </div>
        </main>
      </n-layout-content>
    </n-layout>

    <n-drawer
      v-model:show="drawerMenu"
      class="sd-drawer"
      :default-width="getAppShellDrawerWidth()"
      placement="left"
    >
      <n-drawer-content body-content-style="padding: 0;" :native-scrollbar="false">
        <AppSidebar
          :advanced-config-counter="advancedConfigCounter"
          @enable-advanced-config="enableAdvancedConfig"
        />
      </n-drawer-content>
    </n-drawer>

    <AppSearchMenu
      v-if="renderSearchMenu"
      ref="searchMenuRef"
      :advanced-config-counter="advancedConfigCounter"
    />
  </n-layout>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, ref, watch } from 'vue';
import { useEventListener, useWindowSize } from '@vueuse/core';
import { useDialog, useMessage } from 'naive-ui';
import {
  getAppShellContainerClass,
  getAppShellContentClass,
  getAppShellDrawerWidth,
  getAppShellViewportMode,
  shouldCollapseAppShellSidebar,
  type AppShellContainerMode,
  type AppShellContentMode,
} from './appShellLayout';
import AppBreadcrumb from './AppBreadcrumb.vue';
import AppSidebar from './AppSidebar.vue';
import AppUnsavedChangesPanel from './AppUnsavedChangesPanel.vue';
import {
  activeUnsavedChangesSource,
  hasUnsavedChanges,
  setUnsavedChangesConfirmHandler,
} from '@/features/unsavedChanges';

interface AppSearchMenuHandle {
  open: () => void;
}

const props = withDefaults(
  defineProps<{
    contentMode?: AppShellContentMode;
    containerMode?: AppShellContainerMode;
  }>(),
  {
    contentMode: 'default',
    containerMode: 'default',
  }
);

const loadAppSearchMenu = () => import('./AppSearchMenu.vue');
const AppSearchMenu = defineAsyncComponent(loadAppSearchMenu);

const { width: viewportWidth } = useWindowSize();
const viewportMode = computed(() => getAppShellViewportMode(viewportWidth.value));

const drawerMenu = ref(false);
const collapsedMenu = ref(shouldCollapseAppShellSidebar(viewportMode.value));
const advancedConfigCounter = ref(0);
const renderSearchMenu = ref(false);
const searchMenuRef = ref<AppSearchMenuHandle | null>(null);
const message = useMessage();
const dialog = useDialog();

function enableAdvancedConfig() {
  advancedConfigCounter.value += 1;
  if (advancedConfigCounter.value > 8) {
    message.info('高级设置页已经开启');
  } else if (advancedConfigCounter.value === 8) {
    message.success('已开启高级设置页');
  } else if (advancedConfigCounter.value > 2) {
    message.info(`再按 ${8 - advancedConfigCounter.value} 次开启高级设置页`);
  }
}

function toggleSidebar() {
  if (viewportMode.value === 'mobile') {
    drawerMenu.value = true;
    return;
  }
  collapsedMenu.value = !collapsedMenu.value;
}

async function openSearch() {
  renderSearchMenu.value = true;
  await loadAppSearchMenu();
  await nextTick();
  searchMenuRef.value?.open();
}

watch(viewportMode, mode => {
  collapsedMenu.value = shouldCollapseAppShellSidebar(mode);
  drawerMenu.value = false;
});

useEventListener(window, 'keydown', event => {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault();
    void openSearch();
  }
});

useEventListener(window, 'beforeunload', event => {
  if (!hasUnsavedChanges.value) return;
  event.preventDefault();
  event.returnValue = '';
});

setUnsavedChangesConfirmHandler(
  source =>
    new Promise(resolve => {
      dialog.warning({
        title: '确认离开',
        content: source.confirmMessage,
        positiveText: '确定忽略',
        negativeText: '取消',
        onPositiveClick: () => resolve(true),
        onNegativeClick: () => resolve(false),
        onClose: () => resolve(false),
        onMaskClick: () => resolve(false),
        onEsc: () => resolve(false),
      });
    })
);
</script>

<style scoped>
.sd-shell {
  display: flex;
  width: 100%;
  min-width: 0;
  height: 100vh;
  min-height: 100vh;
  background: var(--sd-bg-shell);
}

:global(.sd-shell > .n-layout-scroll-container) {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
}

.sd-body {
  width: 100%;
  flex: 1 1 auto;
  min-height: 0;
  background: var(--sd-bg-page);
}

:global(.sd-body > .n-layout-scroll-container) {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
}

.sd-sidebar {
  height: 100%;
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  background: var(--sd-bg-sidebar);
  text-align: left;
  transition: width var(--sd-transition-base);
}

.sd-content-pane {
  flex: 1 1 auto;
  min-width: 0;
  background: var(--sd-bg-page);
  text-align: left;
}

:global(.sd-content-inner) {
  width: 100%;
  min-height: 100%;
}

.sd-main-container {
  box-sizing: border-box;
  width: 100%;
  max-width: var(--sd-content-max-width);
  min-height: 100%;
  min-width: 0;
  margin-inline: auto;
  padding: var(--sd-space-xl) clamp(var(--sd-space-md), 2vw, var(--sd-space-2xl))
    var(--sd-space-2xl);
}

.sd-main-container--wide {
  width: 100%;
  max-width: none;
  margin-inline: 0;
}

.sd-page-shell {
  min-width: 0;
}

.sd-page-shell--workspace {
  display: flex;
  min-height: calc(100vh - 5.5rem);
  flex-direction: column;
}

.sd-floating-panel-layer {
  position: fixed;
  top: 4.4rem;
  left: 50%;
  z-index: 40;
  display: flex;
  justify-content: center;
  pointer-events: none;
  width: min(100%, calc(100vw - 15rem));
  transform: translateX(-50%);
}

.sd-floating-panel-layer--active {
  pointer-events: none;
}

:global(.sd-drawer .n-drawer-content) {
  background: var(--sd-bg-sidebar);
}

@media screen and (max-width: 767.9px) {
  .sd-sidebar {
    display: none;
  }

  .sd-main-container {
    padding: var(--sd-space-md);
  }

  .sd-floating-panel-layer {
    top: 4rem;
    width: calc(100vw - 1rem);
  }
}
</style>
