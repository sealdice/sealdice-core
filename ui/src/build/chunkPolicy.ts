export function classifyVendorChunk(id: string): string | undefined {
  if (!id.includes('/node_modules/')) return undefined;

  if (id.includes('/eruda/')) {
    return 'vendor-debug';
  }

  if (
    id.includes('/vue/') ||
    id.includes('/@vue/') ||
    id.includes('/vue-router/') ||
    id.includes('/@tanstack/query-core/') ||
    id.includes('/@tanstack/vue-query/')
  ) {
    return 'vendor-framework';
  }

  // 全局队列跨多个页面复用，上传协议库独立缓存，不混入应用入口。
  if (id.includes('/@uppy/') || id.includes('/tus-js-client/')) {
    return 'vendor-upload';
  }

  if (id.includes('/lodash-es/')) {
    return 'vendor-lodash';
  }

  if (id.includes('/dayjs/') || id.includes('/date-fns/')) {
    return 'vendor-date';
  }

  if (
    id.includes('/@vueuse/core/') ||
    id.includes('/@vueuse/shared/') ||
    id.includes('/@ant-design/colors/') ||
    id.includes('/@ant-design/fast-color/') ||
    id.includes('/pinyin-pro/') ||
    id.includes('/dompurify/')
  ) {
    return 'vendor-utility';
  }

  return undefined;
}
