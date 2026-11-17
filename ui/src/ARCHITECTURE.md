# UI 架构说明

这份说明面向第一次接手 `ui` 的开发者。项目是 SealDice 新版管理后台，目标是用一个可复现、可生成、以 V2 OpenAPI 为数据源的 Vue 3 前端替代旧管理界面。

## 总体架构

```txt
后端 Huma V2 OpenAPI
  -> pnpm run generate-api
  -> openapi.json
  -> src/api/generated/*
  -> src/api/client.ts 注入 baseUrl / token / 错误处理 / 会话清理
  -> page / 独立 composable 持有页面草稿与操作
  -> page 组装页面，共享关键状态由 Pinia 持有
  -> 组件负责展示
```

`openapi.json` 和 `src/api/generated` 是构建期产物，不提交到仓库。正式构建会先生成它们；本地开发如果缺失或 API 变化，先运行 `pnpm run generate-api`。

这个项目按实际职责组织代码，不要求每个功能都经过固定的封装层：

- 公共能力放在 `api/`、`features/`、`router/`、`components/app-shell/` 和少量全局 `layouts/` 中。
- 页面可以直接使用生成的 API、query 和 store；只有可复用的业务规则或需要独立管理的生命周期才抽成 composable。
- 不为单一调用方增加逐项转发 props、refs 或 actions 的兼容层，也不复制查询缓存到另一份全局状态。
- 真正跨页面复用的 UI 才进入 `components/shared/`。
- 路由页面直接承载整页界面，不再套一个只被它引用的“整页组件”。页内工具栏和静态说明不必单独建文件；较大的独立交互区可以保留组件。
- 单页查询、保存和草稿状态由页面闭环管理，不拆成互相传递 `getConfigPayload` / `onConfigSaved` 的固定层次。拦截和帮助文档页面是这类流程的示例。

## 目录职责

- `src/api/`：API 边界。
  - `generated/` 由 OpenAPI 生成，不手改、不提交。
  - `client.ts` 负责全局 axios client 配置、token 注入、错误反馈、401 会话清理。
  - `config.ts` 负责 API baseUrl。
  - `download.ts` 放文件下载这类生成器不方便表达的薄封装。
- `src/features/`：业务域逻辑。
  - 页面共享的状态、query、mutation、实时事件适配、上传控制器、认证状态都放这里。
  - 这一层负责“把后端接口和页面状态翻译成可用的业务模型”。
- `src/components/app-shell/`：后台应用壳层组件。
  - 包括侧边栏、面包屑、搜索、主题切换、解锁弹窗、未保存提示。
  - 这些组件可以依赖 router、auth、theme、unsaved changes 等应用级 feature。
- `src/components/shared/`：跨页面复用组件。
  - 只通过 props / emits / `defineModel` 接收数据。
  - 不直接发请求，不保存业务域状态。
- `src/components/<domain>/`：某个业务页的拆分组件。
  - 例如 JS 扩展页的 list/config/data 视图、指令测试页的聊天窗口。
- `src/layouts/`：页面壳。
  - `default` 与 `wide` 都保留后台导航，只调整内容区宽度。
  - `plain` 用于轻量流程页。
- `src/pages/`：路由入口。
  - 页面负责组合 query、mutation、feature composable 和展示布局。
  - 页面不承载全局基础设施。
- `src/router/`：路由补充配置。
  - 路由由 `vue-router/auto-routes` 生成。
  - 标题、布局、菜单、动态导航语义集中维护。

## 公共能力的位置

这部分是本项目最重要的“不要放错地方”的约定。

- HTTP 边界在 `src/api/client.ts`：只负责认证头、令牌续期、错误归一化和会话清理，不导入组件库、主题或页面模块。
  - `src/api/errors.ts` 定义纯错误类型和展示归属；迟到的旧请求不能清理或覆盖新会话。
  - `main.ts` 将 `features/feedback/apiFeedback.ts` 的全局展示回调注入客户端，应用卸载时释放两者。
  - 网络故障、404 和 500 交给全局展示；非登录 401 清理会话后由 `AppUnlockDialog` 统一引导重新登录，不再额外弹出全局错误框。参数错误、展示模式限制和其他业务错误由来源页面处理。
  - 页面捕获接口错误后使用 `notifyError(error, message, fallback)`，已全局反馈或被取消的请求不再弹第二次 toast；本地校验可以直接提示。
