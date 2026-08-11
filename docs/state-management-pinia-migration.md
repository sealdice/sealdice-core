# 状态管理规范与 Pinia 迁移方案

> 面向：所有参与 SealDice 新版前端（`ui/`）开发的同学，尤其是后端背景、前端经验较少的同学。
>
> 状态：**规范已定，阶段 0 待执行**。本文是后续状态管理相关工作（含胖页面重构）的地基。

## TL;DR

- 项目当前**不用 Pinia**，全局状态靠 feature 文件里 `export const x = ref()` 的模块级单例实现。
- 这个模式在当前规模能 work，但对**前端人力不充裕、以后端为主的团队**有三个会持续恶化的风险：可发现性差、调试盲区、约束太弱。
- 决定：**引入 Pinia 作为全局状态的唯一合法容器**，但**不一刀切全量重构**，而是「立规范 + 核心先行 + 渐进迁移」。
- 范式已确定：**setup 语法 + store 放 feature 内（`features/<domain>/store.ts`）**。
- 第一步是**阶段 0（奠基）**：装 Pinia、写规范、把 `theme` 迁成第一个范式 store。

---

## 一、问题：当前模式为什么会给团队带来维护困难

### 1.1 现状

项目（Vue 3 + TS + Naive UI + Vue Query）**没有引入 Pinia**。全局共享状态通过「模块级单例 ref」实现——在 feature 文件顶层写 `export const x = ref(...)`，任何 import 它的组件拿到的是同一个响应式实例。典型的全局状态点：

| 位置 | 管理的状态 |
|---|---|
| `features/auth/state.ts` | access token |
| `features/realtime/client.ts` | 连接状态（connected/connecting/lastError/transport） |
| `features/connect/realtime.ts` | IM 连接列表、工作流、二维码 |
| `features/base/logStream.ts` | 日志流缓冲 |
| `features/unsavedChanges/state.ts` | 未保存变更注册表 |
| `features/theme/useAppTheme.ts` | 主题模式、调色板 |
| `features/upload/resumableUpload.ts` | 上传任务 |

服务端状态统一走 Vue Query，这块没问题。问题集中在**客户端全局状态**。

### 1.2 三个会恶化的痛点（对后端为主的团队尤其致命）

**痛点 1：可发现性差——新人不知道「全局状态在哪」**

Pinia 在 Vue Devtools 里有一棵状态树，所有 store 一览无余。模块级 ref 散落在 30 个 feature 文件里，新人接手时只能靠 `grep` + 读 `ARCHITECTURE.md` 去拼凑「原来 `connect/realtime.ts` 第 30 行有一个全局 `connections`」。后端团队前端人少、新人多、缺少传带，这个认知成本会被**反复支付**。

**痛点 2：调试盲区——Devtools 看不到游离 ref（已确认）**

项目装了 `vite-plugin-vue-devtools`，但它和 Pinia Devtools 不是一回事。它能看组件树、路由、性能，**却不会把游离的模块级 ref 聚合成可浏览的状态树**，更没有时间旅行/快照。状态出 bug 时，只能靠 `console.log` 人肉追。后端同学前端调试能力本来就弱，这个损失被放大。

**痛点 3：约束太弱，会无序膨胀（已经在发生）**

没有任何机制阻止任何人在任意 feature 顶层随手加一个 `export const x = ref()` 当全局状态用。代码审查报告已经发现这个 drift 趋势。Pinia 至少让「这是全局状态」是一个**显式的、仪式性的动作**（`defineStore`）。后端团队 review 力量弱，弱约束 = 必然熵增。

### 1.3 为什么说「现在还不算困难，但不能不管」

当前全局状态总量不大、`ARCHITECTURE.md` 写得完整，所以**当下还能维护**。但上面三个痛点会随「时间 + 人员流动 + 状态数量增长」非线性放大。对前端不充裕的团队，越晚治理成本越高——这就是现在动手的理由。

---

## 二、原因分析：为什么不是「直接全上 Pinia」或「继续不上」

### 2.1 为什么不维持现状（模块级 ref）

