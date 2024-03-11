import { defineUserConfig } from "vuepress";
import viteBundler from "@vuepress/bundler-vite";
import { registerComponentsPlugin } from "@vuepress/plugin-register-components";
// @ts-ignore
import { path } from "vuepress/utils";
import theme from "./theme";

const basePath: any = process.env.BASE_PATH ?? "/sealdice-manual-next/";

export default defineUserConfig({
  base: basePath,
  lang: "zh-CN",
  title: "海豹手册",
  description: "海豹核心的全新官方使用手册",

  theme,

  plugins: [
    registerComponentsPlugin({
      components: {
        ChatBox: path.resolve(__dirname, "./components/ChatBox.vue"),
      },
    }),
  ],

  bundler: viteBundler({
    viteOptions: {},
    vuePluginOptions: {},
  })
});
