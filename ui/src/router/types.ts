import type { RouteRecordName } from 'vue-router';

export type AppLayoutName = 'default' | 'plain' | 'wide';

export interface NavigationItem {
  label: string;
  routeName?: RouteRecordName;
  path?: string;
  icon?: string;
  hidden?: boolean;
  requiresAdvancedConfig?: boolean;
  dynamicChildren?: 'customTextCategories';
  children?: NavigationItem[];
}

export interface NavigationSearchItem {
  label: string;
  path: string;
  icon?: string;
}

declare module 'vue-router' {
  interface RouteMeta {
    layout?: AppLayoutName;
    title?: string;
  }
}
