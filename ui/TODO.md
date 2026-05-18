# SealDice UI 迁移进度

## 已完成

### 基础设施

- [x] OpenAPI 客户端生成链路（@hey-api/openapi-ts + Vue Query）
- [x] 文件路由（vue-router/auto-routes + routeMeta）
- [x] 布局体系（default/plain/wide）
- [x] 侧边栏菜单（router/navigation.ts）
- [x] 认证状态管理（features/auth）
- [x] 路由进度条（nprogress）
- [x] 主题切换（features/theme）
- [x] 实时日志流（features/base/logStream）

### 共享组件

- [x] FoldableCard — 可折叠卡片
- [x] DiffViewer — 代码 diff 查看器
- [x] ResourceRenderer — 图片/音频/视频渲染
- [x] ConditionBuilder — 条件构建器
- [x] NestedRuleEditor — 拖拽嵌套规则编辑器
- [x] CustomTextBox — 自定义文案分组
- [x] DynamicForm — 动态表单渲染
- [x] PlaceholderPage — 占位页

### 页面迁移

| 页面 | V2 API | 前端 |
|------|--------|------|
| 主页 | ✅ | ✅ |
| 连接管理 | ✅ | ✅ |
| 牌堆管理 | ✅ | ✅ |
| 跑团日志 | ✅ | ✅ |
| JS 扩展 | ✅ | ✅ |
| 自定义回复 | ✅ | ✅ |
| 自定义文案 | ✅ | ✅ |
| 帮助文档 | ✅ | ✅ |
| 拦截管理 | ✅ | ✅ |
| 群组管理 | ✅ | ✅ |
| 黑白名单 | ✅ | ✅ |
| 基本设置 | ✅ | ✅ |
| 高级设置 | ✅ | ✅ |
| 备份 | ✅ | ✅ |
| 公骰设置 | ✅ | ✅ |
| 指令测试 | ✅ | ✅ |
| 资源管理 | ✅ | ✅ |
| 关于 | ✅ | ✅ |

### 上传体系

- [x] 公共分片上传控制器（features/upload/resumableUpload.ts）
- [x] 牌堆上传接入（分片 + 断点续传 + 进度）
- [x] JS 扩展上传（简单 multipart）

## 待办

### 增强需求

- [ ] 跑团日志：日志链接展示与强制上传（已实现，待 UI 验证）
- [ ] 跑团日志：按月数清理 + VACUUM
- [ ] 跑团日志：ID 作为一等公民
- [ ] 帮助文档：接入分片上传
- [ ] 升级包：接入分片上传
- [ ] JS 扩展：接入分片上传（uploadcore）

### 代码质量

- [ ] 全面检查各页面移动端适配
- [ ] 移除旧 sealdice-naiveui 引用（如已完成迁移）
- [ ] 补充 API 错误处理边界情况
