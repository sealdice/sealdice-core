import { generate } from '@ant-design/colors';
import type { GlobalThemeOverrides } from 'naive-ui';
import type { ResolvedTheme } from './themeState';

export type ThemeColorKey = 'primary' | 'success' | 'warning' | 'error';
export type ThemePalette = Record<ThemeColorKey, string>;

type ComponentColorKey = ThemeColorKey | 'info';

export const THEME_COLOR_KEYS: ThemeColorKey[] = ['primary', 'success', 'warning', 'error'];

// 应用仅维护四种有彩语义。info 是组件库兼容名称，始终映射到 primary。
export const DEFAULT_THEME_PALETTE: ThemePalette = {
  primary: '#2563eb',
  success: '#15803d',
  warning: '#b45309',
  error: '#dc2626',
};

const componentColorKeys: ComponentColorKey[] = ['primary', 'info', 'success', 'warning', 'error'];
const darkBackground = '#0f172a';

const colorTokenNames: Record<
  ComponentColorKey,
  {
    base: keyof NonNullable<GlobalThemeOverrides['common']>;
    hover: keyof NonNullable<GlobalThemeOverrides['common']>;
    pressed: keyof NonNullable<GlobalThemeOverrides['common']>;
    suppl: keyof NonNullable<GlobalThemeOverrides['common']>;
  }
> = {
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

function getPaletteColor(key: ComponentColorKey): string {
  return key === 'info' ? DEFAULT_THEME_PALETTE.primary : DEFAULT_THEME_PALETTE[key];
}

function getGeneratedColor(color: string, index: number, theme: ResolvedTheme): string {
  const colors =
    theme === 'dark'
      ? generate(color, { theme: 'dark', backgroundColor: darkBackground })
      : generate(color);
  return colors[index] ?? color;
}

function hexToRgb(color: string): [number, number, number] {
  return [
    Number.parseInt(color.slice(1, 3), 16),
    Number.parseInt(color.slice(3, 5), 16),
    Number.parseInt(color.slice(5, 7), 16),
  ];
}

function rgbToHex(rgb: [number, number, number]): string {
  return `#${rgb.map(value => Math.round(value).toString(16).padStart(2, '0')).join('')}`;
}

function mixHex(start: string, end: string, ratio: number): string {
  const startRgb = hexToRgb(start);
  const endRgb = hexToRgb(end);
  return rgbToHex(
    startRgb.map((value, index) => value + (endRgb[index] - value) * ratio) as [
      number,
      number,
      number,
    ]
  );
}

function relativeLuminance(color: string): number {
  return hexToRgb(color)
    .map(channel => channel / 255)
    .map(channel => (channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4))
    .reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0);
}

function contrastRatio(foreground: string, background: string): number {
  const foregroundLuminance = relativeLuminance(foreground);
  const backgroundLuminance = relativeLuminance(background);
  const lighter = Math.max(foregroundLuminance, backgroundLuminance);
  const darker = Math.min(foregroundLuminance, backgroundLuminance);
  return (lighter + 0.05) / (darker + 0.05);
}

function getAccessibleSemanticColor(color: string, theme: ResolvedTheme): string {
  const background = theme === 'dark' ? '#182133' : '#ffffff';
  const target = theme === 'dark' ? '#ffffff' : '#111827';
  if (contrastRatio(color, background) >= 4.5) return color;

  for (let step = 1; step <= 20; step += 1) {
    const candidate = mixHex(color, target, step / 20);
    if (contrastRatio(candidate, background) >= 4.5) return candidate;
  }
  return target;
}

function getButtonTextColor(background: string): string {
  return contrastRatio('#ffffff', background) >= contrastRatio('#111827', background)
    ? '#ffffff'
    : '#111827';
}

function createStatusColorOverrides(
  key: ComponentColorKey,
  theme: ResolvedTheme
): NonNullable<GlobalThemeOverrides['common']> {
  const color = getPaletteColor(key);
  const tokenNames = colorTokenNames[key];
  const accessibleColor = getAccessibleSemanticColor(color, theme);
  const accessibleHoverColor = getAccessibleSemanticColor(
    getGeneratedColor(color, 4, theme),
    theme
  );
  const accessiblePressedColor = getAccessibleSemanticColor(
    getGeneratedColor(color, 6, theme),
    theme
  );

  return {
    [tokenNames.base]: accessibleColor,
    [tokenNames.hover]: accessibleHoverColor,
    [tokenNames.pressed]: accessiblePressedColor,
    [tokenNames.suppl]: accessibleHoverColor,
  };
}

function createButtonColorOverrides(
  key: ComponentColorKey,
  theme: ResolvedTheme
): Record<string, string> {
  const color = getPaletteColor(key);
  const typeName = `${key[0].toUpperCase()}${key.slice(1)}`;
  const accessibleColor = getAccessibleSemanticColor(color, theme);
  const accessibleHoverColor = getAccessibleSemanticColor(
    getGeneratedColor(color, 4, theme),
    theme
  );
  const accessiblePressedColor = getAccessibleSemanticColor(
    getGeneratedColor(color, 6, theme),
    theme
  );
  const buttonTextColor = getButtonTextColor(accessibleColor);
  const buttonHoverTextColor = getButtonTextColor(accessibleHoverColor);
  const buttonPressedTextColor = getButtonTextColor(accessiblePressedColor);

  return {
    [`color${typeName}`]: accessibleColor,
    [`colorHover${typeName}`]: accessibleHoverColor,
    [`colorPressed${typeName}`]: accessiblePressedColor,
    [`colorFocus${typeName}`]: accessibleHoverColor,
    [`colorDisabled${typeName}`]: accessibleColor,
    [`textColor${typeName}`]: buttonTextColor,
    [`textColorHover${typeName}`]: buttonHoverTextColor,
    [`textColorPressed${typeName}`]: buttonPressedTextColor,
    [`textColorFocus${typeName}`]: buttonHoverTextColor,
    [`textColorDisabled${typeName}`]: buttonTextColor,
    [`textColorText${typeName}`]: accessibleColor,
    [`textColorTextHover${typeName}`]: accessibleHoverColor,
    [`textColorTextPressed${typeName}`]: accessiblePressedColor,
    [`textColorTextFocus${typeName}`]: accessibleHoverColor,
    [`textColorTextDisabled${typeName}`]: accessibleColor,
    [`textColorGhost${typeName}`]: accessibleColor,
    [`textColorGhostHover${typeName}`]: accessibleHoverColor,
    [`textColorGhostPressed${typeName}`]: accessiblePressedColor,
    [`textColorGhostFocus${typeName}`]: accessibleHoverColor,
    [`textColorGhostDisabled${typeName}`]: accessibleColor,
  } as Record<string, string>;
}