- 登录态直接通过 `features/auth/store.ts` 的 `useAuthStore()` 读取，响应式引用用 `storeToRefs()`。不新增转发型 `state.ts`，不在页面读写 localStorage。
- Vue Query 全局缓存和重试策略在 `src/queryClient.ts`；查询继续使用生成客户端，禁止自行维护第二套服务端缓存。
- 基础设置的固定分组、字段布局和搜索文案在 `features/baseSetting/schema.ts`；后端仅提供当前值与类型化补丁，完整校验后才应用配置与副作用。
- JS 配置由 `JsConfigView` 持有唯一编辑补丁并直接注册全局未保存提示；页面不复制脏标志或通过组件引用调用保存，字段编辑器不另存本地值。
  - Vue Query 保存服务端基线，展示时叠加编辑补丁。按插件提交固定快照，成功后只清除与快照相同的编辑；部分失败或保存期间的新输入继续保留。
  - Cron 校验按字段取消旧请求，等待校验期间禁止保存。重置和删除与保存互斥，配置标签切换保留草稿。
  - V2 后端通过 `ConfigManager.SetConfigs` / `ResetConfigs` 在锁内预检整批配置、一次原子写盘；写盘失败恢复内存值，成功后更新已校验的定时任务。
- 实时连接由 `features/realtime/store.ts` 持有，调用方直接使用 store；订阅自动初始化连接，不在模块导入时创建 store。
  - 连接列表、HTTP 首屏快照和 SSE 更新共用完整连接 DTO（含 workflow/qrcode）；业务 store 只保存连接列表，其余展示数据从列表派生。
  - 日志、连接等业务 store 只订阅所需事件，释放时注销自己的监听器；全局传输 store 释放时停止 SSE、重连定时器和认证监听。
- 账号创建、编辑各自由对应向导/弹窗持有查询、草稿和提交。页面只管理弹窗入口、账号列表操作与二维码展示；卡片直接读取完整连接 DTO，不再拆出工作流和二维码映射逐层传递。
  - 创建接口已经负责官方 QQ 凭据探测和重复校验，前端不再先调用测试接口重复探测。
  - 写操作后刷新账号列表查询，SSE 断开时仍通过 HTTP 更新。编辑弹窗只在草稿未修改时接受刷新；签名服务只补缺失默认值，不随刷新覆盖手动选择。
- 主题和 PWA 安装直接使用各自的 store；PWA 安装事件在应用启动时监听，卸载时释放。
- 路由语义统一在 `src/router/navigation.ts`、`src/router/navigationModel.ts`。
  - 新页面一般只需要先补导航模型，标题、布局、面包屑等会一起派生。
- 未保存提示统一在 `src/features/unsavedChanges/`。
  - 页面只注册 source，不自己实现全局拦截。
  - 状态通过 `useUnsavedChangesStore()` 消费；注册 hook 负责生命周期，路由守卫每次导航独立确认，不保留跨导航跳过标志。
  - `canSave=false` 是明确禁止操作，不能作为“未提供”回退；执行入口同时检查是否可保存与是否正在保存。
  - 保存、放弃和重载操作由 `AppUnsavedChangesPanel` 展示；`PageHeader` 只负责标题和页面自己的操作插槽，不再复制全局待处理状态。
- 上传统一通过 `useUploadStore()` 消费；响应式状态用 `storeToRefs()`，不再增加逐个转发的 `state.ts` 门面。
  - `enqueue()` 只等待文件完成入队准备，`upload()` 等待单文件安装完成；失败与取消都会结束后者的 Promise。
  - store 持有并释放 Uppy；queue 管理业务任务，engine 负责第三方身份和事件映射。
