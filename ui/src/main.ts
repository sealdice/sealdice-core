import './assets/main.css';

import { createApp } from 'vue';
import { createPinia } from 'pinia';

import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import relativeTime from 'dayjs/plugin/relativeTime';

import App from './App.vue';
import router from './router';

// 引入字体: 通用字体 / 等宽字体
import 'vfonts/Lato.css';
import 'vfonts/FiraCode.css';

// 配置dayjs
dayjs.locale('zh-cn');
dayjs.extend(relativeTime);

// naive-ui 样式冲突处理
const meta = document.createElement('meta');
meta.name = 'naive-ui-style';
document.head.appendChild(meta);

const app = createApp(App);

app.use(createPinia());
app.use(router);

app.mount('#app');
