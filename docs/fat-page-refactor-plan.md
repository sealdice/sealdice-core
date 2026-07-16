# 胖页面下沉重构计划

> 面向：所有参与 `ui/` 开发的同学。
>
> 状态：**待执行**。依赖：公共 helper 抽象（§三）必须先于页面重构完成。
>
> 关联文档：[状态管理规范与 Pinia 迁移方案](./state-management-pinia-migration.md)——本文的 feature 内状态迁移与之合并推进。

## TL;DR

- 项目有 4 个严重超标的"胖页面"（`story.vue` 886 行、`connect.vue` 791 行、`deck.vue` 791 行、`group.vue` 554 行），把本应下沉到 feature 的业务逻辑全堆在页面里。
- 全仓存在 **~45 处 `enabled: hasAccessToken` 样板、~69 处手写 queryKey、~27 处 queryFn 三件套、~10+ 处 testMode 错误模板、25 处确认弹窗样板**——全部可以靠公共 helper 消灭。
- 已有 3 个模范 feature（`censor` / `baseSetting` / `helpdoc`）证明"页面 < 260 行 + queries.ts + mutations.ts"是可达成的。
- 计划分两步：**先抽公共 helper（§三），再逐页下沉（§四）**。每页独立验证、可停在任意节点。

---

## 一、问题现状

### 1.1 四个胖页面

| 页面 | 行数 | 页面内 query | 页面内 mutation | 命令式 fetch | 本地 ref | 函数 | 已下沉的 feature |
|---|---|---|---|---|---|---|---|
| `pages/mod/story.vue` | 886 | 1 | 3 | 4 处 | 10 | 19 | 仅 backup（`useStoryBackup`） |
| `pages/connect.vue` | 791 | 4 | 4 | 2 处 | 13 | 17 | realtime + endpointDisplay |
| `pages/mod/deck.vue` | 791 | 1 | 3 | 1 处 | 5 | 7 | **无（`features/deck/` 不存在）** |
| `pages/misc/group.vue` | 554 | 2 | 0 | 5 处裸调 | — | — | 无 |

对照模范薄页：

| 页面 | 行数 | 说明 |
|---|---|---|
| `pages/mod/censor.vue` | 252 | 全委托 `useCensorConfigDraft` / `useCensorMutations` / `useCensor*Query` |
| `pages/misc/base-setting.vue` | 305 | 全委托 `useBaseSettingDraft` / `useBaseSettingMutations` |
| `pages/tool/test.vue` | 180 | 全委托 `useToolTest()` |
| `pages/mod/reply.vue` | 17 | 纯组装 |

### 1.2 重复样板统计

| 样板 | 出现次数 | 涉及文件 | 典型形态 |
|---|---|---|---|
| `enabled: hasAccessToken`（含 computed 变体） | ~45 | 22 | 每个 query 手写 token gate |
| 手写 `queryKey:` 字面量 | ~69 | 18 | 定义处和 invalidate 处各写一遍 |
| queryFn 三件套 `const { data } = await getX({ throwOnError: true }); return data.item;` | ~27 | 13 | 逐字相同的解构 + 取 .item |
| `throwOnError: true` | 121 | — | 每个 API 调用手动带 |
| mutation onSuccess invalidate | 42 | 16 | 手写一串 invalidateQueries |
| `message.success('xxx成功')` | 73 | 25 | 成功提示 + 状态清理 + 失效缓存三连 |
| onError 的 testMode + getErrorMessage 模板 | ~10+ | 4 | `isTestModeApiError → warning，否则 → error` |
| `dialog.warning(` 确认弹窗 | 25 | 16 | title + content + onPositiveClick |

### 1.3 queryKey 的四种混乱状态

目前 key 管理**同时存在四种写法**，且互相不对齐：

1. 手写字面量数组（`['deck-list', params]`）——最多
2. `queryKeys.ts` 工厂函数——仅 `js/` 和 `story/`，且 `js/` 内部仍未贯彻（`useJsData.ts` 退回手写）
3. OpenAPI 生成的 `getXxxQueryKey()`——约 12 处
4. spread `getXxxOptions()` 连 key + queryFn 一起拿——connect/backup/advanced-setting

**隐患**：同一个 key 字符串（如 `['custom-reply-files']`、`['helpdoc-tree']`）在定义处和 invalidate 处各写一遍，靠人工保持一致。`helpdoc.vue:163` 已出现页面手写 key 与 feature helper 内 key 对不上的实例。

---

## 二、目标架构

### 2.1 页面层的契约（薄页面标准）

页面**只做三件事**：

1. 组合 feature 暴露的 query / mutation / composable
2. 管理纯 UI 局部状态（弹窗开关、当前 tab 等）
3. 渲染组件，通过 props / emits 传递数据

**禁止**：定义 useQuery / useMutation、手写 queryKey、命令式 fetch、复杂业务状态机。

