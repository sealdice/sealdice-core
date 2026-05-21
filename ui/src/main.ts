import 'fake-qq-ui/styles/fake-qq-ui.css';
import 'fake-qq-ui/styles/light.css';
import 'fake-qq-ui/styles/dark.css';
import './assets/main.css';
import './polyfills/structuredClone';

import { createApp } from 'vue';
import { VueQueryPlugin } from '@tanstack/vue-query';

import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import relativeTime from 'dayjs/plugin/relativeTime';
import safeHtmlDirective from './directives/safeHtml';
import {
  create,
  ProInput ,
  ProSelect ,
  ProDigit ,
  ProSwitch
} from 'pro-naive-ui'
import App from './App.vue';
import router from './router';

import { setupApiClient } from './api';
import { queryClient } from './queryClient';

// 应用入口只负责装配全局基础设施：
// 1. 全局样式、字体、polyfill；
// 2. Naive UI 所需的运行时补丁；
// 3. OpenAPI fetch client、router、Vue Query。
// 业务初始化不要放在这里，优先放进布局或 feature composable，便于测试和按页面加载。

// 引入字体: 通用字体 / 等宽字体
import 'vfonts/Lato.css';
import 'vfonts/FiraCode.css';

// 配置 dayjs
dayjs.locale('zh-cn');
dayjs.extend(relativeTime);

// Naive UI 会按运行时顺序插入 style 标签。显式插入标记节点可以稳定样式优先级，
// 避免局部 scoped CSS 与组件库 CSS 在热更新/构建后出现顺序漂移。
const meta = document.createElement('meta');
meta.name = 'naive-ui-style';
document.head.appendChild(meta);

// 生成客户端本身不带业务态。这里集中注入 baseUrl、token、401 清理和错误反馈。
setupApiClient();

const app = createApp(App);

// 未来考虑换掉这个玩意，还得手动引入，真麻烦啊。
const proNaive = create({
  components: [ProInput, ProSelect, ProDigit, ProSwitch]
})

app.use(proNaive)

app.directive('safe-html', safeHtmlDirective);
app.use(router);
app.use(VueQueryPlugin, {
  queryClient,
});

app.mount('#app');