见上文三个痛点。对后端团队，「靠自觉的规范」不可持续。

### 2.2 为什么不「一刀切全量迁移到 Pinia」

- 工作量大：30 个 feature 的状态要逐个迁移。
- 和更紧迫的维护债抢工期：代码审查显示，**胖页面下沉（`connect.vue` 791 行、`story.vue` 886 行）、query 样板重复（35 处手写 key）、胖 composable（`useCustomReplyEditor` 639 行）才是当前最大的维护风险**，优先级高于状态管理。
- 风险集中：一次性大改容易引入隐蔽的响应性 / 时序 bug。

### 2.3 为什么最终选 Pinia（而非别的）

- 后端团队需要**显式、强约束、可遵循**的模式，Pinia 的 `defineStore` 正好满足。
- Pinia 是 Vue 官方推荐，Devtools 一等公民，文档/社区充足，学习曲线低（setup store 对后端同学就是「一个返回状态对象的函数」）。
- 与现有 Vue Query / composable 架构无冲突。

### 2.4 采用「渐进式 Pinia」的核心思路

**规范要强**（满足后端团队的确定性诉求），**节奏要缓**（适合人力不充裕）：先立强规范 + 引入 Pinia + 做一个范式 store，让团队看清样板；然后核心基础设施先行；其余状态在后续重构里顺手迁移。

---

## 三、规范（强约束，所有人必须遵循）

| 规则 | 内容 |
|---|---|
| **全局状态** | 跨页面/跨组件共享的可变状态，**必须**用 Pinia store 定义。 |
| **局部状态** | feature 内部、单个页面内使用的状态，用 `ref` / `reactive`，**不**进 store。 |
| **服务端状态** | 一律 Vue Query（不变）。 |
| **store 语法** | **setup 语法**（`defineStore('name', () => {...})`）。 |
| **store 位置** | **feature 内**：`src/features/<domain>/store.ts`。保持 feature 自治。 |
| **禁止** | ① 模块级 `export const x = ref()` 作为全局状态；② 在页面组件里写跨页面共享的 ref；③ 直接解构 store 实例（会丢响应性，必须用 `storeToRefs`）。 |

**范式参考**：`src/features/theme/store.ts`（阶段 0 产出，见下文）。

**判断「这个状态该不该进 store」的简易标准**：
- 它是否被 **2 个及以上**页面/组件共享？→ 进 store。
- 它是否需要 **Devtools 可见、可追踪**？→ 进 store。
- 仅一个页面内用、随组件销毁即丢弃？→ 用局部 `ref`，不进 store。

---

## 四、迁移路线（分阶段）

### 阶段 0：奠基（最小可验证，立即执行）

目标：引入 Pinia、落地规范文档、交付一个团队可照抄的范式 store。

**步骤 1：装依赖**
```sh
pnpm add pinia
```

**步骤 2：`src/main.ts` 注册 Pinia**
在 `createApp(App)` 之后、`app.use(router)` 之前：
```ts
import { createPinia } from 'pinia';
// ...
app.use(createPinia());
```

**步骤 3：新建 `src/features/theme/store.ts`（范式样板）**

把现有 `useAppTheme.ts` 里的模块级单例**原样收进** `defineStore` setup 函数，业务逻辑几乎零改动，只是换了容器：