目标行数：**≤ 300 行**（ censor 252 行、base-setting 305 行是基准）。

### 2.2 feature 层的标准结构

每个"有服务端数据"的 feature 目录应包含：

```
features/<domain>/
├── queryKeys.ts          ← 所有 queryKey 的唯一来源（工厂函数）
├── queries.ts            ← useXxxQuery()，封装 enabled + queryFn
├── mutations.ts          ← useXxxMutations(options)，封装 mutationFn + onSuccess + onError
├── viewModel.ts          ← 纯数据转换函数（normalize / build / format）
├── useXxxDraft.ts        ← 草稿状态 + 脏检测（有编辑场景才需要）
└── *.test.ts             ← viewModel / queryKeys 的纯函数测试
```

**范式参考**：`features/censor/` 是最完整的实现，新 feature 照抄即可。

### 2.3 公共 helper 层

在 `features/` 之上新建一层薄 helper（放 `src/lib/query/` 或 `src/features/queryHelpers/`），消灭跨 feature 的重复样板：

| helper | 消灭的样板 | 设计 |
|---|---|---|
| `useAuthedQuery(options)` | 45 处 `enabled: hasAccessToken` + 三件套 queryFn | 自动注入 `enabled: hasAccessToken`（合并额外条件），queryFn 强制 `throwOnError: true` |
| `useAuthedMutation(options)` | 10+ 处 testMode onError 模板 | 自动注入 testMode-aware onError |
| `makeInvalidateKeys(keys)` | 42 处手写 invalidateQueries | 工厂返回 `() => Promise.all([...invalidate])` |
| `useConfirmAction()` | 25 处 `dialog.warning` | 统一 title/content/onPositiveClick 样板 |

---

## 三、公共 helper 抽象（前置依赖，必须先做）

> 这一阶段是所有页面重构的前置条件——先有标准 helper，才能把页面逻辑下沉成标准形态。否则只是把样板从一个地方搬到另一个地方。

### 3.1 `useAuthedQuery`

```ts
// src/lib/query/useAuthedQuery.ts
import { computed, toValue, type MaybeRefOrGetter } from 'vue';
import { useQuery, type QueryOptions } from '@tanstack/vue-query';
import { hasAccessToken } from '@/features/auth/state';

export function useAuthedQuery<TData>(
  options: QueryOptions<TData> & {
    enabled?: MaybeRefOrGetter<boolean>;
  },
) {
  return useQuery({
    ...options,
    enabled: computed(() => {
      if (!hasAccessToken.value) return false;
      return options.enabled ? toValue(options.enabled) : true;
    }),
    // throwOnError: true 是 OpenAPI client 的默认值，这里不重复设
  });
}
```

**效果**：45 处 `enabled: hasAccessToken` 全部消失。需要额外条件的（如 `selectedProtocolKey === 'lagrange'`）传 `enabled: () => condition`。

### 3.2 `useAuthedMutation`（testMode-aware onError）

```ts
// src/lib/query/useAuthedMutation.ts
import type { MutationOptions } from '@tanstack/vue-query';
import { isTestModeApiError, getTestModeBlockMessage } from '@/features/testMode/state';
import { getErrorMessage } from '@/features/auth/error';
import type { MessageApi } from 'naive-ui';

export function makeMutationErrorHandler(
  message: MessageApi,
  defaultErrorText: string,
) {
  return (error: unknown) => {
    if (isTestModeApiError(error)) {
      message.warning(getTestModeBlockMessage(error));
      return;
    }
    message.error(getErrorMessage(error, defaultErrorText));
  };
}
```

**效果**：connect/backup/resource/ban 里重复 10+ 次的 onError 模板压缩为一行调用。

### 3.3 queryKeys 规范

**规则**：每个 feature 必须有 `queryKeys.ts`，所有 key 通过工厂函数定义，**禁止手写字面量数组**。

```ts
// 范式：features/censor/queryKeys.ts（新建）
export const censorKeys = {
  all: ['censor'] as const,
  status: () => [...censorKeys.all, 'status'] as const,
  config: () => [...censorKeys.all, 'config'] as const,
  files: () => [...censorKeys.all, 'files'] as const,
  // ...
};
```

invalidate 只引用工厂函数：
```ts
await queryClient.invalidateQueries({ queryKey: censorKeys.all });
```

**效果**：消灭 69 处手写字面量 + 定义/invalidate 两处对不上的风险。

### 3.4 `useConfirmAction`

```ts
// src/lib/dialog/useConfirmAction.ts
import type { DialogApi } from 'naive-ui';

export function useConfirmAction(dialog: DialogApi) {
  return (options: {
    title: string;
    content: string;
    positiveText?: string;
    onConfirm: () => Promise<void> | void;
  }) => {
    dialog.warning({
      title: options.title,
      content: options.content,
      positiveText: options.positiveText ?? '确定',
      negativeText: '取消',
      onPositiveClick: async () => { await options.onConfirm(); },
    });
  };
}
```

