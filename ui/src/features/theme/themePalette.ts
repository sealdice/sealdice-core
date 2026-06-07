import { generate } from '@ant-design/colors';
import type { GlobalThemeOverrides } from 'naive-ui';
import type { ResolvedTheme, ThemeStorage } from './themeState';

export type ThemeColorKey = 'primary' | 'info' | 'success' | 'warning' | 'error';
export type ThemePalette = Record<ThemeColorKey, string>;

export const THEME_PALETTE_STORAGE_KEY = 'sd-theme-palette';

export const THEME_COLOR_KEYS: ThemeColorKey[] = [
  'primary',
  'info',
  'success',
  'warning',
  'error',
];

export const DEFAULT_THEME_PALETTE: ThemePalette = {
  primary: '#1d4ed8',
  info: '#0891b2',
  success: '#16a34a',
  warning: '#ca8a04',
  error: '#dc2626',
};

const darkBackground = '#0f172a';
const hexColorPattern = /^#[\da-fA-F]{6}$/;
const paletteVariableNames: Record<ThemeColorKey, string> = {
  primary: '--sd-primary',
  info: '--sd-info',
  success: '--sd-success',
  warning: '--sd-warning',
  error: '--sd-error',
};
const semanticColorVariableNames: Record<ThemeColorKey, string[]> = {
  primary: ['--qq-overlay_hover_brand', '--qq-tag_blue_bg'],
  info: [],
  success: ['--qq-tag_sage_green_bg'],
  warning: ['--qq-tag_orange_bg'],
  error: ['--qq-tag_red_bg'],
};
const colorTokenNames: Record<ThemeColorKey, {
  base: keyof NonNullable<GlobalThemeOverrides['common']>;
  hover: keyof NonNullable<GlobalThemeOverrides['common']>;
  pressed: keyof NonNullable<GlobalThemeOverrides['common']>;
  suppl: keyof NonNullable<GlobalThemeOverrides['common']>;
}> = {
  primary: {
    base: 'primaryColor',
    hover: 'primaryColorHover',
    pressed: 'primaryColorPressed',
    suppl: 'primaryColorSuppl',
  },
  info: {
    base: 'infoColor',
    hover: 'infoColorHover',
    pressed: 'infoColorPressed',
    suppl: 'infoColorSuppl',
  },
  success: {
    base: 'successColor',
    hover: 'successColorHover',
    pressed: 'successColorPressed',
    suppl: 'successColorSuppl',
  },
  warning: {
    base: 'warningColor',
    hover: 'warningColorHover',
    pressed: 'warningColorPressed',
    suppl: 'warningColorSuppl',
  },
  error: {
    base: 'errorColor',
    hover: 'errorColorHover',
    pressed: 'errorColorPressed',
    suppl: 'errorColorSuppl',
  },
};

export function isThemeColorKey(value: unknown): value is ThemeColorKey {
  return typeof value === 'string' && THEME_COLOR_KEYS.includes(value as ThemeColorKey);
}

export function isThemeHexColor(value: unknown): value is string {
  return typeof value === 'string' && hexColorPattern.test(value);
}

export function normalizeThemePalette(value: unknown): ThemePalette {
  if (!value || typeof value !== 'object') return { ...DEFAULT_THEME_PALETTE };

  const palette = { ...DEFAULT_THEME_PALETTE };
  const source = value as Partial<Record<ThemeColorKey, unknown>>;
  for (const key of THEME_COLOR_KEYS) {
    if (isThemeHexColor(source[key])) {
      palette[key] = source[key];
    }
  }
  return palette;
}

export function readStoredThemePalette(storage: ThemeStorage | undefined): ThemePalette {
  if (!storage) return { ...DEFAULT_THEME_PALETTE };

  try {
    const value = storage.getItem(THEME_PALETTE_STORAGE_KEY);
    return value ? normalizeThemePalette(JSON.parse(value)) : { ...DEFAULT_THEME_PALETTE };
  } catch {
    return { ...DEFAULT_THEME_PALETTE };
  }
}

export function writeStoredThemePalette(
  storage: ThemeStorage | undefined,
  palette: ThemePalette,
): void {
  if (!storage) return;

  try {
    storage.setItem(THEME_PALETTE_STORAGE_KEY, JSON.stringify(normalizeThemePalette(palette)));
  } catch {
    // localStorage 在隐私模式或嵌入环境里可能不可写，主题色失败时保持当前内存态即可。
  }
}

function getGeneratedColor(color: string, index: number, theme: ResolvedTheme): string {
  const colors = theme === 'dark'
    ? generate(color, { theme: 'dark', backgroundColor: darkBackground })
    : generate(color);
  return colors[index] ?? color;
}

