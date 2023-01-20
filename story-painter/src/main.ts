import { createApp } from "vue";
import App from "./App.vue";

// import "~/styles/element/index.scss";

// import ElementPlus from "element-plus";
// import all element css, uncommented next line
import "element-plus/dist/index.css";

// or use cdn, uncomment cdn link in `index.html`

import "~/styles/index.scss";

// If you want to use ElMessage, import it.
import "element-plus/theme-chalk/src/message.scss"
import "element-plus/theme-chalk/el-loading.css"
import 'element-plus/es/components/message-box/style/css'
import './str.polyfill.ts'
// import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'

import { createPinia } from 'pinia'
// import { RecycleScroller, DynamicScroller, DynamicScrollerItem } from 'vue-virtual-scroller'

const app = createApp(App);
app.use(createPinia())
// app.component('DynamicScroller', DynamicScroller);
// app.component('DynamicScrollerItem', DynamicScrollerItem);
// app.component('RecycleScroller', RecycleScroller);
// app.use(ElementPlus);
app.mount("#app");
