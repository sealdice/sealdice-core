import { defineUserConfig } from "vuepress";
import { searchProPlugin } from "vuepress-plugin-search-pro";
import theme from "./theme";

console.log('xxxxxxxxx', process.env.BASE_PATH)
const basePath = process.env.BASE_PATH ?? "/sealdice-manual-next/";

export default defineUserConfig({
  base: basePath,
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
