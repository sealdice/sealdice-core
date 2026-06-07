# 账号管理 UI Step 向导设计

## 日期

2026-05-19

## 背景

当前账号添加流程为：Modal 弹窗 → `<n-select>` 下拉选协议 → `<DynamicForm>` 填表单。用户需要在十几项平面协议中快速找到目标协议，对新用户不友好。

## 目标

- 把协议选择拆分为四层可导航的 Step 向导：平台 → 方式 → 协议 → 填信息
- 每层都有介绍说明，帮助用户理解区别
- 流程保持一致（即使只有单一方式的平台也显示完整步骤）
- 保留废弃协议但标记为 `[已废弃]`
- 使用 Naive UI `<n-steps>` 组件
- 用户手动点击"下一步"推进（以便阅读介绍）
- 旧前端（sealdice-ui）不做兼容

## 非目标

- 不改造表单 schema 体系（`forms.json` 和 `<DynamicForm>` 保持不变）
- 不改连接列表页主表格（`connect.vue` 的主表格部分不变）
- 不改编辑已有账号的交互（编辑仍用原来的简单表单弹窗）

## 数据模型改动

### 后端 `GET /sd-api/v2/imconnection/protocols`

当前返回平面列表 `ProtocolDefinition[]`，改造为树形结构 `PlatformTreeNode[]`：

```go
type PlatformTreeNode struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Methods     []MethodTreeNode       `json:"methods"`
}

type MethodTreeNode struct {
    ID          string                   `json:"id"`
    Name        string                   `json:"name"`
    Description string                   `json:"description"`
    Protocols   []ProtocolDefinition      `json:"protocols"`
}

// ProtocolDefinition 保持不变（已有 deprecated、schemaKey、capabilities 等字段）
// 新增 description 字段
```

### 层级映射

```
Platform: qq
  Method: builtin (内置客户端)
    Protocol: lagrange
    Protocol: milky-internal
    Protocol: gocq (deprecated)
  Method: separate (分离客户端)
    Protocol: milky
    Protocol: gocq-separate
    Protocol: onebot-reverse
    Protocol: officialqq
    Protocol: red (deprecated)

Platform: dingtalk
  Method: default
    Protocol: dingtalk

Platform: discord
  Method: default
    Protocol: discord

Platform: kook
  Method: default
    Protocol: kook

Platform: telegram
  Method: default
    Protocol: telegram

Platform: minecraft
  Method: default
    Protocol: minecraft

Platform: dodo
  Method: default
    Protocol: dodo

Platform: slack
  Method: default
    Protocol: slack

Platform: satori
  Method: default
    Protocol: satori

Platform: sealchat
  Method: default
    Protocol: sealchat
```

> 说明：
> - 所有平台都有 `methods` 数组（保证流程一致）
> - 非 QQ 平台的 method 只有一个 `default`
> - `description` 字段为简要介绍，1-2 句话
> - `deprecated: true` 的协议在 UI 上显示 `[已废弃]` 标签

## 前端交互设计

### 弹窗结构

```
┌─────────────────────────────────────────────────────────────┐
│  添加账号                                        [×]         │
│                                                              │
│  ○ 选择平台  →  ○ 选择方式  →  ○ 选择协议  →  ○ 填写信息     │
│  ━━━━━━━━━━       ────────        ──────        ──────      │
│                                                              │
│  ┌─────────────────────────────────────┐                     │
│  │                                     │                     │
│  │  [ 平台/方式/协议卡片列表 ]          │                     │
│  │                                     │                     │
│  └─────────────────────────────────────┘                     │
│                                                              │
│              [上一步]          [下一步 →]                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 四步详情

#### Step 1 - 选择平台

- 渲染每个平台的卡片（`n-card` 或自定义卡片组件）
- 每张卡片显示：平台图标、平台名称、简介
- 点击卡片选中（高亮）
- 点击"下一步"进入 Step 2
- "上一步" 按钮禁用

#### Step 2 - 选择方式

- 根据 Step 1 所选 `platform.id`，渲染该平台下的 `methods`
- 每张卡片显示：方式名称、简介（区别说明）
- 点击卡片选中
- 点击"上一步"可回退到 Step 1（已选平台保留）
- 点击"下一步"进入 Step 3

#### Step 3 - 选择协议

- 根据前序选择，渲染 `methods.protocols`
- 每张卡片显示：协议名称、简介（适用场景）
- 废弃协议卡片上显示 `n-tag` 标签 `[已废弃]`
- 点击卡片选中
- 点击"上一步"可回退到 Step 2
- 点击"下一步"进入 Step 4

#### Step 4 - 填写信息

- 复用现有 `<DynamicForm>` 组件
- 根据所选协议的 `schemaKey` 加载表单字段
- "上一步"可回退到 Step 3
- "添加"按钮提交 → `POST /sd-api/v2/imconnection/`
- 提交成功后 Modal 关闭，显示 "账号已添加" toast

### 状态管理

前端用一个 reactive 对象维护状态：

```ts
const wizardState = reactive({
  currentStep: 1, // 1-4
  platform: null as PlatformTreeNode | null,
  method: null as MethodTreeNode | null,
  protocol: null as ProtocolDefinition | null,
});
```

## API 影响

| API | 变化 |
|-----|------|
| `GET /sd-api/v2/imconnection/protocols` | 响应体从 `ProtocolDefinition[]` 变为 `PlatformTreeNode[]` |
| `POST /sd-api/v2/imconnection/` | 不变 |
| `GET /sd-api/v2/imconnection/schemas` | 不变（Step 4 仍用 `schemaKey` 加载） |
| 其余 API | 不变 |

## 涉及文件

### 后端

- `api/v2/imconnection/service.go` — `buildProtocolDefinitions()` 改为生成树形结构
- `api/v2/imconnection/handler.go` — 调整响应类型
- `api/v2/model/imconnection/resp.go` — 新增 `PlatformTreeNode`、`MethodTreeNode` 类型
- `api/v2/imconnection/imconnection.go` — OpenAPI/Huma 接口定义更新

### 前端

- `ui/src/pages/connect.vue` — 重写添加账号 Modal 为 Steps 向导
- `ui/src/api/generated/` — 重新生成 OpenAPI 类型

## 废弃协议处理

- `deprecated: true` 的协议在 Step 3 卡片上显示 `[已废弃]` 标签
- 仍可被选中，但视觉上有明确标识
- 与当前下拉选择框中已有的废弃标记一致

## 编辑流程

- 编辑已有账号：保持现有行为，直接弹出表单 Modal
- 不经过 Steps 向导

## 风险和限制

1. **API 破坏性变更**：`/protocols` 返回体结构改变，若前端有其他地方消费该接口需一并更新。
2. **旧前端不兼容**：旧前端不再维护此接口即可。
3. **描述文案**：文案为占位式简要说明，后续需持续优化。
