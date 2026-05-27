import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import type { PluginOption } from 'vite'
import VueRouter from 'vue-router/vite'
import legacy from '@vitejs/plugin-legacy'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import { VitePWA } from 'vite-plugin-pwa'
import AutoImport from 'unplugin-auto-import/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import { ProNaiveUIResolver } from 'pro-naive-ui-resolver'
import Components from 'unplugin-vue-components/vite'
import Icons from 'unplugin-icons/vite'
import IconsResolver from 'unplugin-icons/resolver'
import { visualizer } from 'rollup-plugin-visualizer'
import { classifyVendorChunk } from './src/build/chunkPolicy'
import { resolveViteBuildOptions, resolveVitePublicBase } from './src/build/embedConfig'
import { shouldSuppressRollupWarning } from './src/build/warningPolicy'

const DEFAULT_PROXY_TARGET = 'http://localhost:3211'
const apiProxyTarget = (
  process.env.VITE_API_PROXY_TARGET ||
  process.env.DEV_PROXY_SERVER ||
  process.env.VITE_API_BASE_URL ||
  DEFAULT_PROXY_TARGET
).trim()

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const plugins: PluginOption[] = [
    VueRouter({
      routesFolder: 'src/pages',
      dts: 'typed-router.d.ts',
      importMode: 'async',
    }),
    vue(),
    vueJsx(),
    mode === 'development' ? vueDevTools() : null,
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['favicon.ico', 'pwa-192.svg', 'pwa-512.svg', 'pwa-maskable.svg'],
      workbox: {
        globIgnores: ['**/stats.html'],
      },
      manifest: {
        name: 'SealDice 控制台',
        short_name: 'SealDice',
        description: 'SealDice 核心管理控制台',
        theme_color: '#0f172a',
        background_color: '#0f172a',
        display: 'standalone',
        start_url: './',
        scope: './',
        icons: [
          {
            src: './pwa-192.svg',
            sizes: '192x192',
            type: 'image/svg+xml',
            purpose: 'any',
          },
          {
            src: './pwa-512.svg',
            sizes: '512x512',
            type: 'image/svg+xml',
            purpose: 'any',
          },
          {
            src: './pwa-maskable.svg',
            sizes: '512x512',
            type: 'image/svg+xml',
            purpose: 'maskable',
          },
        ],
      },
      devOptions: {
        enabled: true,
      },
    }),
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
      // 补充Pro-naive-ui自动引入
      resolvers: [NaiveUiResolver(), ProNaiveUIResolver(), IconsResolver()]
    }),
    Icons({
      compiler: 'vue3',
      autoInstall: true
    }),
    legacy({
      // 用Chrome79测了一下好像问题不算特别大的样子，有些功能降级，但能用
      // 感觉这就很极端的场景了，现在连2345浏览器都上100+了
      targets: ['Chrome >= 78', 'Firefox >= 67', 'Safari >= 14'],
      renderLegacyChunks: false // 这个已经是老古董级别的了，海豹啥的都不支持，别费心了还得打包一份额外文件
    }),
    mode === 'analyze' ? visualizer({
      filename: 'stats.html',
      emitFile: true,
      title: 'SealDice UI Bundle',
      gzipSize: true,
      brotliSize: true,
      template: 'treemap',
    }) : null,
  ]

  return {
    base: resolveVitePublicBase(mode),
    build: {
      chunkSizeWarningLimit: 650,
      ...resolveViteBuildOptions(mode),
      rolldownOptions: {
        onLog(level, log, defaultHandler) {
          if (log.code === 'CIRCULAR_DEPENDENCY') {
            return; // Ignore circular dependency warnings
          }
          if (level === 'warn') {
            if (shouldSuppressRollupWarning(log)) return
            defaultHandler('error',log)
          } else {
            defaultHandler(level, log); // otherwise, just print the log
          }
        },
        output: {
          codeSplitting: {
            groups: [
              {
                name(moduleId) {
                  classifyVendorChunk(moduleId)
                },
              },
            ],
          },
        },
      },
    },
    plugins,
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
  }
})
