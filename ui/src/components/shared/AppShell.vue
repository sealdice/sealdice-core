<script setup lang="ts">
import { defineAsyncComponent, nextTick, ref } from 'vue';
import { breakpointsTailwind, useBreakpoints, useEventListener } from '@vueuse/core';
import { useMessage } from 'naive-ui';
import AppBreadcrumb from './AppBreadcrumb.vue';
import AppSidebar from './AppSidebar.vue';
import AppUnlockDialog from './AppUnlockDialog.vue';
import { useAuthSession } from '@/features/auth/useAuthSession';

interface AppSearchMenuHandle {
  open: () => void;
}

const loadAppSearchMenu = () => import('./AppSearchMenu.vue');
const AppSearchMenu = defineAsyncComponent(loadAppSearchMenu);

const breakpoints = useBreakpoints(breakpointsTailwind);
const isDesktop = breakpoints.greater('md');

const drawerMenu = ref(false);
const collapsedMenu = ref(!isDesktop.value);
const advancedConfigCounter = ref(0);
const renderSearchMenu = ref(false);
const searchMenuRef = ref<AppSearchMenuHandle | null>(null);
const message = useMessage();
const authSession = useAuthSession();

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
  if (isDesktop.value) {
    collapsedMenu.value = !collapsedMenu.value;
  } else {
    drawerMenu.value = true;
  }
}

async function openSearch() {
  renderSearchMenu.value = true;
  await loadAppSearchMenu();
  await nextTick();
  searchMenuRef.value?.open();
}

useEventListener(window, 'keydown', event => {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault();
    void openSearch();
  }
});

void authSession.tryDefaultSignin();
</script>

<template>
  <n-layout id="root" class="sd-shell">
    <n-layout class="sd-body" has-sider>
      <n-layout-sider
        class="sd-sidebar no-scrollbar"
        collapse-mode="width"
        :collapsed-width="64"
        :width="240"
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
          :mobile-mode="!isDesktop"
          @toggle-sidebar="toggleSidebar"
          @open-search="openSearch"
        />
        <main class="sd-main-container">
          <slot />
        </main>
      </n-layout-content>
    </n-layout>

    <n-drawer
      v-model:show="drawerMenu"
      class="sd-drawer"
      default-width="50%"
      placement="left"
    >
      <n-drawer-content body-content-style="padding: 0;" :native-scrollbar="false">
        <AppSidebar
          :advanced-config-counter="advancedConfigCounter"
          @enable-advanced-config="enableAdvancedConfig"
        />
      </n-drawer-content>
    </n-drawer>

    <AppUnlockDialog />
    <AppSearchMenu
      v-if="renderSearchMenu"
      ref="searchMenuRef"
      :advanced-config-counter="advancedConfigCounter"
    />
  </n-layout>
</template>

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
  background: var(--sd-bg-sidebar);
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
  min-height: 100%;
  padding: 1.5rem 2rem 2rem;
}

:global(.sd-drawer .n-drawer-content) {
  background: var(--sd-bg-sidebar);
}

@media screen and (max-width: 639.9px) {
  .sd-sidebar {
    display: none;
  }

  .sd-main-container {
    padding: 1rem;
  }
}
</style>