- 扩展包安装用例在 `features/package/operations.ts`，统一安装、启用、重载和刷新。展示组件选择上传／URL／商店来源并展示结果，不另写三套后置状态机。
- 扩展包详情由 `features/package/usePackageDetail.ts` 管理，查询与保存回调都按包 ID 隔离；Vue Query 负责取消旧查询，页面不再维护另一份配置请求状态。
  - 详情抽屉保存期间阻止继续编辑；安装后的列表刷新失败单独反馈，不把已经完成的安装报成失败。
- 扩展包目录查询由 `features/package/usePackageCatalog.ts` 持有：安装列表、仓库后端、商店页共享认证条件、取消信号和刷新逻辑。页面保留筛选、展示与用户确认。
- App 壳布局宽度逻辑统一在 `src/components/app-shell/appShellLayout.ts`。
  - `default`/`wide` 的差异是内容区宽度，不是另一套页面体系。

## 状态管理原则

服务端状态优先使用 Vue Query，稳定数据通过 OpenAPI generated client 获取。

客户端状态分三类管理：

| 类型                              | 容器                                                        | 说明                                                                                 |
| --------------------------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| **全局状态**（跨页面/跨组件共享） | **Pinia store**（`features/<domain>/store.ts`，setup 语法） | 如 token、实时连接状态、主题等。禁止用模块级 `export const x = ref()` 充当全局状态。 |
| **局部状态**（单页面/组件内）     | `ref` / `reactive`                                          | 随组件生命周期销毁。                                                                 |
| **持久化偏好**                    | Pinia store + localStorage                                  | 只有主题、token 等需要跨刷新保留的才持久化。                                         |

**范式参考**：`src/features/theme/store.ts`——setup 语法、feature 内、消费者使用 `storeToRefs()`。

每份状态都应有明确的写入方。子组件接收展示数据、发出操作意图，避免原地修改父级数组或传入可变对象后让多处共同写入；自定义文案条目通过索引和新文本通知编辑器，由编辑器更新草稿。预览和帮助信息来自 Vue Query，“是否自定义”等本地标记从草稿派生，不能回写查询缓存。认证、实时连接、上传队列、全局未保存提示继续由 Pinia 持有。

自定义回复只持有一份完整文件草稿，分页列表与修改状态直接派生。条件、结果和排序通过事件交给编辑器写入，不复制分页草稿再用深度监听同步。拦截设置及查询表单通过模型更新事件提交新值，不原地修改传入的嵌套对象。

页面层应该尽量只做三件事：

1. 读取 feature/store/composable 暴露出来的状态。
2. 触发 mutation 或本地交互。
3. 渲染组件。

不要把以下内容塞进页面：

- 全局错误处理。
- token 注入和清理。
- 实时连接管理。
- 大量跨页面共享状态（应下沉到 store）。
- 复杂的数据转换规则。

## 一个简单功能的完整流程

下面以 `/tool/test` 的“指令测试”页面为例。

### 1. 路由进入

- 用户访问 `/tool/test`。
- 页面文件是 `src/pages/tool/test.vue`。
- 页面标题、布局等由 `src/router/navigation.ts` 和 `src/router/navigationModel.ts` 派生。

### 2. 页面初始化

- 页面里通过 `useToolTest()` 获取业务状态。
- 视图只负责渲染：
  - 模式切换
  - 消息窗口
  - 输入框
  - 快捷操作
  - 错误提示

### 3. feature 组织业务状态

- `src/features/toolTest/useToolTest.ts` 是核心。
- 它负责：
  - 管理当前模式 `private/group`
  - 管理输入框内容
  - 发送测试消息
  - 轮询 pending 消息
  - 触发牌堆 / JS / 帮助文档重载
- `src/features/toolTest/model.ts` 负责纯数据模型和纯函数：
  - 初始消息
  - 追加自发消息
  - 追加 pending 消息

### 4. 数据来源

- 待处理消息来自 `getSdApiV2ToolTestMessagesPending()`。
- 发送消息来自 `postSdApiV2ToolTestMessages()`。
- 快捷操作来自：
  - `postSdApiV2DeckReload()`
  - `postSdApiV2JsReload()`
  - `postSdApiV2HelpdocReload()`

