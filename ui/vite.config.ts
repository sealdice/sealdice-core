import path from "path";
import {defineConfig} from "vite";
import vue from "@vitejs/plugin-vue";
import legacy from "@vitejs/plugin-legacy";

import AutoImport from 'unplugin-auto-import/vite'
import Components from "unplugin-vue-components/vite";
import {ElementPlusResolver} from "unplugin-vue-components/resolvers";
import Icons from "unplugin-icons/vite";
import IconsResolver from "unplugin-icons/resolver";

const pathSrc = path.resolve(__dirname, "src");

// https://vitejs.dev/config/
export default defineConfig({
  base: "./",
  resolve: {
    alias: {
      "~/": `${pathSrc}/`,
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        additionalData: `@use "~/styles/element/index.scss" as *;`,
      },
    },
  },
  server: {
    port: 3000,
  },
  plugins: [
    vue(),
    AutoImport({
      include: [
        /\.[tj]sx?$/, // .ts, .tsx, .js, .jsx
        /\.vue$/,
        /\.vue\?vue/, // .vue
      ],
      imports: ["vue", "pinia", "vue-router", "@vueuse/core"],
      dts: true,
      vueTemplate: true,
      eslintrc: {
        enabled: true,
      },
      resolvers: [
        ElementPlusResolver({
          importStyle: "sass",
        }),
        IconsResolver(),
      ],
    }),
    Components({
      resolvers: [
        ElementPlusResolver({
          importStyle: "sass",
        }),
        IconsResolver(),
      ],
    }),
    Icons({
      compiler: "vue3",
      autoInstall: true,
    }),
    legacy({
      targets: ["defaults", "not IE 11"],
    }),
  ],
  build: {
    sourcemap: false,
    chunkSizeWarningLimit: 1024,
    rollupOptions: {
      output: {
        manualChunks: {
          base: ["vue", "pinia"],
          element: ["element-plus"],
          lodash: ["lodash-es"],
          codemirror: ["codemirror", "@codemirror/lang-javascript"],
          network: ["ofetch", "axios", "axios-retry", "ky"],
          util: [
            "@vueuse/core",
            "asmcrypto.js",
            "clipboard",
            "dayjs",
            "filesize",
            "vue-diff",
            "vuedraggable",
          ],
        },
      },
    },
  },
});
