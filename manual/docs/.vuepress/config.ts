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
        text: '常见问题',
        link: '/faq.md',
        activeMatch: '/faq.html'
      },
      {
        text: '部署',
        children: [
          '/deploy/quick-start.md',
          '/deploy/transfer.md',
        ],
      },
      {
        text: '配置',
        children: [
          '/config/custom_text.md',
          '/config/reply.md',
          '/config/deck.md',
          '/config/jsscript.md',
          '/config/helpdoc.md',
          '/config/censor.md',
        ],
      },
      {
        text: '使用',
        children: [
          '/use/introduce.md',
          '/use/quick-start.md',
          '/use/special_feature.md',
          '/use/core.md',
          '/use/helper.md',
          '/use/coc7.md',
          '/use/dnd5e.md',
          '/use/story.md',
          '/use/log.md',
          '/use/fun.md',
          '/use/deck_and_reply.md',
          '/use/sr.md',
        ],
      },
      {
        text: '进阶',
        children: [
          '/advanced/introduce.md',
          '/advanced/script.md',
          '/advanced/edit_complex_custom_text.md',
          '/advanced/edit_reply.md',
          '/advanced/edit_deck.md',
          '/advanced/edit_jsscript.md',
          '/advanced/edit_helpdoc.md',
          '/advanced/edit_sensitive_words.md',
        ],
      },
      {
        text: '开发',
        children: [
          '/develop/develop.md',
        ],
      },
      {
        text: '关于',
        link: '/about.md',
        activeMatch: '/about.html'
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
      figure: true,
      imgLazyload: true,
      imgMark: true,
      imgSize: true,
    }),
  ],
})