### 5. 状态流转

- 有 token 时，`watch(hasAccessToken)` 会加载上下文并启动轮询。
- 用户发送消息时：
  - 先把自己的消息追加到本地会话里。
  - 再发请求给后端。
  - 成功后继续拉取 pending 消息。
  - 失败时追加 tip 提示。
- 轮询失败时会停止轮询，并把错误显示到页面。

### 6. 视图层展示

- `src/components/tool-test/ToolTestChatWindow.vue` 负责聊天窗口展示。
- 页面只传 `title` 和 `messages`。
- 这样以后如果聊天窗口样式变化，业务逻辑不用跟着改。

### 7. 这个流程体现的分层

- `pages/` 负责组合。
- `features/` 负责业务状态。
- `components/` 负责展示。
- `api/` 负责请求边界。
- `queryClient.ts` 负责缓存策略。

## 新页面开发顺序

1. 确认后端 V2 OpenAPI 已覆盖接口，运行 `pnpm run generate-api` 更新本地 `openapi.json` 和 `src/api/generated`。
2. 在 `pages/` 新建路由入口。
3. 在 `router/navigation.ts` 增加导航语义。
4. 由 `buildRouteMeta` 从导航模型派生页面标题和 layout。
5. 单页流程先在页面内保持完整；有复用规则或独立生命周期时再提取到 `features/<domain>/`，共享关键状态进入 Pinia。
6. 跨页面复用组件才放 `components/shared/`。
7. 运行 `pnpm run type-check` 和 `pnpm run build-only`。

## 维护约定

- 不手改或提交 `src/api/generated/`。需要变更 API 类型时改后端 OpenAPI，再重新生成本地产物。
- 不在页面里实现全局错误处理、token 注入、401 清理，这些属于 `src/api/client.ts`。
- 不在页面里直接创建 SSE，这些属于 `src/features/realtime/store.ts`。
- 不把老前端的大 store 模式搬进来。每个业务域维护自己的最小状态。
- 公共能力放在对应 feature / api / router / app-shell；只服务一个页面的简单派生状态和布局保留在使用处。

## 构建与验证边界

只使用 `package.json` 锁定的 pnpm。全新检出先 `pnpm install --frozen-lockfile`、`pnpm run generate-api`，再运行 `pnpm run check:prepared`；后者依次验证嵌入式构建、自动声明后的类型检查、真实 Vite 配置、实际产物引用及测试。

`src/build/resourceBudget.ts` 对单个 JS 块、入口 gzip、首批静态依赖图 gzip 和 PWA 预缓存分别设置预算。预算依据构建产物，不代表浏览器首屏时间。可选在线调试面板不参与预缓存；业务路由和离线资源保留。超限应检查依赖及加载边界，而不是只提高警告阈值。

后端的 Go 编译依赖嵌入式资源，不能与清空该输出目录的前端构建并行。前端构建完成后再执行 Go 测试、race 和静态检查。

测试优先采用标准 `expect`、独立行为名称、真实 HTTP / 第三方适配器 / 挂载事件；不复制框架断言，也不通过源码字符串锁定 CSS 或内部变量。确有格式契约的 PNG 等资源直接校验文件内容，构建接线校验实际产物。

## 常见落点速查

- API 客户端配置：`src/api/client.ts`
- 认证 token：`src/features/auth/store.ts`
- Vue Query 配置：`src/queryClient.ts`
- 实时连接：`src/features/realtime/store.ts`
- 路由菜单语义：`src/router/navigation.ts`
- 路由派生元信息：`src/router/navigationModel.ts`
- App 壳宽度逻辑：`src/components/app-shell/appShellLayout.ts`
- 未保存拦截：`src/features/unsavedChanges/`
- PWA 升级与受控刷新：`src/features/pwa/swUpdate.ts`（SW 换版/异步 chunk 404 时统一走升级提示，见 `AppUpdatePrompt.vue`）