function getSoftGeneratedColor(color: string, theme: ResolvedTheme): string {
  return getGeneratedColor(color, theme === 'dark' ? 1 : 0, theme);
}

function createStatusColorOverrides(
  key: ThemeColorKey,
  color: string,
  theme: ResolvedTheme,
): NonNullable<GlobalThemeOverrides['common']> {
  const tokenNames = colorTokenNames[key];

  return {
    [tokenNames.base]: color,
    [tokenNames.hover]: getGeneratedColor(color, 4, theme),
    [tokenNames.pressed]: getGeneratedColor(color, 6, theme),
    [tokenNames.suppl]: getGeneratedColor(color, 4, theme),
  };
}

function createCommonOverrides(
  palette: ThemePalette,
  theme: ResolvedTheme,
): NonNullable<GlobalThemeOverrides['common']> {
  const common = THEME_COLOR_KEYS.reduce<Record<string, string>>((result, key) => {
    return {
      ...result,
      ...createStatusColorOverrides(key, palette[key], theme),
    };
  }, {});

  if (theme === 'dark') {
    return {
      ...common,
      borderColor: '#334155',
      bodyColor: '#0f172a',
      cardColor: '#182133',
      modalColor: '#182133',
      popoverColor: '#182133',
    };
  }

  return common;
}

const sharedMenuOverrides: NonNullable<GlobalThemeOverrides['Menu']> = {
  itemTextColor: '#ffffff',
  itemTextColorHover: '#ffffff',
  itemTextColorActive: '#fcd34d',
  itemTextColorActiveHover: '#fcd34d',
  itemTextColorChildActive: '#fcd34d',
  itemTextColorChildActiveHover: '#fcd34d',
  itemIconColor: '#ffffff',
  itemIconColorHover: '#ffffff',
  itemIconColorActive: '#fcd34d',
  itemIconColorActiveHover: '#fcd34d',
  itemIconColorChildActive: '#fcd34d',
  itemIconColorChildActiveHover: '#fcd34d',
  // 侧栏使用深色背景，折叠后 Naive UI 会读取 collapsed 专用 token；这里显式保持白色避免回落成深色图标。
  itemIconColorCollapsed: '#ffffff',
  itemIconColorCollapsedInverted: '#ffffff',
  arrowColor: '#ffffff',
  arrowColorHover: '#ffffff',
  arrowColorActive: '#fcd34d',
  itemColorHover: 'rgba(67, 74, 84, 0.76)',
  itemColorActive: 'transparent',
  itemColorActiveHover: 'rgba(67, 74, 84, 0.76)',
  itemColorActiveCollapsed: 'transparent',
  borderRadius: '0',
};

const sharedLayoutOverrides: NonNullable<GlobalThemeOverrides['Layout']> = {
  color: 'var(--sd-bg-shell)',
  siderColor: 'var(--sd-bg-sidebar)',
  headerColor: 'var(--sd-bg-shell)',
  footerColor: 'var(--sd-bg-shell)',
  colorEmbedded: 'var(--sd-bg-page)',
};

export function createThemeOverrides(
  palette: ThemePalette,
  theme: ResolvedTheme,
): GlobalThemeOverrides {
  // Naive UI 组件主题只接收最终 token；项目壳层背景仍由 --sd-* 管，避免 provider 外层取不到变量。
  const overrides: GlobalThemeOverrides = {
    common: createCommonOverrides(palette, theme),
    Menu: sharedMenuOverrides,
    Layout: sharedLayoutOverrides,
  };

  if (theme === 'dark') {
    return {
      ...overrides,
      DataTable: {
        thColor: '#111827',
        tdColor: '#182133',
        tdColorHover: '#1f2a40',
        hoverColor: '#1f2a40',
        borderColor: '#334155',
      },
      Drawer: {
        color: '#182133',
      },
      Dropdown: {
        color: '#182133',
      },
    };
  }

  return overrides;
}

export function syncDocumentThemePalette(root: HTMLElement | undefined, palette: ThemePalette): void {
  if (!root) return;

  const theme = root.dataset?.theme === 'dark' ? 'dark' : 'light';

  // 这里同步的是项目级语义色，Tailwind 和少量自定义 CSS 可以直接读这些变量。
  for (const key of THEME_COLOR_KEYS) {
    root.style.setProperty(paletteVariableNames[key], palette[key]);

    const softColor = getSoftGeneratedColor(palette[key], theme);
    for (const variableName of semanticColorVariableNames[key]) {
      root.style.setProperty(variableName, softColor);
    }
  }
}
