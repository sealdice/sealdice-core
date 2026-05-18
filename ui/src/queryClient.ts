import { MutationCache, QueryCache, QueryClient } from '@tanstack/vue-query';

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