function createCommonOverrides(theme: ResolvedTheme): NonNullable<GlobalThemeOverrides['common']> {
  const common = componentColorKeys.reduce<Record<string, string>>(
    (result, key) => {
      return {
        ...result,
        ...createStatusColorOverrides(key, theme),
      };
    },
    {
      borderRadius: 'var(--sd-radius-sm)',
      borderRadiusSmall: 'var(--sd-radius-xs)',
    }
  );

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
  itemTextColor: 'var(--sd-text-inverse-soft)',
  itemTextColorHover: 'var(--sd-text-inverse)',
  itemTextColorActive: 'var(--sd-text-inverse)',
  itemTextColorActiveHover: 'var(--sd-text-inverse)',
  itemTextColorChildActive: 'var(--sd-text-inverse)',
  itemTextColorChildActiveHover: 'var(--sd-text-inverse)',
  itemIconColor: 'var(--sd-text-inverse-muted)',
  itemIconColorHover: 'var(--sd-text-inverse)',
  itemIconColorActive: 'var(--sd-text-inverse)',
  itemIconColorActiveHover: 'var(--sd-text-inverse)',
  itemIconColorChildActive: 'var(--sd-text-inverse)',
  itemIconColorChildActiveHover: 'var(--sd-text-inverse)',
  itemIconColorCollapsed: 'var(--sd-text-inverse)',
  itemIconColorCollapsedInverted: 'var(--sd-text-inverse)',
  arrowColor: 'var(--sd-text-inverse-muted)',
  arrowColorHover: 'var(--sd-text-inverse)',
  arrowColorActive: 'var(--sd-text-inverse)',
  arrowColorChildActive: 'var(--sd-text-inverse-soft)',
  arrowColorChildActiveHover: 'var(--sd-text-inverse)',
  itemColorHover: 'var(--sd-bg-sidebar-hover)',
  itemColorActive: 'var(--sd-bg-sidebar-selected)',
  itemColorActiveHover: 'var(--sd-bg-sidebar-selected-strong)',
  itemColorActiveCollapsed: 'var(--sd-bg-sidebar-selected)',
  borderRadius: 'var(--sd-radius-sm)',
};

const sharedLayoutOverrides: NonNullable<GlobalThemeOverrides['Layout']> = {
  color: 'var(--sd-bg-shell)',
  siderColor: 'var(--sd-bg-sidebar)',
  headerColor: 'var(--sd-bg-shell)',
  footerColor: 'var(--sd-bg-shell)',
  colorEmbedded: 'var(--sd-bg-page)',
};

export function createThemeOverrides(theme: ResolvedTheme): GlobalThemeOverrides {
  const overrides: GlobalThemeOverrides = {
    common: createCommonOverrides(theme),
    Button: componentColorKeys.reduce<Record<string, string>>(
      (result, key) => ({
        ...result,
        ...createButtonColorOverrides(key, theme),
      }),
      {}
    ) as NonNullable<GlobalThemeOverrides['Button']>,
    Menu: sharedMenuOverrides,
    Layout: sharedLayoutOverrides,
  };

  if (theme === 'dark') {
    return {
      ...overrides,
      DataTable: {
        thColor: 'var(--sd-bg-elevated-soft)',
        tdColor: 'var(--sd-bg-elevated)',
        tdColorHover: 'var(--sd-bg-control)',
        hoverColor: 'var(--sd-bg-control)',
        borderColor: 'var(--sd-border)',
      },
      Drawer: {
        color: 'var(--sd-bg-elevated)',
      },
      Dropdown: {
        color: 'var(--sd-bg-elevated)',
      },
    };
  }

  return overrides;
}

export function syncDocumentThemePalette(root: HTMLElement | undefined): void {
  if (!root) return;

  const theme = root.dataset?.theme === 'dark' ? 'dark' : 'light';
  const resolvedColors = Object.fromEntries(
    THEME_COLOR_KEYS.map(key => [
      key,
      getAccessibleSemanticColor(DEFAULT_THEME_PALETTE[key], theme),
    ])
  ) as ThemePalette;

  for (const key of THEME_COLOR_KEYS) {
    root.style.setProperty(`--sd-${key}`, resolvedColors[key]);
  }

  // 兼容组件库的 info API；它不是独立语义色。
  root.style.setProperty('--sd-info', resolvedColors.primary);
  root.style.setProperty('--sd-accent', resolvedColors.primary);
  root.style.setProperty(
    '--sd-accent-strong',
    getAccessibleSemanticColor(getGeneratedColor(DEFAULT_THEME_PALETTE.primary, 6, theme), theme)
  );
}
