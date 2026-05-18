import type { AppLayoutName, NavigationItem } from './types';

const layoutNames = ['default', 'plain', 'wide'] satisfies AppLayoutName[];

const navigationItems = [
  {
    label: '首页',
    routeName: '/',
  },
] satisfies NavigationItem[];

void layoutNames;
void navigationItems;
