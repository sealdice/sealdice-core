import type { RouteLocationNormalized, RouteRecordRaw } from 'vue-router';

type RouteMetaMap = Record<string, RouteRecordRaw['meta']>;

// vue-router/auto-routes 会把 /tool/resource 生成成父 /tool + 子 resource。
// 合并产品导航 meta 时必须用累计完整路径查表，同时保留子路由原本的相对 path。
export function withRouteMeta(
  records: readonly RouteRecordRaw[],
  routeMeta: RouteMetaMap,
  parentPath = '',
): RouteRecordRaw[] {
  return records.map(record => {
    const baseRecord = record as RouteRecordRaw;
    const fullPath = resolveFullPath(parentPath, baseRecord.path);
    const nextRecord = {
      ...baseRecord,
      meta: {
        ...baseRecord.meta,
        ...routeMeta[fullPath],
      },
      children: baseRecord.children ? withRouteMeta(baseRecord.children, routeMeta, fullPath) : baseRecord.children,
    } as RouteRecordRaw;

    // 文件路由会正确生成动态 path，但页面组件需要稳定的 string prop。
    // 在这里统一转换，页面就不用关心 route.params 的 string/string[] 分支。
    if (fullPath === '/custom-text/:category') {
      nextRecord.props = (route: RouteLocationNormalized) => ({
        category: String((route.params as Record<string, string | string[] | undefined>)['category'] ?? ''),
      });
    }

    return nextRecord;
  });
}

function resolveFullPath(parentPath: string, path: string): string {
  if (path.startsWith('/')) return normalizePath(path);
  if (!parentPath || parentPath === '/') return normalizePath(`/${path}`);
  return normalizePath(`${parentPath}/${path}`);
}

function normalizePath(path: string): string {
  const next = path.replace(/\/+/g, '/');
  if (next.length > 1) return next.replace(/\/$/, '');
  return next || '/';
}
