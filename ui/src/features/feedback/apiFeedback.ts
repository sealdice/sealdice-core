import { computed } from 'vue';
import { createDiscreteApi, darkTheme, lightTheme, dateZhCN, zhCN } from 'naive-ui';
import { appPinia } from '@/pinia';
import { useThemeStore } from '@/features/theme/store';
import type { GlobalApiFeedback } from '@/api/client';

export function createApiFeedback() {
  const theme = useThemeStore(appPinia);
  const configProviderProps = computed(() => ({
    locale: zhCN,
    dateLocale: dateZhCN,
    theme: theme.resolvedTheme === 'dark' ? darkTheme : lightTheme,
    themeOverrides: theme.themeOverrides,
  }));
  const { dialog, unmount } = createDiscreteApi(['dialog'], { configProviderProps });
  let activeKey = '';
  const report: GlobalApiFeedback = (error, clearSession) => {
    const offline = error.status === 0;
    const unauthorized = error.status === 401;
    const key = `${error.status}:${offline ? '' : error.message}`;
    if (activeKey === key) return;
    activeKey = key;
    dialog.warning({
      title: offline ? '连接异常' : unauthorized ? '身份信息' : '接口报错',
      content: offline
        ? '连接暂时中断，请检查后台是否正常运行。'
        : `${error.status}：${error.message}`,
      positiveText: unauthorized ? '重新登录' : '我知道了',
      maskClosable: false,
      onPositiveClick: unauthorized ? clearSession : undefined,
      onAfterLeave: () => {
        if (activeKey === key) activeKey = '';
      },
    });
  };
  return { report, dispose: unmount };
}
