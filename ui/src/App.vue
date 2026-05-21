<script setup lang="ts">
import { darkTheme, lightTheme, dateZhCN } from 'naive-ui';
import { ProConfigProvider, zhCN } from 'pro-naive-ui';
import { computed, defineAsyncComponent, provide, ref } from 'vue';
import { RouterView } from 'vue-router';
import AppThemeTransition from './components/app-shell/AppThemeTransition.vue';
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
const { resolvedTheme, themeOverrides, toggleTheme } = useAppTheme();
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

// 全局 provider 只接收最终主题对象。颜色计算集中在 features/theme，
// 避免根组件和业务页面各自维护一套主色、状态色和暗色覆盖。

// 实时通道是全局单例：App 挂载后根据 token 自动连接，业务模块只订阅事件。
// 这样首页日志、连接管理等页面可以共享同一条 WS/SSE 连接。
useRealtimeClient();

</script>

<template>
  <ProConfigProvider
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
                  <Transition name="page-fade" mode="out-in">
                    <component :is="Component" :key="route.path" />
                  </Transition>
                </component>
              </RouterView>
              <AppThemeTransition ref="themeTransitionRef" />
            </n-loading-bar-provider>
          </n-dialog-provider>
        </n-modal-provider>
      </n-notification-provider>
    </n-message-provider>
  </ProConfigProvider>
</template>

<style>
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.2s ease;
}

.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
}
</style>
