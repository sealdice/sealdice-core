import './assets/main.css';

import { createApp } from 'vue';
import { VueQueryPlugin } from '@tanstack/vue-query';

import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import relativeTime from 'dayjs/plugin/relativeTime';

import App from './App.vue';
import router from './router';

import { setupApiClient } from './api';
import { queryClient } from './queryClient';

// 引入字体: 通用字体 / 等宽字体
import 'vfonts/Lato.css';
import 'vfonts/FiraCode.css';

// 配置 dayjs
dayjs.locale('zh-cn');
dayjs.extend(relativeTime);

// naive-ui 样式冲突处理
const meta = document.createElement('meta');
meta.name = 'naive-ui-style';
document.head.appendChild(meta);

setupApiClient();

const app = createApp(App);

app.use(router);
app.use(VueQueryPlugin, {
  queryClient,
});

app.mount('#app');
