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

  assertMatch(
    /<AppInstallButton v-if="!isCompactMode" \/>/,
    'AppBreadcrumb.vue to keep the install action directly visible on desktop'
  );
  assertMatch(
    /<AppHeaderOverflowMenu v-if="isCompactMode" \/>/,
    'AppBreadcrumb.vue to move secondary actions into an overflow menu on compact widths'
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
});

it('renders the old UI fallback in the sidebar footer', async () => {
  const source = readFileSync(path.resolve(currentDir, 'AppSidebar.vue'), 'utf8');

  const expectedPatterns: Array<[RegExp, string]> = [
    [/class="sd-sidebar-footer"/, 'a dedicated sidebar footer'],
    [/<i-tabler-history\s*\/>/, 'the Tabler history icon'],
    [/>旧版 UI</, 'the old UI label'],
    [/:href="oldUIUrl"/, 'the resolved old UI URL'],
    [/resolveOldUIUrlFromLocation/, 'the shared old UI URL resolver'],
  ];

  for (const [pattern, label] of expectedPatterns) {
    if (!pattern.test(source)) throw new Error(`expected AppSidebar.vue to render ${label}`);
  }
  if (source.includes('主题色')) {
    throw new Error('AppSidebar.vue should not expose theme color customization');
  }
});
