import { createApp } from "vue";
import App from "./App.vue";

// import "~/styles/element/index.scss";

// import ElementPlus from "element-plus";
// import all element css, uncommented next line
// import "element-plus/dist/index.css";

// or use cdn, uncomment cdn link in `index.html`

import "~/styles/index.css";

// If you want to use ElMessage, import it.
// import "element-plus/theme-chalk/src/message.scss"
import "element-plus/es/components/message-box/style/css"
import "element-plus/es/components/dialog/style/css"
import 'element-plus/theme-chalk/display.css'
import './str.polyfill'

import { useStore } from "~/store";
const loading = useStorage('router-view-loading', true)
import router from "~/router";

router.beforeEach((to, from, next) => {
  loading.value = true
  next()
})
router.afterEach((to, from) => {
  setTimeout(() => {
    loading.value = false
  }, 300)
})

import VueDiff from 'vue-diff'
import 'vue-diff/dist/index.css'
import yaml from 'highlight.js/lib/languages/yaml'
import toml from 'highlight.js/lib/languages/ini'

const app = createApp(App);
app.use(createPinia())

VueDiff.hljs.registerLanguage('yaml', yaml)
VueDiff.hljs.registerLanguage('toml', toml)
app.use(VueDiff, {
  componentName: 'VueDiff'
})
  .use(router)

const store = useStore();

store.trySignIn().then(() => {
  app.mount("#app")
})

try {
  (window as any).store = store;
  (globalThis as any).store = store;
} catch (e) {
}

// app.use(ElementPlus);
