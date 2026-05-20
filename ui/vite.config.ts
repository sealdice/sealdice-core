import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import VueRouter from 'vue-router/vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import AutoImport from 'unplugin-auto-import/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import Components from 'unplugin-vue-components/vite'
import Icons from 'unplugin-icons/vite'
import IconsResolver from 'unplugin-icons/resolver'
import { visualizer } from 'rollup-plugin-visualizer'

const DEFAULT_PROXY_TARGET = 'http://localhost:3211'
const apiProxyTarget = (
  process.env.VITE_API_PROXY_TARGET ||
  process.env.DEV_PROXY_SERVER ||
  process.env.VITE_API_BASE_URL ||
  DEFAULT_PROXY_TARGET
).trim()

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  base: './',
  plugins: [
    VueRouter({
      routesFolder: 'src/pages',
      dts: 'typed-router.d.ts',
      importMode: 'async',
    }),
    vue(),
    vueJsx(),
    vueDevTools(),
    AutoImport({
      imports: [
        // 'vue', // 感觉vue自动引入有点乱，还是手动吧
        {
          'naive-ui': [
            'useDialog',
            'useMessage',
            'useNotification',
            'useLoadingBar'
          ]
        }
      ],
      resolvers: [IconsResolver()]
    }),
    Components({
      resolvers: [NaiveUiResolver(), IconsResolver()]
    }),
    Icons({
      compiler: 'vue3',
      autoInstall: true
    }),
    mode === 'analyze' && visualizer({
      filename: 'stats.html',
      emitFile: true,
      title: 'SealDice UI Bundle',
      gzipSize: true,
      brotliSize: true,
      template: 'treemap',
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5175,
    strictPort: true,
    proxy: {
      '^/api': {
        target: apiProxyTarget,
        changeOrigin: true,
        xfwd: true,
      },
      '^/sd-api': {
        target: apiProxyTarget,
        changeOrigin: true,
        ws: true,
        xfwd: true,
      },
      '^/openapi.json$': {
        target: apiProxyTarget,
        changeOrigin: true,
        xfwd: true,
      },
      '^/docs(?:/.*)?$': {
        target: apiProxyTarget,
        changeOrigin: true,
        xfwd: true,
      },
      '^/schemas(?:/.*)?$': {
        target: apiProxyTarget,
        changeOrigin: true,
        xfwd: true,
      },
    },
  },
  build: {
    target: 'chrome90',
    cssTarget: 'chrome90',
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('/node_modules/')) return undefined
          if (
            id.includes('/vue/') ||
            id.includes('/@vue/') ||
            id.includes('/vue-router/') ||
            id.includes('/@tanstack/query-core/') ||
            id.includes('/@tanstack/vue-query/')
          ) {
            return 'vendor-framework'
          }
          return undefined
        },
      },
    },
  },
}))
