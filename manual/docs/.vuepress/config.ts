import { defineUserConfig } from "vuepress";
import { searchProPlugin } from "vuepress-plugin-search-pro";
import theme from "./theme";

const basePath: any = process.env.BASE_PATH ?? "/sealdice-manual-next/";

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