**效果**：25 处 `dialog.warning` 样板压缩为单行调用。

### 3.5 执行步骤与验证

```
1. 建 src/lib/query/ 目录，实现 useAuthedQuery + makeMutationErrorHandler
2. 建 src/lib/dialog/ 目录，实现 useConfirmAction
3. 选一个已有 feature（如 censor）试点切换到新 helper，跑 type-check + test
4. 跑 lint 确认无新错误
```

---

## 四、分页面重构计划

### 执行顺序

按"风险/收益比"排序：

```
deck（零 feature，纯增量，风险最低）
  → group（裸调多，下沉后收益明显）
  → story（最大，但有 useStoryBackup 范式可参照）
  → connect（最复杂，有 wizard 状态机 + TSX 列，放最后）
```

每页独立验证、可停在任意节点。

---

### 4.1 `pages/mod/deck.vue`（791 行 → 目标 ≤ 200 行）

**现状**：`features/deck/` 完全不存在，全部逻辑堆在页面。

**新增 feature 结构**：

```
features/deck/
├── queryKeys.ts           ← ['deck-list'] 等 key 工厂
├── useDeckList.ts         ← 列表 query + 分页 + 搜索 + 排序
├── useDeckMutations.ts    ← reload / delete mutation（用 makeMutationErrorHandler）
├── useDeckUpdate.ts       ← 检查更新 → diff 预览 → 确认更新（完整状态机）
├── useDeckUpload.ts       ← 包住 useResumableUpload + deck adapter + restore
└── deckUploadAdapter.ts   ← init/complete/buildChunkUrl 配置
```

**新增组件**：

```
components/deck/
├── DeckUploadPanel.vue    ← 上传队列 UI（进度/状态/重试，行 57–99）
├── DeckItemCard.vue       ← 单个牌堆 FoldableCard（行 121–238）
└── DeckUpdateDiffModal.vue ← diff 模态 + DiffViewer（行 252–270）
```

**页面重构后只做**：组合 4 个 composable + 渲染 3 个组件 + 弹窗开关。

**关键注意**：
- `doCheckUpdate`（行 604–624）当前是命令式 fetch + 手写 loading，下沉时统一成 useMutation。
- 上传 adapter 的 4MB 常量和 4 个后端 hook 是纯业务逻辑，必须从页面搬出。

---

### 4.2 `pages/misc/group.vue`（554 行 → 目标 ≤ 200 行）

**现状**：5 处裸调 `postSdApiV2Group*`（行 327–473），手写 loading + 手写 `data.message !== 'ok'` 成功判定（行 380/425/439），完全没用 Vue Query。

**新增 feature**：

```
features/group/
├── queryKeys.ts
├── useGroupList.ts        ← 列表 query + 分页 + 搜索
└── useGroupMutations.ts   ← 批量退群 / 批量通知 / 修改 / 退群
```

**关键改动**：
- 5 处裸调全部改用 useMutation。
- `data.message !== 'ok'` 改为 `throwOnError: true`（统一错误处理，不再手动判 message 字段）。
- 退群偏好状态已在 `features/group/quitPreference.ts`（纯函数），保持不动。

---

### 4.3 `pages/mod/story.vue`（886 行 → 目标 ≤ 250 行）

**现状**：仅 backup 子流程已下沉。日志列表/条目查看/清理/上传四个子流程全在页面。还有 4 处命令式 fetch（绕过 Vue Query）。

**新增 feature**：

```
features/story/
├── queryKeys.ts           ← 已有，扩展
├── useStoryLogs.ts        ← 列表 query + 分页 + 搜索 + 删除（含 delLog/delLogs）
├── useStoryLogItems.ts    ← 条目分页 + 模式切换 + 用户颜色映射
├── useStoryCleanup.ts     ← 清理表单 + 预览 + 执行
├── useStoryUpload.ts      ← 上传 mutation
├── model.ts               ← linkStateText / randomColorWithIndex 纯函数
└── useStoryBackup.ts      ← 已有，不动
```

**新增组件**：

```
components/story/
├── StoryLogCard.vue       ← 日志列表卡片（行 48–185 内联模板）
├── StoryLogItemView.vue   ← 条目分页文本视图（行 188–269 内联模板）
└── StoryCleanupPanel.vue  ← 清理面板
```

**关键改动**：
- 4 处命令式 fetch（`searchLogs` / `openRawItem` / `handleItemPageChange` / `refreshCleanupPreview`）改为 useQuery，统一数据获取范式。
- `itemsView` computed 有副作用（行 721–730 写 `users.value`），下沉时必须改为显式 watchEffect 或 fetch 后构建。
- `dialog.warning` 4 处用 `useConfirmAction` 替代。

