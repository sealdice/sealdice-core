<script setup lang="ts">
import { darkTheme, lightTheme, zhCN, dateZhCN, type GlobalThemeOverrides } from 'naive-ui';
import { computed, defineAsyncComponent, provide, ref } from 'vue';
import { RouterView } from 'vue-router';
import AppThemeTransition from './components/shared/AppThemeTransition.vue';
import { useRealtimeClient } from './features/realtime/client';
import { useAppTheme } from './features/theme';
import {
  type ThemeTransitionSource,
  triggerThemeTransitionKey,
} from './features/theme/themeTransition';
import type { AppLayoutName } from './router/types';

const layouts = {
  default: defineAsyncComponent(() => import('./layouts/DefaultLayout.vue')),
  plain: defineAsyncComponent(() => import('./layouts/PlainLayout.vue')),
  wide: defineAsyncComponent(() => import('./layouts/WideLayout.vue')),
} satisfies Record<AppLayoutName, unknown>;

// App 是全局 provider 和 layout 分发层。页面不要直接挂全局 provider，
// 否则会出现消息、弹窗、QueryClient 或主题状态多实例的问题。
const { resolvedTheme, toggleTheme } = useAppTheme();
const themeTransitionRef = ref<InstanceType<typeof AppThemeTransition> | null>(null);

// 主题切换动画需要知道点击来源坐标，所以通过 provide 暴露给任意按钮调用。
// 如果动画组件尚未挂载，则回退到普通主题切换，保证首屏不会因为 ref 为空失败。
provide(triggerThemeTransitionKey, (source?: ThemeTransitionSource) => {
  if (themeTransitionRef.value) {
    themeTransitionRef.value.toggle(source);
  } else {
    toggleTheme();
  }
});

const activeTheme = computed(() => (resolvedTheme.value === 'dark' ? darkTheme : lightTheme));

// Naive UI 主题覆盖是本项目视觉系统的入口。业务页面应优先使用 CSS 变量
// 或 Naive token，不要在页面里重复写一套按钮/菜单主色。
const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#1d4ed8',
    primaryColorHover: '#1e40af',
    primaryColorPressed: '#1e3a8a',
    primaryColorSuppl: '#2563eb',
    // borderRadius: '18px',
    // fontFamily: '"Lato", "Segoe UI", sans-serif',
  },
  Menu: {
    itemTextColor: '#ffffff',
    itemTextColorHover: '#ffffff',
    itemTextColorActive: '#fcd34d',
    itemTextColorActiveHover: '#fcd34d',
    itemTextColorChildActive: '#fcd34d',
    itemTextColorChildActiveHover: '#fcd34d',
    itemIconColor: '#ffffff',
    itemIconColorHover: '#ffffff',
    itemIconColorActive: '#fcd34d',
    itemIconColorActiveHover: '#fcd34d',
    itemIconColorChildActive: '#fcd34d',
    itemIconColorChildActiveHover: '#fcd34d',
    arrowColor: '#ffffff',
    arrowColorHover: '#ffffff',
    arrowColorActive: '#fcd34d',
    itemColorHover: 'rgba(67, 74, 84, 0.76)',
    itemColorActive: 'transparent',
    itemColorActiveHover: 'rgba(67, 74, 84, 0.76)',
    itemColorActiveCollapsed: 'transparent',
    borderRadius: '0',
  },
  Layout: {
    color: 'var(--sd-bg-shell)',
    siderColor: 'var(--sd-bg-sidebar)',
    headerColor: 'var(--sd-bg-shell)',
    footerColor: 'var(--sd-bg-shell)',
    colorEmbedded: 'var(--sd-bg-page)',
  },
};

const darkThemeOverrides: GlobalThemeOverrides = {
  ...lightThemeOverrides,
  common: {
    ...lightThemeOverrides.common,
    borderColor: '#334155',
    bodyColor: '#0f172a',
    cardColor: '#182133',
    modalColor: '#182133',
    popoverColor: '#182133',
  },
  DataTable: {
    thColor: '#111827',
    tdColor: '#182133',
    tdColorHover: '#1f2a40',
    hoverColor: '#1f2a40',
    borderColor: '#334155',
  },
  Drawer: {
    color: '#182133',
  },
  Dropdown: {
    color: '#182133',
  },
  Layout: {
    color: 'var(--sd-bg-shell)',
    siderColor: 'var(--sd-bg-sidebar)',
    headerColor: 'var(--sd-bg-shell)',
    footerColor: 'var(--sd-bg-shell)',
    colorEmbedded: 'var(--sd-bg-page)',
  },
};

const themeOverrides = computed<GlobalThemeOverrides>(() =>
  resolvedTheme.value === 'dark' ? darkThemeOverrides : lightThemeOverrides,
);

// 实时通道是全局单例：App 挂载后根据 token 自动连接，业务模块只订阅事件。
// 这样首页日志、连接管理等页面可以共享同一条 WS/SSE 连接。
useRealtimeClient();

</script>

<template>
  <n-config-provider
    :locale="zhCN"
    :date-locale="dateZhCN"
    :theme="activeTheme"
    :theme-overrides="themeOverrides"
  >
    <n-message-provider>
      <n-notification-provider>
        <n-modal-provider>
          <n-dialog-provider>
            <n-loading-bar-provider>
              <RouterView v-slot="{ Component, route }">
                <component :is="layouts[route.meta.layout ?? 'default']">
                  <component :is="Component" />
                </component>
              </RouterView>
              <AppThemeTransition ref="themeTransitionRef" />
            </n-loading-bar-provider>
          </n-dialog-provider>
        </n-modal-provider>
      </n-notification-provider>
    </n-message-provider>
  </n-config-provider>
</template>
