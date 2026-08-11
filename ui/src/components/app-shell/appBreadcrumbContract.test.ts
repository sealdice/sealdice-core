import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { it } from 'vitest';

const currentDir = path.dirname(fileURLToPath(import.meta.url));

it('renders an old-ui fallback entry in the breadcrumb action area', async () => {
  const source = readFileSync(path.resolve(currentDir, 'AppBreadcrumb.vue'), 'utf8');

  const assertMatch = (pattern: RegExp, label: string) => {
    if (!pattern.test(source)) {
      throw new Error(`expected ${label}`);
    }
  };

  assertMatch(
    /resolveOldUIUrlFromLocation/,
    'AppBreadcrumb.vue to resolve the old UI URL from shared config'
  );
  assertMatch(/tag="a"/, 'AppBreadcrumb.vue to render the fallback entry as a real link');
  assertMatch(/:href="oldUIUrl"/, 'AppBreadcrumb.vue to bind the fallback entry href');
  assertMatch(/回退老 UI/, 'AppBreadcrumb.vue to expose the old UI fallback label');
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
});
