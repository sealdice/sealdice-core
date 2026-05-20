import type { RouteRecordRaw } from 'vue-router';
import { appNavigation } from './navigation';
import { buildRouteMeta } from './navigationModel';

// routeMeta 从产品导航模型派生。新增页面时只维护 navigation.ts，
// 侧栏、搜索、面包屑和路由元信息会一起更新。
export const routeMeta: Record<string, RouteRecordRaw['meta']> = buildRouteMeta(appNavigation);