```ts
import { defineStore } from 'pinia';
import { computed, ref, watch } from 'vue';
import { usePreferredDark } from '@vueuse/core';
import {
  type ResolvedTheme, type ThemeMode,
  readStoredThemeMode, resolveThemeMode, syncDocumentTheme, writeStoredThemeMode,
} from './themeState';
import {
  DEFAULT_THEME_PALETTE, createThemeOverrides, normalizeThemePalette,
  readStoredThemePalette, syncDocumentThemePalette, writeStoredThemePalette,
  type ThemeColorKey, type ThemePalette,
} from './themePalette';

const storage = typeof window === 'undefined' ? undefined : window.localStorage;

export const useThemeStore = defineStore('theme', () => {
  const themeMode = ref<ThemeMode>(readStoredThemeMode(storage));
  const themePalette = ref<ThemePalette>(readStoredThemePalette(storage));
  const preferredDark = usePreferredDark();

  const resolvedTheme = computed<ResolvedTheme>(() =>
    resolveThemeMode(themeMode.value, preferredDark.value));
  const isDark = computed(() => resolvedTheme.value === 'dark');
  const themeOverrides = computed(() =>
    createThemeOverrides(themePalette.value, resolvedTheme.value));

  watch(resolvedTheme, theme => {
    if (typeof document === 'undefined') return;
    syncDocumentTheme(document.documentElement, theme);
    syncDocumentThemePalette(document.documentElement, themePalette.value);
  }, { immediate: true });
  watch(themeMode, mode => writeStoredThemeMode(storage, mode));
  watch(themePalette, palette => {
    writeStoredThemePalette(storage, palette);
    if (typeof document !== 'undefined') {
      syncDocumentThemePalette(document.documentElement, palette);
    }
  }, { deep: true, immediate: true });

  function setThemeMode(mode: ThemeMode) { themeMode.value = mode; }
  function toggleTheme() { themeMode.value = isDark.value ? 'light' : 'dark'; }
  function setThemePalette(palette: ThemePalette) {
    themePalette.value = normalizeThemePalette(palette);
  }
  function setThemeColor(key: ThemeColorKey, color: string) {
    themePalette.value = normalizeThemePalette({ ...themePalette.value, [key]: color });
  }
  function resetThemePalette() { themePalette.value = { ...DEFAULT_THEME_PALETTE }; }

  return {
    isDark, resolvedTheme, themeMode, themeOverrides, themePalette,
    setThemeMode, toggleTheme, setThemePalette, setThemeColor, resetThemePalette,
  };
});
```

**步骤 4：`src/features/theme/useAppTheme.ts` 改为兼容包装器（调用方零改动）**

> ⚠️ **Pinia 经典坑**：setup store 返回的 state/getter，直接解构会丢响应性，**必须用 `storeToRefs`**。

```ts
import { storeToRefs } from 'pinia';
import { useThemeStore } from './store';

export function useAppTheme() {
  const store = useThemeStore();
  const { isDark, resolvedTheme, themeMode, themeOverrides, themePalette } = storeToRefs(store);
  return {
    isDark, resolvedTheme, themeMode, themeOverrides, themePalette,
    setThemeMode: store.setThemeMode,
    toggleTheme: store.toggleTheme,
    setThemePalette: store.setThemePalette,
    setThemeColor: store.setThemeColor,
    resetThemePalette: store.resetThemePalette,
  };
}
```

现有调用方（如 `App.vue:59`、`AppThemeSwitch`）**一行不用改**。

> 新代码请直接用 `const themeStore = useThemeStore()`，`useAppTheme()` 仅为过渡兼容。
>
> `features/theme/index.ts` 无需改动——它的 re-export 路径（`useAppTheme` 从 `useAppTheme.ts` 导出）保持不变，store 文件只增不减。

**步骤 5：更新 `src/ARCHITECTURE.md` 状态管理章节**

把「状态管理原则」章节改写为本规范（第三节），并把 `features/theme/store.ts` 标注为范式参考，附迁移路线图。

**步骤 6：验证**
- `vue-tsc --build --force` 退出码 0
- 57 个测试全过（`themeState.test.ts` / `themePalette.test.ts` 测纯函数，不受影响，但仍需确认）
- **Devtools 检查**：打开 Vue Devtools → Pinia 面板 → 确认 `theme` store 可见，切换主题时 `isDark` / `resolvedTheme` 实时变化——这是迁移最主要的收益，必须亲眼确认
- **手动验证三件事**：暗/亮主题切换、调色板改色、刷新后主题持久化——这是 theme 的核心行为，迁移后必须照常
- **调用方验证**：`grep useAppTheme src/ -r` 确认所有调用方（`App.vue`、`AppThemeSwitch`、`AppThemeTransition` 等）type-check 无报错——`useAppTheme()` 的返回类型在迁移前后兼容，TS 类型检查足以冒烟

### 阶段 1：核心基础设施迁移

