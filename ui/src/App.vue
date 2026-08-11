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
                <AppTestModeFrame
                  :active="testMode.isTestMode.value"
                  :banner-text="testMode.bannerText.value"
                  :watermark-text="testMode.watermarkText.value"
                >
                  <PlainLayout v-if="route.meta.layout === 'plain'">
                    <Transition name="page-fade" mode="out-in">
                      <component :is="Component" :key="route.path" />
                    </Transition>
                  </PlainLayout>
                  <AppShell
                    v-else
                    :content-mode="getAppShellContentMode(route.meta.layout)"
                    :container-mode="getAppShellContainerMode(route.meta.layout)"
                  >
                    <Transition name="page-fade" mode="out-in">
                      <component :is="Component" :key="route.path" />
                    </Transition>
                  </AppShell>
                </AppTestModeFrame>
              </RouterView>
              <AppUnlockDialog />
            </n-loading-bar-provider>
          </n-dialog-provider>
        </n-modal-provider>
      </n-notification-provider>
    </n-message-provider>
  </ProConfigProvider>
</template>

<script setup lang="ts">
import { darkTheme, lightTheme, dateZhCN } from 'naive-ui';
import { ProConfigProvider, zhCN } from 'pro-naive-ui';
import { computed } from 'vue';
import { RouterView } from 'vue-router';
import AppTestModeFrame from './components/app-shell/AppTestModeFrame.vue';
import AppShell from './components/app-shell/AppShell.vue';
import AppUnlockDialog from './components/app-shell/AppUnlockDialog.vue';
import PlainLayout from './layouts/PlainLayout.vue';
import {
  getAppShellContentMode,
  getAppShellContainerMode,
} from './components/app-shell/appShellLayout';
import { useAuthSession } from './features/auth/useAuthSession';
import { useRealtimeClient } from './features/realtime/client';
import { useTestMode } from './features/testMode/state';
import { useAppTheme } from './features/theme';
// App 是全局 provider 和 layout 分发层。页面不要直接挂全局 provider，
// 否则会出现消息、弹窗、QueryClient 或主题状态多实例的问题。
const { resolvedTheme, themeOverrides } = useAppTheme();

const activeTheme = computed(() => (resolvedTheme.value === 'dark' ? darkTheme : lightTheme));
const testMode = useTestMode();
const authSession = useAuthSession();

// 全局 provider 只接收最终主题对象。颜色计算集中在 features/theme，
// 避免根组件和业务页面各自维护一套主色、状态色和暗色覆盖。

// 实时通道是全局单例：App 挂载后根据 token 自动连接，业务模块只订阅事件。
// 这样首页日志、连接管理等页面可以共享同一条 SSE 连接。
useRealtimeClient();
void authSession.tryDefaultSignin();
</script>

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