---

### 4.4 `pages/connect.vue`（791 行 → 目标 ≤ 250 行）

**现状**：最复杂——4 query + 4 mutation + 创建向导状态机 + 签名服务联动 + TSX 表格列 + realtime 桥接。

**新增 feature**：

```
features/connect/
├── endpointDisplay.ts     ← 已有，扩展 workflowTag/workflowText/detailRows
├── realtime.ts            ← 已有，不动（分层最干净）
├── queryKeys.ts           ← 新建
├── queries.ts             ← 4 个 useAuthedQuery + REST/realtime 合并 watch
├── mutations.ts           ← 4 个 mutation（用 makeMutationErrorHandler）
├── createWizard.ts        ← 向导状态机（step/platform/method/protocol + goNext/goPrev）
├── signInfo.ts            ← lagrange 签名服务联动（query + watch + 派生）
└── endpointRow.ts         ← adapterOf/workflowOf/detailRows（并入 endpointDisplay 或独立）
```

**新增组件**：

```
components/connect/
├── ConnectCreateWizard.vue ← 已有，但 12 props 瘦身为直接消费 composable
├── ConnectTableColumns.tsx ← TSX 列定义（行 599–667）外移
└── ConnectEditDialog.vue   ← 编辑弹窗
```

**关键注意**：
- realtime 这条线（`useRealtimeConnections`）**保持不动**——它已经是这个页面的最佳实践样板。
- 向导状态机下沉后，`ConnectCreateWizard` 可以直接消费 composable，12 个 props 大幅瘦身。
- `getSdApiV2ImconnectionByIdConfig`（编辑配置读取，行 569）当前是裸 await，下沉时改为 query。

---

## 五、验证策略

每个页面重构后必须通过：

1. **`vue-tsc --build --force`**：0 错误（类型安全）
2. **`vitest run`**：57+ 测试全过（现有测试不被破坏；新 feature 的 viewModel/queryKeys 补测试）
3. **手动验证**：该页面的核心业务流程（列表加载 → 搜索 → 增删改 → 弹窗交互）照常工作
4. **行数检查**：页面 ≤ 300 行

---

## 六、风险与注意事项

| 风险 | 说明 | 应对 |
|---|---|---|
| 命令式 fetch → Vue Query 是行为变更 | story 的 `searchLogs` 失败时手写 `message.error`；改 useQuery 后错误走 `api/client.ts` 全局分类策略 | ARCHITECTURE 已记录错误策略（dialog/business/message 三类），迁移后 toast 行为由全局统一，可接受 |
| `itemsView` computed 副作用 | story.vue 行 721–730 在 computed 里写 `users.value`，是反模式 | 下沉时改为显式 watchEffect 或 fetch 后构建，不让反模式扩散到 feature |
| ConnectCreateWizard props 大改 | 向导状态机下沉后 wizard 从 12 props 变为直接消费 composable | 这是有意改善（减少 props drilling），但要同步改 wizard 组件 |
| queryKey 迁移遗漏 | 4 种 key 写法并存，迁移时如果漏改某个 invalidate，缓存不失效 | queryKeys.ts 是唯一来源 + grep 确认无残留字面量 |

---

## 七、不做的事

- ❌ **不拆 `components/shared/`**：shared 组件已经是纯展示，不在本次范围。
- ❌ **不动 realtime**：`useRealtimeConnections` 分层干净，别碰。
- ❌ **不补组件测试**：人力不充裕，纯函数测试优先（vitest 阶段 B/C 按需追加）。
- ❌ **不追求"完美架构"**：目标是"页面 ≤ 300 行 + 无样板重复"，不是理论上的最优分层。

---

## 八、执行顺序总览

```
阶段 1：公共 helper（§三）
  ├── src/lib/query/useAuthedQuery.ts
  ├── src/lib/query/makeMutationErrorHandler.ts
  ├── src/lib/dialog/useConfirmAction.ts
  └── 试点 censor 切换验证

阶段 2：deck 下沉（§4.1，风险最低）
  ├── 建 features/deck/ 全套
  ├── 建 components/deck/ 3 个组件
  └── 页面 ≤ 200 行

阶段 3：group 下沉（§4.2）
  ├── 建 features/group/
  └── 5 处裸调 → mutation

阶段 4：story 下沉（§4.3，工作量最大）
  ├── 建 features/story/ 4 个 composable
  ├── 建 components/story/ 3 个组件
  └── 4 处命令式 fetch → useQuery

阶段 5：connect 下沉（§4.4，最复杂）
  ├── 建 features/connect/ queries/mutations/wizard/signInfo
  ├── ConnectCreateWizard props 瘦身
  └── TSX 列外移

每个阶段独立验证（type-check + vitest + 手动），可停在任意节点。
```