目标：把最常被引用的全局基础设施状态转成 store，让 Devtools 立即能看到核心状态、确立团队写法。

- `features/auth/state.ts` → `features/auth/store.ts`（token 唯一源）
- `features/realtime/` 的全局 ref（connected/connecting/lastError/activeTransport）→ realtime store；`client.ts` **保留 SSE 连接逻辑**，只把状态抽到 store
- `features/unsavedChanges/state.ts` → store

> ⚠️ auth 敏感：`currentAccessToken()` 被 `api/client.ts` 拦截器**同步**调用。
> 迁成 store 后，拦截器需要通过 `useAuthStore()` 读取 token，这要求 Pinia 已初始化。
>
> 这是安全的——推导如下：
> ```txt
> main.ts 执行顺序:
>   setupApiClient()          ← 只注册拦截器，不发起 HTTP 请求
>   app.use(createPinia())    ← Pinia 此时就绪
>   app.mount('#app')         ← 组件 setup、Vue Query 开始发请求
>                             ← 拦截器调用 useAuthStore()，Pinia 已可用
> ```
> `syncErudaFromStorage()` 在 mount 前是 fire-and-forget 的 localStorage 读，不发 HTTP，不触发拦截器。
>
> 迁移时需要验证：auth store 暴露的 ref/computed 返回值与原 `currentAccessToken()` / `hasAccessToken` 行为一致（同步可读、响应式）。
>
> ⚠️ realtime 复杂：generation 防护 + 重连逻辑与状态交织，迁移要干净地把「状态」与「连接机制」分离。
>
> 分离骨架（不需要写完整代码，先理解切割面）：
> ```txt
> realtime/client.ts（保留）
>   ├── 连接逻辑: connectWS / connectSSE / scheduleReconnect / generation 防护
>   ├── 事件分发: dispatch / parseEnvelope / subscribeRealtimeEvent
>   └── 注销后把 ref 赋值改为更新 store
>       e.g. connected.value = true   →   useRealtimeStore().connected = true
> 
> realtime/store.ts（新建）
>   └── 仅持有四个 ref: connected / connecting / lastError / activeTransport
>       （它们被 client.ts 写入、被组件读取）
> ```
> 组件层只需从 store 读状态（`useRealtimeStore().connected`），订阅仍走 client.ts。

### 阶段 2：业务状态迁移（与胖页面下沉合并）

`connect` / `base` / `upload` / `testMode` / `pwa` 等业务全局状态，**在做 H 类胖页面下沉重构时顺手迁移**，不单独排期，避免抢工期。

### 阶段 3：强约束收口

- 清理残留的模块级全局 ref。
- 引入强约束：一段 CI shell 脚本 `grep` `export const.*= ref(` 在 `src/features/` 下非 `store.ts` 文件里，发现即失败。ESLint 没有内置这条规则，自定义插件成本过高；shell 脚本对后端团队更顺手且同样有效。同时在 `AGENTS.md` review checklist 里加上这条，双保险。

---

## 五、风险与回退

| 风险 | 说明 | 应对 |
|---|---|---|
| 响应性丢失 | 漏用 `storeToRefs` 直接解构 store，调用方不响应 | 验证步骤「手动主题切换」会立即暴露；review 时把 `storeToRefs` 列为检查项 |
| store 惰性初始化时序 | store 在首次 `useXxxStore()` 才初始化 watch，与现状（模块加载即 watch）有微小差异 | theme 的首次 use 发生在 `App.vue` setup（mount 前），实际无影响；auth 的同步读取需在阶段 1 专门验证 |
| 迁移破坏现有行为 | 状态/动作搬动时漏掉或改错 | 每阶段保持 type-check + 57 测试全绿；核心行为手动验证 |
| store 自身无自动化测试 | 纯函数测试（themeState/themePalette）不受影响，但 store 层的集成行为（useThemeStore → useAppTheme → 组件）没有测试覆盖 | 依靠步骤 6 的手动验证清单兜底；后续阶段开始后考虑为 store 补关键路径测试 |
| `storeToRefs` 每次返回新包装 | 迁移后 `useAppTheme().isDark` 和 `useAppTheme().isDark` 是不同 ToRef 对象（底层 ref 相同，`watch` 不受影响，但 ref 身份比较或 Set/Map 用 ref 做 key 会出问题） | 现有代码未发现 ref 身份依赖，概率极低；迁移后 type-check 可验证类型兼容性 |
| 回退 | 阶段出问题 | `git revert` 即可，无破坏性数据/配置变更；各阶段相互独立，可停在任意阶段 |

