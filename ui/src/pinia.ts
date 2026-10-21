import { createPinia } from 'pinia';

// 统一导出单例 Pinia，保证在 app.use() 之前也能安全创建全局 store。
export const appPinia = createPinia();
