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
          importStyle: "css",
        }),
        IconsResolver(),
      ],
    }),
    Components({
      resolvers: [
        ElementPlusResolver({
          importStyle: "css",
        }),
        IconsResolver(),
      ],
    }),
    Icons({
      compiler: "vue3",
      autoInstall: true,
    }),
    legacy({
      targets: ["defaults"],
    }),
  ],
  build: {
    sourcemap: false,
    chunkSizeWarningLimit: 1024,
    rollupOptions: {
      output: {
        manualChunks: {
          base: ["vue", "pinia", "vue-router"],
          element: ["element-plus"],
          codemirror: ["codemirror", "@codemirror/lang-javascript"],
          utils: [
            "@vueuse/core",
            "asmcrypto.js",
            "axios",
            "axios-retry",
            "clipboard",
            "dayjs",
            "filesize",
            "lodash-es",
            "randomcolor",
            "vue-diff",
            "vuedraggable",
          ],
        },
      },
    },
  },
});
