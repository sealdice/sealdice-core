# SealDice UI 迁移 TODO

## Phase 0：共享组件迁移（零后端依赖）✅ 已完成

- [x] `shared/FoldableCard.vue` — 可折叠卡片，带错误状态 (← `utils/foldable-card.vue`)
- [x] `shared/DiffViewer.vue` — 代码 diff 查看器 (← `utils/diff-viewer.vue`)
- [x] `shared/ResourceRenderer.vue` — 图片/音频/视频渲染 (← `utils/resource-render.vue`)
- [x] `shared/ConditionBuilder.vue` — 条件构建器 (← `utils/custom-reply-conditions.vue`)
- [x] `shared/NestedRuleEditor.vue` — 拖拽嵌套回复规则 (← `utils/nested.vue`)
- [x] `shared/CustomTextBox.vue` — 可折叠文本组 (← `customText/CustomTextBox.vue`)

**新增依赖**: `vuedraggable@4.1.0`, `vue-diff@1.2.4`

## Phase 1：类型定义 + API 层脚手架

- [ ] `types/common.ts` — 通用类型
- [ ] `types/dice.ts` — DiceServer, DiceConfig
- [ ] `types/im.ts` — 连接类型定义
- [ ] `types/censor.ts` — 审查模块类型
- [ ] `types/deck.ts` — 牌堆类型
- [ ] `types/group.ts` — 群组类型
- [ ] `types/story.ts` — 故事日志类型
- [ ] `types/backup.ts` — 备份类型
- [ ] `types/ban.ts` — 黑名单类型
- [ ] `types/resource.ts` — 资源类型
- [ ] `types/js.ts` — JS 插件类型
- [ ] `types/helpdoc.ts` — 帮助文档类型
- [ ] `api/modules/auth.ts` — 认证 API
- [ ] `api/modules/dice.ts` — 骰子核心 API
- [ ] `api/modules/imConnections.ts` — IM 连接 API
- [ ] `api/modules/censor.ts` — 审查 API
- [ ] `api/modules/deck.ts` — 牌堆 API
- [ ] `api/modules/group.ts` — 群组 API
- [ ] `api/modules/story.ts` — 故事日志 API
- [ ] `api/modules/backup.ts` — 备份 API
- [ ] `api/modules/banconfig.ts` — 黑名单 API
- [ ] `api/modules/resource.ts` — 资源 API
- [ ] `api/modules/js.ts` — JS 插件 API
- [ ] `api/modules/helpdoc.ts` — 帮助文档 API
- [ ] `api/modules/others.ts` — 其他 API（health, baseInfo, log）
- [ ] `api/modules/utils.ts` — 工具 API（news, cron, network health）
- [ ] `api/modules/configs.ts` — 配置 API（customText, customReply）

## Phase 2：认证 + 心跳

- [ ] 迁移 `utils.ts` 中的 `passwordHash`, `sleep` 到 `composables/`
- [ ] 心跳检测 logic (`useHeartbeat`)
- [ ] Token 存储与自动登录
- [ ] 密码框与登录弹窗组件

## Phase 3：首页 + 关于页

- [ ] `views/HomeView.vue` — 仪表盘：内存、网络健康、系统日志
- [ ] `views/AboutView.vue` — 版本信息 + 贡献者

## Phase 4：IM 连接管理

- [ ] `views/ConnectView.vue` — 主控制器
- [ ] 子组件：平台卡片、登录表单（QR/SMS/验证码）

## Phase 5：系统设置

- [ ] `views/SettingsView.vue` — 大表单 split 为子组件

## Phase 6：牌堆 + 帮助文档

- [ ] `views/DeckView.vue` + 子组件
- [ ] `views/HelpDocView.vue` — 文件树 + 搜索

## Phase 7：自定义文本 + 自定义回复

- [ ] `views/CustomTextView.vue`
- [ ] `views/CustomReplyView.vue` — 拖拽规则编辑器

## Phase 8：内容审查

- [ ] `views/CensorView.vue` + 子组件

## Phase 9：故事日志 + 群组 + 黑名单

- [ ] `views/StoryView.vue`
- [ ] `views/GroupView.vue`
- [ ] `views/BanView.vue`

## Phase 10：收尾

- [ ] `views/ResourceView.vue`
- [ ] `views/TestView.vue`
- [ ] `views/PublicDiceView.vue`
- [ ] `views/AdvancedSettingsView.vue`
