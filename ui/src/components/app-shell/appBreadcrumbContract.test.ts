import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

it('keeps the breadcrumb action area focused on primary shell actions', async () => {
  const source = readFileSync(path.resolve(currentDir, 'AppBreadcrumb.vue'), 'utf8');

  const assertMatch = (pattern: RegExp, label: string) => {
    if (!pattern.test(source)) {
      throw new Error(`expected ${label}`);
    }
  };

  // 搜索在明暗切换左侧：两者都是常驻工具，顺序固定以免跨端漂移。
  assertMatch(
    /search-entry[\s\S]*<AppThemeSwitch \/>/,
    'AppBreadcrumb.vue to place the search entry before the theme switch'
  );
  assertMatch(
    /<AppVersionPopover \/>/,
    'AppBreadcrumb.vue to expose version details behind the channel badge on compact widths'
  );
  assertMatch(
    /<AppChannelTag \/>[\s\S]*<AppVersionDetails \/>/,
    'AppBreadcrumb.vue to render the channel badge beside the version details on desktop'
  );
  assertMatch(
    /viewportMode:\s*AppShellViewportMode/,
    'AppBreadcrumb.vue to distinguish mobile, tablet and desktop viewports'
  );
  assertMatch(
    /visibleBreadcrumbItems/,
    'AppBreadcrumb.vue to hide the current page title on mobile'
  );
  if (source.includes('AppThemePaletteButton')) {
    throw new Error('AppBreadcrumb.vue should not render the theme settings action');
  }
  if (source.includes('回退老 UI')) {
    throw new Error('AppBreadcrumb.vue should not render the old UI fallback action');
  }
  // 安装入口统一收在侧边栏页脚，顶栏不再重复承载。
  if (source.includes('AppInstallButton')) {
    throw new Error('AppBreadcrumb.vue should not render the install action');
  }
});

it('renders the old UI fallback and the install action in the sidebar footer', async () => {
  const source = readFileSync(path.resolve(currentDir, 'AppSidebar.vue'), 'utf8');

  const expectedPatterns: Array<[RegExp, string]> = [
    [/class="sd-sidebar-footer"/, 'a dedicated sidebar footer'],
    [/<i-tabler-history\s*\/>/, 'the Tabler history icon'],
    [/>旧版 UI</, 'the old UI label'],
    [/:href="oldUIUrl"/, 'the resolved old UI URL'],
    [/resolveOldUIUrlFromLocation/, 'the shared old UI URL resolver'],
    // 安装入口与「旧版 UI」并列，桌面与移动端共用同一个落点。
    [/<AppInstallButton :collapsed="props\.collapsed" \/>/, 'the PWA install action'],
  ];

  for (const [pattern, label] of expectedPatterns) {
    if (!pattern.test(source)) throw new Error(`expected AppSidebar.vue to render ${label}`);
  }
  if (source.includes('主题色')) {
    throw new Error('AppSidebar.vue should not expose theme color customization');
  }
});
