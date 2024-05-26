import path from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import legacy from '@vitejs/plugin-legacy'

import Components from 'unplugin-vue-components/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'

const pathSrc = path.resolve(__dirname, 'src')

// https://vitejs.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '~/': `${pathSrc}/`,
    },
  },
  base: './',
  plugins: [
    vue(),
    Components({
      // allow auto load markdown components under `./src/components/`
      extensions: ['vue', 'md'],
      // allow auto import and register components used in markdown
      include: [/\.vue$/, /\.vue\?vue/, /\.md$/],
      resolvers: [
        NaiveUiResolver(),
      ],
      dts: 'src/components.d.ts',
    }),
    legacy({
      targets: ['defaults', 'not IE 11']
    })
  ],
  server: {
    proxy: {

      '/api': {
          changeOrigin: true,
          target: 'https://worker.firehomework.top/dice/api',
          // target: 'http://8.130.140.128:8082',
          // target: 'http://localhost:8088',

          rewrite: (path) => path.replace(/^\/api/, ''),

      },
    }
  },
})
