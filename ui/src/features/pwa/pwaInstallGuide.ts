export type PwaPlatform = 'android' | 'ios' | 'linux' | 'macos' | 'windows' | 'other';
export type PwaBrowser = 'chromium' | 'firefox' | 'safari' | 'other';

export interface PwaInstallEnvironment {
  platform: PwaPlatform;
  browser: PwaBrowser;
  isSecureContext: boolean;
  isDevelopment: boolean;
}

export interface PwaInstallGuide {
  title: string;
  description: string;
  steps: string[];
  warning?: string;
}

export function detectPwaPlatform(
  userAgent: string,
  navigatorPlatform = '',
  maxTouchPoints = 0
): PwaPlatform {
  if (/Android/i.test(userAgent)) return 'android';
  if (/iPhone|iPad|iPod/i.test(userAgent)) return 'ios';
  if (/Mac/i.test(navigatorPlatform) && maxTouchPoints > 1) return 'ios';
  if (/Windows/i.test(userAgent) || /Win/i.test(navigatorPlatform)) return 'windows';
  if (/Macintosh|Mac OS X/i.test(userAgent) || /Mac/i.test(navigatorPlatform)) return 'macos';
  if (/Linux/i.test(userAgent) || /Linux/i.test(navigatorPlatform)) return 'linux';
  return 'other';
}

export function detectPwaBrowser(userAgent: string): PwaBrowser {
  if (/Firefox|FxiOS/i.test(userAgent)) return 'firefox';
  if (/Edg|Chrome|Chromium|CriOS|OPR|SamsungBrowser/i.test(userAgent)) return 'chromium';
  if (/Safari/i.test(userAgent)) return 'safari';
  return 'other';
}

export function getPwaInstallEnvironment(): PwaInstallEnvironment {
  const userAgent = typeof navigator === 'undefined' ? '' : navigator.userAgent;
  const navigatorPlatform = typeof navigator === 'undefined' ? '' : navigator.platform;
  const maxTouchPoints = typeof navigator === 'undefined' ? 0 : navigator.maxTouchPoints;

  return {
    platform: detectPwaPlatform(userAgent, navigatorPlatform, maxTouchPoints),
    browser: detectPwaBrowser(userAgent),
    isSecureContext: typeof window !== 'undefined' && window.isSecureContext,
    isDevelopment: import.meta.env.DEV,
  };
}

export function buildPwaInstallGuide(environment: PwaInstallEnvironment): PwaInstallGuide {
  if (environment.isDevelopment) {
    return {
      title: '开发环境未启用应用安装',
      description:
        '当前页面来自 Vite 开发服务器，未注册生产版 Web App Manifest 和 Service Worker，因此 Chrome 不会提供安装提示。',
      steps: ['构建前端生产版本。', '通过 SealDice 的生产服务地址重新打开页面并安装。'],
    };
  }

  if (!environment.isSecureContext) {
    return {
      title: '当前地址不满足完整安装条件',
      description: '浏览器只允许 HTTPS、localhost 或 127.0.0.1 使用完整的 PWA 安装与离线能力。',
      steps: ['为 SealDice 配置 HTTPS 反向代理。', '使用 HTTPS 地址重新打开页面后再安装。'],
      warning: '浏览器菜单可能仍允许创建普通桌面快捷方式，但它不等同于完整安装。',
    };
  }

  if (environment.platform === 'ios') {
    return {
      title: '在 iPhone 或 iPad 上安装',
      description: 'iOS 和 iPadOS 不提供网页内安装弹窗，需要从浏览器的分享菜单添加。',
      steps: ['打开浏览器的“分享”菜单。', '选择“添加到主屏幕”。', '确认名称后点击“添加”。'],
    };
  }

  if (environment.platform === 'macos' && environment.browser === 'safari') {
    return {
      title: '在 Safari 中安装',
      description: 'Safari 17 及以上版本可将网站作为应用添加到程序坞。',
      steps: ['打开 Safari 的“文件”菜单。', '选择“添加到程序坞”。', '确认名称与图标后完成添加。'],
    };
  }

  if (environment.browser === 'firefox' && environment.platform !== 'android') {
    return {
      title: '桌面 Firefox 暂不支持安装',
      description: '桌面 Firefox 当前不支持将 Web App Manifest 网站安装为独立 PWA。',
      steps: [
        'Windows 或 Linux 可改用 Chrome、Edge 等 Chromium 浏览器。',
        'macOS 也可使用 Safari 的“添加到程序坞”。',
      ],
    };
  }

  if (environment.platform === 'android') {
    return {
      title: '在 Android 上安装',
      description: '当前浏览器没有提供网页内安装弹窗，可以从浏览器菜单手动安装。',
      steps: ['打开浏览器菜单。', '选择“安装应用”或“添加到主屏幕”。', '按浏览器提示完成安装。'],
    };
  }

  if (environment.browser === 'chromium') {
    return {
      title: 'Chrome 尚未提供安装提示',
      description:
        '页面当前没有收到浏览器的安装事件。应用可能已安装，或 Chrome 尚未完成安装条件检查。',
      steps: [
        '等待页面加载完成后再试一次。',
        '也可以打开浏览器菜单，在“投放、保存和分享”中选择“安装 SealDice”。',
      ],
    };
  }

  return {
    title: '从浏览器菜单安装',
    description: '当前浏览器没有提供网页内安装弹窗，请使用其内置菜单完成安装。',
    steps: ['打开浏览器主菜单。', '查找“安装应用”“添加到主屏幕”或类似选项。'],
  };
}
