import { fileURLToPath, URL } from 'node:url'
import tailwindcss from 'tailwindcss'
import autoprefixer from 'autoprefixer'
import postcssColorMix from '@csstools/postcss-color-mix-function'

import { defineConfig } from 'vite'
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
export default defineConfig(({ mode }) => ({
  base: resolveVitePublicBase(mode),
  plugins: [
    VueRouter({
      routesFolder: 'src/pages',
      dts: 'typed-router.d.ts',
      importMode: 'async',
    }),
    vue(),
    vueJsx(),
    vueDevTools(),
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
      // Keep a single module build and let plugin-legacy inject the full
      // modern polyfill bundle for the supported browser matrix below.
      modernPolyfills: true,
      modernTargets: ['Chrome >= 78',
        "Firefox >= 67",
        'Safari >= 14'],
      // We do not ship nomodule/SystemJS bundles anymore; all supported users
      // stay on the module path and receive the modern polyfill bundle.
      renderLegacyChunks: false,
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
  css: {
    postcss: {
      plugins: [
        tailwindcss(),
        postcssColorMix({
          preserve: true,
        }),
        autoprefixer(),
      ],
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
    chunkSizeWarningLimit: 650,
    ...resolveViteBuildOptions(mode),
    rollupOptions: {
      onwarn(warning, defaultHandler) {
        if (shouldSuppressRollupWarning(warning)) return
        defaultHandler(warning)
      },
      output: {
        manualChunks: classifyVendorChunk,
      },
    },
  },
}))
