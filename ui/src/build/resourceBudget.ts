import { gzipSync } from 'node:zlib';
import type { Plugin } from 'vite';

// 预算针对构建产物，不等同于浏览器首屏下载量或加载时间。
export const RESOURCE_BUDGET = {
  chunkBytes: 650 * 1024,
  entryGzipBytes: 200 * 1024,
  initialJsGzipBytes: 750 * 1024,
  precacheBytes: 5 * 1024 * 1024,
};

export function checkBudget(name: string, bytes: number, limit: number): void {
  if (bytes > limit) throw new Error(`${name}: ${bytes} bytes exceeds budget ${limit}`);
}

export function resourceBudget(): Plugin {
  return {
    name: 'sealdice-resource-budget',
    apply: 'build',
    generateBundle(_options, bundle) {
      this.emitFile({
        type: 'asset',
        fileName: '.vite/resource-budget.json',
        source: JSON.stringify(RESOURCE_BUDGET),
      });
      const initial = new Set<string>();
      const visit = (name: string) => {
        if (initial.has(name)) return;
        const item = bundle[name];
        if (!item || item.type !== 'chunk') return;
        initial.add(name);
        item.imports.forEach(visit);
      };
      for (const item of Object.values(bundle)) {
        if (item.type !== 'chunk') continue;
        checkBudget(item.fileName, Buffer.byteLength(item.code), RESOURCE_BUDGET.chunkBytes);
        if (item.isEntry) {
          checkBudget(
            `entry ${item.fileName} (gzip)`,
            gzipSync(item.code).byteLength,
            RESOURCE_BUDGET.entryGzipBytes
          );
          visit(item.fileName);
        }
      }
      const initialBytes = [...initial].reduce((size, name) => {
        const item = bundle[name];
        return size + (item?.type === 'chunk' ? gzipSync(item.code).byteLength : 0);
      }, 0);
      checkBudget(
        'initial static JS graph (gzip)',
        initialBytes,
        RESOURCE_BUDGET.initialJsGzipBytes
      );
      console.log(`Resource budget: initial static JS graph ${initialBytes} gzip bytes`);
    },
  };
}

export function enforcePrecacheBudget<T extends { url: string; size: number }>(entries: T[]) {
  // 调试控制台是显式开启的在线诊断能力，不挤占普通管理页面的离线缓存。
  const manifest = entries.filter(entry => !/(?:^|\/)vendor-debug-[^/]+\.js$/.test(entry.url));
  const total = manifest.reduce((size, entry) => size + entry.size, 0);
  checkBudget('PWA precache', total, RESOURCE_BUDGET.precacheBytes);
  console.log(`Resource budget: PWA precache ${total} bytes, ${manifest.length} entries`);
  return { manifest, warnings: [] };
}