---

## 六、决策记录

| 决策点 | 选择 | 理由 |
|---|---|---|
| 是否引入 Pinia | **是** | 后端团队需要显式、强约束、Devtools 可见的状态管理模式 |
| 迁移节奏 | **渐进式**（不一刀切） | 小团队人力有限，避免和胖页面重构抢工期 |
| store 语法 | **setup 语法** | 与项目现有 composable 风格一致，后端同学易理解 |
| store 位置 | **feature 内**（`features/<domain>/store.ts`） | 保持 feature 自治，状态就近 |
| 起步 | **阶段 0**（theme 范式先行） | 最小可验证，团队看清样板再推开 |
| 阶段 0 兼容策略 | **保留 `useAppTheme()` 包装器** | 阶段 0 零调用方改动，可独立验证 |

---

## 七、和其它重构的关系

本规范是后续所有前端重构的**地基**。特别是：

- **胖页面下沉（H 类）**：`connect.vue`/`story.vue`/`deck.vue` 等页面内的全局状态，下沉时自然落进对应 feature 的 store。
- **query/confirm/pagination 样板抽象**：与状态管理正交，可并行推进。

建议执行顺序：**阶段 0（本文档）→ 阶段 1（核心迁移）→ 胖页面下沉 + 阶段 2（业务迁移合并）→ 阶段 3（收口）**。

---

## 八、测试体系升级

### 8.1 现状

项目**没有引入任何测试框架**（无 vitest / jest / mocha）。测试靠自研 runner 驱动：

```
scripts/run-tests.mjs（36 行）
  └── jiti 递归 import 全部 src/**/*.test.ts
       └── throw 即失败；无 describe/it/expect/watch/coverage/mock
```

当前有 **57 个测试文件**，全部是**纯函数测试**，贴源存放（`feature/domain/xxx.test.ts`），覆盖：

| 类别 | 文件数 | 代表 |
|---|---|---|
| 构建策略 | 5 | chunkPolicy / bundlePolicy / embedConfig / warningPolicy |
| 路由派生 | 4 | routeMeta / navigationModel / routeRecords / historyBase |
| viewModel 转换 | 8 | about / backup / ban / resource / publicDice / customText / baseSetting / helpdoc |
| storyPainter | 11 | renderers / state / textExport / parquetDataset / compat…… |
| 工具/状态 | 29 | crypto / fileHash / eruda / clipboard / logDisplay / connect / pwa…… |

**完全空缺的测试类型**：
- 零组件测试（17+ 页面和数十个组件一个都没测）
- 零 Vue Query hook 测试
- 零 HTTP mock / 异步流程测试
- 零交互 / 表单 / 导航测试

每个测试文件都在**手写 `assertEqual` / `assertDeepEqual`**——从抽样看，这两个 helper 在 4 个文件里定义的一字不差。这说明团队**想要共享断言能力**，只是没搭起框架。

### 8.2 判断：对后端团队，这个现状比表面看上去更合理

两个反直觉的事实：

**"自研 runner + 手写 assert"不是业余，是精确命中 ROI 最高区。**
后端同学最擅长写纯函数测试——给输入、验输出、不看 UI。57 个测试全部打在这个区域，没有一份精力浪费在"学 jsdom + 写组件渲染"上。如果当初上了 vitest 全家桶但没写够纯函数测试，反而更糟。现在的**测试文化和测试目标（测什么）是对的**，缺的是**工具链（怎么测）**。

**复制粘贴 assertEqual 暴露的不是纪律问题，是"工具链摩擦"。**
团队想要共享断言，但没建公共 `test-utils.ts`，也没引入 `expect`。这恰好说明**需要一个轻量框架来消除这些摩擦**，而不是"团队不写测试"或"团队不会写测试"。

