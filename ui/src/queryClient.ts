import { MutationCache, QueryCache, QueryClient } from '@tanstack/vue-query';

// Vue Query 是本项目服务端状态的默认缓存层。
// 全局 onError 只在开发环境打印，用户可见的错误提示由 api/client.ts 统一处理，
// 避免同一个请求同时弹多条错误消息。
const defaultOnError = (error: unknown) => {
  if (import.meta.env.DEV) {
    console.error('[vue-query] error', error);
  }
};

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: defaultOnError,
  }),
  mutationCache: new MutationCache({
    onError: defaultOnError,
  }),
  defaultOptions: {
    queries: {
      // 管理后台数据变化频繁但不需要每次切页都立即打后端。
      // 1 分钟 staleTime 保持页面切换流畅，显式保存/删除后由 mutation 精准 invalidate。
      staleTime: 1000 * 60, // 1 min
      gcTime: 1000 * 60 * 5, // 5 min
      refetchOnWindowFocus: true,
      refetchOnMount: true,
      retry: 1,
    },
    mutations: {
      retry: 0,
    },
  },
});
