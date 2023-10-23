import { defaultTheme, defineUserConfig } from 'vuepress'
import { backToTopPlugin } from '@vuepress/plugin-back-to-top'
import { searchPlugin } from '@vuepress/plugin-search'
import { mdEnhancePlugin } from 'vuepress-plugin-md-enhance'

export default defineUserConfig({
  base: '/sealdice-manual-next/',
  lang: 'zh-cn',
  title: '海豹手册',
  description: '海豹核心的全新官方使用手册',
  theme: defaultTheme({
    logo: '/images/sealdice.ico',
    home: '/index.md',
    navbar: [
      {
        text: '首页',
        link: '/',
      },
      {
        text: '部署',
        children: [
          '/deploy/quick-start.md',
        ],
      },
      {
        text: '配置',
        children: [
          '/config/censor.md',
        ],
      },
      {
        text: '使用',
        children: [
          '/use/introduce.md',
          '/use/quick-start.md',
          '/use/core.md',
          '/use/coc7.md',
          '/use/dnd5e.md',
          '/use/sr.md',
        ],
      },
      {
        text: '进阶',
        children: [
          '/advanced/introduce.md',
          '/advanced/decks.md',
        ],
      },
      {
        text: '开发',
        children: [
          '/develop/develop.md',
        ],
      },
    ],
    repo: 'sealdice/sealdice-core',
    editLink: false,
    lastUpdatedText: '最近更新',
    contributors: false,
  }),
  plugins: [
    backToTopPlugin(),
    searchPlugin({
      locales: {
        '/': {
          placeholder: '搜索',
        },
        '/en': {
          placeholder: 'Search',
        },
      },
    }),
    mdEnhancePlugin({
      container: true,
      tabs: true,
    }),
  ],
})