### 8.3 缺什么

| 能力 | 现状 | 对维护的影响 |
|---|---|---|
| 断言库 | 每个文件手写 `assertEqual` | 复制粘贴、报错信息不一致、无 diff 输出 |
| 测试组织（`describe` / `it`） | 模块顶层裸断言 | 失败时只能看行号，不知道"测的哪个场景" |
| watch 模式 | 无 | 改代码后手动 `node scripts/run-tests.mjs`，开发体验为零 |
| mock（`vi.fn()` / `vi.mock()`） | 无 | 所有依赖 HTTP / 浏览器 API / browser-only 的逻辑**不可测**，导致团队回避写这类测试 |
| 覆盖率 | 无 | 不知道测了什么、漏了什么 |
| 组件测试 | 零 | UI 变更全凭人工回归 |

### 8.4 方案：引入 vitest（三段式，零迁移成本）

**为什么是 vitest？** 它和 Vite 同源配置（`vite.config.ts` 里加 `test` 字段即可），且最关键——它**兼容现有 throw 模式**。你把 57 个 `throw new Error(...)` 的文件扔进去，能直接跑（vitest 文件里未捕获的 throw 等于测试失败）。现有测试一行不改。

```
阶段 A：零迁移成本（今天就能做）
  ├── pnpm add -D vitest
  ├── vite.config.ts 加 test: { include: ['src/**/*.test.ts'] }
  ├── package.json: "test": "vitest run"
  ├── 跑 pnpm test 确认 57 个全绿
  └── 收益：
        • pnpm vitest（watch 模式，改代码自动重跑相关测试）
        • pnpm vitest run --coverage（覆盖率报告，首次知道哪些代码有测试）
        • 失败时 vitest 自动 diff 期望值 vs 实际值（告别手写 Error 拼字符串）

阶段 B：渐进统一（不强推，自然过渡）
  ├── 建 src/test-utils/assert.ts，导出 assertEqual / assertDeepEqual
  │   （消除复制粘贴——现有测试不改，新测试 import 公用版本）
  ├── 团队写新测试时用 vitest 的 expect（后端熟悉的断言风格）
  └── 老测试不改，新代码自然过渡

阶段 C：挑高价值场景补测试（不动全局，按需追加）
  ├── auth 流程（token 读/写/清 + 拦截器时序）→ vi.mock() 模拟 localStorage + axios
  ├── realtime 连接状态机（generation 防护、退避）→ 纯逻辑部分可测
  ├── Pinia store（迁移完成后）→ vitest + createTestingPinia
  └── 关键组件（AppShell 布局切换）→ @vue/test-utils + jsdom
```

**vite.config.ts 改动（阶段 A 全部变更量）：**

```ts
// 在 export default defineConfig({...}) 的顶层加
test: {
  include: ['src/**/*.test.ts'],
},
```

### 8.5 不做的事（对你们团队是坑）

- ❌ **不设覆盖率目标**（会让测试变成凑数字，而非验证行为）
- ❌ **不强求组件测试**（`@vue/test-utils` + `jsdom` 学习曲线陡、维护成本高、ROI 低——后端团队优先把精力投在纯函数和状态机测试上）
- ❌ **不上 E2E**（Playwright / Cypress）——你们团队规模撑不起，且 E2E 的维护成本远高于收益
- ❌ **不要求"每个 PR 必须有测试"**——对纯函数高价值场景优先补；UI 交互测试等关键路径出了 bug 时反哺补

### 8.6 执行时机

阶段 A 可以和本方案的**阶段 0（Pinia 奠基）并行或紧随其后**——两者都是工程地基升级，不冲突、不抢工期。投入：装一个依赖 + 改 2 个文件，收益立即可见（watch 模式 + 覆盖率），且为后续 store 迁移提供测试兜底。

阶段 B/C 顺其自然，不强排期。后端团队会在写新代码时自然地用 `expect` 替代 `assertEqual`，不需要一次做完。
