import { resolve } from 'path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import IconsResolver from 'unplugin-icons/resolver'
import Icons from 'unplugin-icons/vite'
import legacy from '@vitejs/plugin-legacy'

function pathResolve(dir: string) {
  return resolve(process.cwd(), '.', dir)
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueJsx(),
    AutoImport({
      dirs: ['src/**'],
      include: [
        /\.[tj]sx?$/, // .ts, .tsx, .js, .jsx
        /\.vue$/,
        /\.vue\?vue/ // .vue
      ],
      imports: ['vue', '@vueuse/core'],
      dts: true,
      vueTemplate: true,
      resolvers: [NaiveUiResolver(), IconsResolver()]
    }),
    Components({
      dirs: ['src/components', 'src/pages'],
      extensions: ['vue'],
      include: [/\.vue$/, /\.vue\?vue/],
      resolvers: [NaiveUiResolver(), IconsResolver()],
      dts: true
    }),
    Icons({
      compiler: 'vue3',
      autoInstall: true
    }),
    legacy({
      targets: ['defaults']
    })
  ],
  resolve: {
    alias: [
      {
        find: '@',
        replacement: pathResolve('src') + '/'
      }
    ]
  },
  build: {
    sourcemap: false,
    chunkSizeWarningLimit: 1024
  },
  server: {
    fs: {
      strict: false
    }
  }
})
