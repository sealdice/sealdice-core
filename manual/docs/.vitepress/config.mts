import { defineConfig } from 'vitepress'
import { tabsMarkdownPlugin } from "vitepress-plugin-tabs"
import { theme } from "./theme"

const base: any = process.env.BASE_PATH ?? "/sealdice-manual-next/";

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: '海豹手册',
  description: '海豹骰官方使用手册',
  head: [
    ['link', { rel: 'icon', href: '/images/sealdice.svg' }]
  ],
  lang: 'zh-CN',
  base,
  lastUpdated: true,
  themeConfig: theme,
  markdown: {
    container: {
      tipLabel: '提示',
      warningLabel: '注意',
      dangerLabel: '危险',
      infoLabel: '信息',
      detailsLabel: '补充'
    },
    config(md) {
      md.use(tabsMarkdownPlugin)
    }
  },
  vite: {}
})
