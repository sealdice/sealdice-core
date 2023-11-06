import path from "path";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import legacy from "@vitejs/plugin-legacy";

import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";
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
  plugins: [
    vue(),
    Components({
      resolvers: [
        ElementPlusResolver({
          importStyle: "sass",
        }),
        IconsResolver(),
      ],
      dts: path.resolve(pathSrc, "components.d.ts"),
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
    target: "es6",
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks: {
          base: ["vue", "pinia"],
          library: ["element-plus", "ofetch", "axios", "axios-retry"],
          util: ["vue-diff", "vuedraggable", "dayjs", "lodash-es"],
        },
      },
    },
  },
});
