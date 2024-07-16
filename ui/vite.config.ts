import path from "path";
import {defineConfig} from "vite";
import vue from "@vitejs/plugin-vue";
import vueJsx from '@vitejs/plugin-vue-jsx'
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
    vueJsx(),
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
          base: ["vue", "pinia", "vue-router"],
          codemirror: ["codemirror", "@codemirror/lang-javascript"],
          common: [
            "element-plus",
            "lodash-es",
          ],
          utils: [
            "@vueuse/core",
            "asmcrypto.js",
            "axios",
            "axios-retry",
            "clipboard",
            "dayjs",
            "filesize",
            "randomcolor",
            "vue-diff",
            "vuedraggable",
          ],
        },
      },
    },
  },
});
