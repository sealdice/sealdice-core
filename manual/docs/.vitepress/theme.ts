import type { DefaultTheme } from 'vitepress';
import { about, advanced, config, deployNav, deploySidebar, useNav, useSidebar } from "./catalogue";

export const theme: DefaultTheme.Config = {
  // https://vitepress.dev/reference/default-theme-config
  logo: {
    light: '/images/sealdice.svg',
    dark: '/images/sealdice-dark.svg',
  },
  nav: [
    {
      text: "官网",
      link: "https://dice.weizaima.com/",
    },
    {
      text: "首页",
      link: "/",
    },
    deployNav,
    config,
    useNav,
    advanced,
    about,
  ] as DefaultTheme.NavItem[],
  sidebar: {
    "/deploy/": deploySidebar,
    "/config/": config,
    "/use/": useSidebar,
    "/advanced/": advanced,
    "/about/": about,
    "/archive/": about,
  } as DefaultTheme.SidebarMulti,
  outline: {
    label: '页面导航',
    level: [2, 3],
  },
  socialLinks: [
    { icon: 'github', link: 'https://github.com/sealdice' }
  ],
  lastUpdated: {
    text: '上次更新于',
    formatOptions: {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    },
  },
  docFooter: {
    prev: '上一节',
    next: '下一节'
  },
  darkModeSwitchLabel: '主题',
  lightModeSwitchTitle: '切换到浅色模式',
  darkModeSwitchTitle: '切换到深色模式',
  sidebarMenuLabel: '菜单',
  returnToTopLabel: '返回顶部',
  search: {
    provider: 'local',
    options: {
      detailedView: true,
      translations: {
        button: {
          buttonText: '搜索文档',
          buttonAriaLabel: '搜索文档'
        },
        modal: {
          displayDetails: '显示详细信息',
          resetButtonTitle: '清除查询条件',
          backButtonTitle: '返回',
          noResultsText: '无法找到相关结果',
          footer: {
            selectText: '选择',
            navigateText: '切换',
            closeText: '关闭',
          }
        }
      },
    } as DefaultTheme.LocalSearchOptions
  },
}
