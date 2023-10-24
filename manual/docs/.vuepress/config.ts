import { defineUserConfig } from "vuepress";
import { searchProPlugin } from "vuepress-plugin-search-pro";
import theme from "./theme";

export default defineUserConfig({
  base: "/sealdice-manual-next/",
  lang: "zh-CN",
  title: "海豹手册",
  description: "海豹核心的全新官方使用手册",

  theme,

  plugins: [
    searchProPlugin({
      indexConetnt: true,
      autoSuggestions: true,
    }),
  ],
});
