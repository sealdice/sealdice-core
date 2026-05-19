# 账号管理 UI Step 向导实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把后端 `/protocols` API 改为返回平台→方式→协议 的树形结构，并把前端添加账号弹窗重构为 Naive UI `<n-steps>` 四步向导。

**Architecture:** 后端保留 `protocolBy` 平面映射用于创建/编辑时按 key 查找协议，新增 `protocolTree` 字段存放层级数据供 `GetProtocols` 返回。前端复用现有 `<DynamicForm>`，只改造"添加账号"弹窗的协议选择流程为步骤卡片。

**Tech Stack:** Go 1.25 + Huma v2 / Vue 3.5 + Naive UI 2.43 + TypeScript + `<script setup lang="tsx">`

---

### 文件结构映射

| 文件 | 职责 |
|------|------|
| `api/v2/model/imconnection/resp.go` | 新增 `PlatformTreeNode`、`MethodTreeNode`；给 `ProtocolDefinition` 加 `Description` |
| `api/v2/imconnection/service.go` | `buildProtocolDefinitions()` 重构为树形构建；`GetProtocols` 改返回树；Service 字段调整 |
| `api/v2/imconnection/service_test.go` | 更新 `protocolByKey` helper 与容器模式测试，适配树形响应 |
| `ui/src/pages/connect.vue` | 重写"添加账号" Modal 为 `<n-steps>` 向导（4 步），编辑/列表/二维码保持原样 |
| `ui/src/api/generated/` | 重新生成 OpenAPI TypeScript 类型（`pnpm run generate-api`） |

---

### Task 1: 后端 — 更新响应模型

**Files:**
- Modify: `api/v2/model/imconnection/resp.go`

- [ ] **Step 1: 给 `ProtocolDefinition` 增加 `Description` 字段**

```go
type ProtocolDefinition struct {
    Key            string             `json:"key"`
    Name           string             `json:"name"`
    Platform       string             `json:"platform"`
    SchemaKey      string             `json:"schemaKey"`
    Deprecated     bool               `json:"deprecated"`
    Available      bool               `json:"available"`
    DisabledReason string             `json:"disabledReason,omitempty"`
    Capabilities   ProtocolCapability `json:"capabilities"`
    Description    string             `json:"description,omitempty"` // 新增
}
```

- [ ] **Step 2: 新增 `MethodTreeNode` 与 `PlatformTreeNode`**

在 `resp.go` 末尾追加：

```go
type MethodTreeNode struct {
    ID          string                  `json:"id"`
    Name        string                  `json:"name"`
    Description string                  `json:"description"`
    Protocols   []*ProtocolDefinition   `json:"protocols"`
}

type PlatformTreeNode struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Methods     []*MethodTreeNode `json:"methods"`
}
```

- [ ] **Step 3: 更新 `ProtocolListResp`**

```go
type ProtocolListResp struct {
    Items []*PlatformTreeNode `json:"items"` // 原来是 []*ProtocolDefinition
}
```

- [ ] **Step 4: 编译检查**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui && go build ./api/v2/model/imconnection/...`
Expected: 编译通过（此时 service 层还未改，会报错，没关系）

---

### Task 2: 后端 — 重构 Service 层协议树构建

**Files:**
- Modify: `api/v2/imconnection/service.go`

- [ ] **Step 1: 重命名 `buildProtocolDefinitions` → `buildProtocolTree` 并返回树形结构**

替换 `service.go` 中 `buildProtocolDefinitions` 方法（第 105-140 行）为以下内容（保留 `baseCapabilities` 与 `withWorkflow` 函数）：

```go
func (s *Service) buildProtocolTree() []*imconnm.PlatformTreeNode {
    baseCapabilities := imconnm.ProtocolCapability{
        Create: true, Update: true, Delete: true, Enable: true,
    }

    // QQ 内置
    qqBuiltin := &imconnm.MethodTreeNode{
        ID:          "builtin",
        Name:        "内置客户端",
        Description: "协议端直接运行在海豹核心内部，无需额外部署。推荐大多数用户使用。",
        Protocols: []*imconnm.ProtocolDefinition{
            {Key: "lagrange", Name: "Lagrange", Platform: "QQ", SchemaKey: "lagrange", Available: true, Description: "新架构内置客户端，稳定性好，支持扫码登录。推荐作为 QQ 内置首选。", Capabilities: withWorkflow(baseCapabilities, true, true, true)},
            {Key: "milky-internal", Name: "内置 Milky", Platform: "QQ", SchemaKey: "milky-internal", Available: true, Description: "基于 Milky 的内置实现，支持扫码登录。", Capabilities: withWorkflow(baseCapabilities, true, true, false)},
            {Key: "gocq", Name: "内置 GoCQ", Platform: "QQ", SchemaKey: "gocq", Deprecated: true, Available: false, DisabledReason: "内置 gocq 已弃用，请使用内置客户端或分离部署", Description: "早期内置方案，已停止维护，不建议使用。", Capabilities: imconnm.ProtocolCapability{}},
        },
    }

    // QQ 分离
    qqSeparate := &imconnm.MethodTreeNode{
        ID:          "separate",
        Name:        "分离客户端",
        Description: "需要自行部署协议端服务，再通过 WebSocket 连接海豹核心。适合高级用户。",
        Protocols: []*imconnm.ProtocolDefinition{
            {Key: "milky", Name: "Milky (外部)", Platform: "QQ", SchemaKey: "milky", Available: true, Description: "外部 Milky 协议端，需自行部署后连接。", Capabilities: baseCapabilities},
            {Key: "gocq-separate", Name: "OneBot11 正向WS", Platform: "QQ", SchemaKey: "gocq-separate", Available: true, Description: "OneBot 11 正向 WebSocket 协议，需配合协议端使用。", Capabilities: baseCapabilities},
            {Key: "onebot-reverse", Name: "OneBot11 反向WS", Platform: "QQ", SchemaKey: "onebot-reverse", Available: true, Description: "OneBot 11 反向 WebSocket 协议，需配合协议端使用。", Capabilities: baseCapabilities},
            {Key: "officialqq", Name: "QQ 官方机器人", Platform: "QQ", SchemaKey: "officialqq", Available: true, Description: "QQ 官方机器人接口，仅支持频道消息。", Capabilities: baseCapabilities},
            {Key: "red", Name: "Red 协议", Platform: "QQ", SchemaKey: "red", Deprecated: true, Available: false, DisabledReason: "Red 协议已弃用", Description: "QQ Red 协议，已废弃。", Capabilities: imconnm.ProtocolCapability{}},
        },
    }

    platforms := []*imconnm.PlatformTreeNode{
        {
            ID: "qq", Name: "QQ",
            Description: "腾讯 QQ 即时通讯平台，支持群聊和私聊。",
            Methods: []*imconnm.MethodTreeNode{qqBuiltin, qqSeparate},
        },
        {
            ID: "dingtalk", Name: "钉钉",
            Description: "阿里巴巴旗下企业协作平台。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过钉钉开放平台接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "dingtalk", Name: "钉钉", Platform: "DingTalk", SchemaKey: "dingtalk", Available: true, Description: "钉钉机器人协议，支持企业群消息收发。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "discord", Name: "Discord",
            Description: "海外流行游戏社区平台。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 Discord Bot 接口接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "discord", Name: "Discord", Platform: "Discord", SchemaKey: "discord", Available: true, Description: "Discord 官方 Bot 接口。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "kook", Name: "KOOK",
            Description: "国内游戏语音与社区平台（开黑啦）。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 KOOK Bot 接口接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "kook", Name: "KOOK(开黑啦)", Platform: "KOOK", SchemaKey: "kook", Available: true, Description: "KOOK 官方 Bot 接口。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "telegram", Name: "Telegram",
            Description: "注重隐私的海外即时通讯平台。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 Telegram Bot 接口接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "telegram", Name: "Telegram", Platform: "Telegram", SchemaKey: "telegram", Available: true, Description: "Telegram Bot 接口。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "minecraft", Name: "Minecraft",
            Description: "Minecraft 游戏服务器接入。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 RCON 协议接入 Minecraft 服务器。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "minecraft", Name: "Minecraft服务器", Platform: "Minecraft", SchemaKey: "minecraft", Available: true, Description: "Minecraft 服务器 RCON 接入。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "dodo", Name: "Dodo",
            Description: "Dodo 语音社区平台。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 Dodo Bot 接口接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "dodo", Name: "Dodo语音", Platform: "Dodo", SchemaKey: "dodo", Available: true, Description: "Dodo 官方 Bot 接口。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "slack", Name: "Slack",
            Description: "企业团队协作平台。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 Slack Bot 接口接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "slack", Name: "Slack", Platform: "Slack", SchemaKey: "slack", Available: true, Description: "Slack Bot 接口。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "satori", Name: "Satori",
            Description: "通用聊天平台协议（开发中）。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 Satori 协议接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "satori", Name: "[WIP]Satori", Platform: "Satori", SchemaKey: "satori", Available: true, Description: "Satori 通用协议。", Capabilities: baseCapabilities},
                },
            }},
        },
        {
            ID: "sealchat", Name: "SealChat",
            Description: "SealChat 协议（开发中）。",
            Methods: []*imconnm.MethodTreeNode{{
                ID: "default", Name: "默认",
                Description: "通过 SealChat 协议接入。",
                Protocols: []*imconnm.ProtocolDefinition{
                    {Key: "sealchat", Name: "[WIP]SealChat", Platform: "SealChat", SchemaKey: "sealchat", Available: true, Description: "SealChat 协议。", Capabilities: baseCapabilities},
                },
            }},
        },
    }

    if s.dm.ContainerMode {
        for _, platform := range platforms {
            for _, method := range platform.Methods {
                for _, item := range method.Protocols {
                    if item.Key == "lagrange" || item.Key == "milky-internal" {
                        item.Available = false
                        item.DisabledReason = "当前为容器模式，内置客户端被禁用"
                    }
                }
            }
        }
    }
    return platforms
}
```

- [ ] **Step 2: 更新 `newService` 以构建树并生成 `protocolBy` 映射**

替换 `newService` 函数（第 37-51 行）为：

```go
func newService(dm *dice.DiceManager, autoServe bool, autoSave bool) *Service {
    _ = loadForms()
    s := &Service{
        dice:       dm.GetDice(),
        dm:         dm,
        autoServe:  autoServe,
        autoSave:   autoSave,
        protocolBy: map[string]*imconnm.ProtocolDefinition{},
    }
    s.protocols = s.buildProtocolTree()
    for _, platform := range s.protocols {
        for _, method := range platform.Methods {
            for _, p := range method.Protocols {
                s.protocolBy[p.Key] = p
            }
        }
    }
    return s
}
```

注意：这里 `protocols` 字段类型仍是 `[]*imconnm.ProtocolDefinition`（未改），但实际存储的是 `[]*imconnm.PlatformTreeNode`。为避免类型不匹配，需要同步改 Service 的字段类型，或者换个字段名。推荐在 Service struct 上把 `protocols` 重命名为 `protocolTree`，类型改为 `[]*imconnm.PlatformTreeNode`。

因此先改 Service struct（第 20-27 行）：

```go
type Service struct {
    dice         *dice.Dice
    dm           *dice.DiceManager
    autoServe    bool
    autoSave     bool
    protocolTree []*imconnm.PlatformTreeNode
    protocolBy   map[string]*imconnm.ProtocolDefinition
}
```

- [ ] **Step 3: 更新 `GetProtocols` handler**

替换第 149-151 行为：

```go
func (s *Service) GetProtocols(_ context.Context, _ *request.Empty) (*response.ItemResponse[imconnm.ProtocolListResp], error) {
    return response.NewItemResponse[imconnm.ProtocolListResp](imconnm.ProtocolListResp{Items: s.protocolTree}), nil
}
```

- [ ] **Step 4: 编译检查**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui && go build ./api/v2/imconnection/...`
Expected: 编译通过

---

### Task 3: 后端 — 更新测试

**Files:**
- Modify: `api/v2/imconnection/service_test.go`

- [ ] **Step 1: 重写 `protocolByKey` helper 以支持树形遍历**

```go
func protocolByKey(tree []*imconnm.PlatformTreeNode, key string) *imconnm.ProtocolDefinition {
    for _, platform := range tree {
        for _, method := range platform.Methods {
            for _, item := range method.Protocols {
                if item != nil && item.Key == key {
                    return item
                }
            }
        }
    }
    return nil
}
```

- [ ] **Step 2: 更新 `TestGetProtocolsReturnsCapabilitiesAndContainerAvailability` 中的调用**

```go
items := resp.Body.Item.Items
lagrange := protocolByKey(items, "lagrange")
```

其余断言保持不变。

- [ ] **Step 3: 运行测试**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui && go test ./api/v2/imconnection/... -v`
Expected: 全部 PASS

---

### Task 4: 重新生成前端 OpenAPI 类型

**Files:**
- Auto-generated: `ui/src/api/generated/*`

- [ ] **Step 1: 生成 OpenAPI JSON**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui/ui && pnpm run generate-openapi`
Expected: 在 `ui/openapi.json` 生成更新后的 spec（`ProtocolListResp.items` 变为 `PlatformTreeNode[]`）

- [ ] **Step 2: 生成 TypeScript 客户端**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui/ui && pnpm run generate-client`
Expected: `ui/src/api/generated/types.gen.ts` 中出现 `PlatformTreeNode`、`MethodTreeNode` 等新类型

- [ ] **Step 3: 验证新类型存在**

Run: `grep -n "PlatformTreeNode\|MethodTreeNode" /home/pinenut/GolandProjects/sealdice-core-newui/ui/src/api/generated/types.gen.ts`
Expected: 输出行号，确认新类型已生成

---

### Task 5: 前端 — 在 connect.vue 中实现 Steps 向导

**Files:**
- Modify: `ui/src/pages/connect.vue`

核心原则：只改"添加账号" Modal（`dialogVisible`），保留编辑 Modal、二维码 Modal、连接列表表格及所有 mutations。

#### 新增状态与计算属性

- [ ] **Step 1: 在 `<script setup>` 顶部新增 wizard 相关状态（放在现有 ref 之后）**

在 `const editFormModel = ref<DynamicFormModel>({});` 之后插入：

```ts
// Step wizard state
const wizardStep = ref(1);
const wizardPlatform = ref<PlatformTreeNode | null>(null);
const wizardMethod = ref<MethodTreeNode | null>(null);
const wizardProtocol = ref<ProtocolDefinition | null>(null);
```

- [ ] **Step 2: 更新 `protocols` computed 以适配树形响应**

把 `const protocols = computed(() => protocolsQuery.data.value?.item.items ?? []);`
改为：

```ts
const protocols = computed(() => (protocolsQuery.data.value?.item.items ?? []) as PlatformTreeNode[]);
```

同时新增一个把所有协议拍平的 helper：

```ts
const allProtocols = computed<ProtocolDefinition[]>(() => {
    const result: ProtocolDefinition[] = [];
    for (const platform of protocols.value) {
        for (const method of platform.methods ?? []) {
            for (const protocol of method.protocols ?? []) {
                result.push(protocol);
            }
        }
    }
    return result;
});
```

- [ ] **Step 3: 更新 `selectedProtocol` computed 以使用 `allProtocols`**

把 `const selectedProtocol = computed(() => protocols.value.find(...) ?? null);`
改为：

```ts
const selectedProtocol = computed(
    () => allProtocols.value.find(item => item.key === selectedProtocolKey.value) ?? null
);
```

- [ ] **Step 4: 删除旧的 `protocolOptions` computed**

把第 93-101 行的 `protocolOptions` 删除（步骤选择不再使用 `<n-select>`）。

- [ ] **Step 5: 删除 `watch(protocols, ...)` 自动选择逻辑**

把第 116-124 行的 `watch(protocols, ...)` 删除（步骤向导不需要自动预选）。

- [ ] **Step 6: 新增 wizard 导航方法与重置**

在 `submit` 函数之前插入：

```ts
const wizardCanNext = computed(() => {
    switch (wizardStep.value) {
        case 1: return !!wizardPlatform.value;
        case 2: return !!wizardMethod.value;
        case 3: {
            const p = wizardProtocol.value;
            return !!p && p.available && !p.deprecated;
        }
        case 4: return canSubmit.value;
    }
    return false;
});

const goNext = () => {
    if (wizardStep.value === 3 && wizardProtocol.value) {
        selectedProtocolKey.value = wizardProtocol.value.key;
        formModel.value = buildDynamicFormInitialModel(selectedSchema.value);
    }
    if (wizardStep.value < 4) {
        wizardStep.value++;
    }
};

const goPrev = () => {
    if (wizardStep.value > 1) {
        wizardStep.value--;
    }
};

const resetWizard = () => {
    wizardStep.value = 1;
    wizardPlatform.value = null;
    wizardMethod.value = null;
    wizardProtocol.value = null;
    selectedProtocolKey.value = '';
    formModel.value = {};
};
```

- [ ] **Step 7: 更新 `openCreateDialog`**

把 `const openCreateDialog = () => { dialogVisible.value = true; };`
改为：

```ts
const openCreateDialog = () => {
    resetWizard();
    dialogVisible.value = true;
};
```

- [ ] **Step 8: 更新 `createMutation` 的 `onSuccess`**

把 `onSuccess` 里的 `dialogVisible.value = false;` 改成：

```ts
onSuccess: () => {
    message.success('账号已添加');
    dialogVisible.value = false;
    resetWizard();
},
```

#### 模板改造

- [ ] **Step 9: 替换"添加账号" Modal 的模板内容**

把整个 `<n-modal v-model:show="dialogVisible" ...>` 到 `</n-modal>`（第 528-649 行）替换为以下 TSX 模板：

```tsx
    <n-modal
      v-model:show="dialogVisible"
      preset="dialog"
      title="添加账号"
      class="account-dialog wizard-dialog"
      :show-icon="false"
      :mask-closable="false"
      @after-leave="resetWizard"
    >
      <n-space vertical size="large">
        <n-steps :current="wizardStep" size="small">
          <n-step title="选择平台" />
          <n-step title="选择方式" />
          <n-step title="选择协议" />
          <n-step title="填写信息" />
        </n-steps>

        <!-- Step 1: 选择平台 -->
        <div v-if="wizardStep === 1" class="wizard-step-panel">
          <div class="option-cards">
            <n-card
              v-for="platform in protocols"
              :key="platform.id"
              hoverable
              :class="['option-card', { 'option-card--selected': wizardPlatform?.id === platform.id }]"
              @click="wizardPlatform = platform"
            >
              <div class="option-card-title">{{ platform.name }}</div>
              <div class="option-card-desc">{{ platform.description }}</div>
            </n-card>
          </div>
        </div>

        <!-- Step 2: 选择方式 -->
        <div v-if="wizardStep === 2" class="wizard-step-panel">
          <div class="option-cards">
            <n-card
              v-for="method in wizardPlatform?.methods"
              :key="method.id"
              hoverable
              :class="['option-card', { 'option-card--selected': wizardMethod?.id === method.id }]"
              @click="wizardMethod = method"
            >
              <div class="option-card-title">{{ method.name }}</div>
              <div class="option-card-desc">{{ method.description }}</div>
            </n-card>
          </div>
        </div>

        <!-- Step 3: 选择协议 -->
        <div v-if="wizardStep === 3" class="wizard-step-panel">
          <div class="option-cards">
            <n-card
              v-for="protocol in wizardMethod?.protocols"
              :key="protocol.key"
              hoverable
              :class="[
                'option-card',
                { 'option-card--selected': wizardProtocol?.key === protocol.key },
                { 'option-card--disabled': !protocol.available },
              ]"
              @click="protocol.available ? (wizardProtocol = protocol) : null"
            >
              <div class="option-card-title">
                {{ protocol.name }}
                <n-tag v-if="protocol.deprecated" type="warning" size="small">已废弃</n-tag>
                <n-tag v-else-if="!protocol.available" type="error" size="small">不可用</n-tag>
              </div>
              <div class="option-card-desc">{{ protocol.description }}</div>
              <n-alert
                v-if="!protocol.available && protocol.disabledReason"
                type="warning"
                :show-icon="false"
                class="mt-2"
              >
                {{ protocol.disabledReason }}
              </n-alert>
            </n-card>
          </div>
        </div>

        <!-- Step 4: 填写信息 -->
        <div v-if="wizardStep === 4" class="wizard-step-panel">
          <n-alert v-if="selectedProtocol && !selectedProtocol.available" type="warning" :show-icon="false">
            {{ selectedProtocol.disabledReason }}
          </n-alert>

          <n-alert v-if="schemasQuery.error.value" type="error" :show-icon="false">
            配置项读取失败，请稍后重试。
          </n-alert>

          <DynamicForm
            v-model="formModel"
            :schema="selectedSchema"
            :disabled="createMutation.isPending.value"
          >
            <template #field="{ item, fieldKey, value, setValue }">
              <AsyncFieldSection
                v-if="selectedProtocolKey === 'lagrange' && fieldKey === 'signServerVersion'"
                :loading="signInfoState.mode === 'loading'"
                :message="signInfoState.message"
                :error="signInfoErrorMessage"
                @retry="retrySignInfo"
              >
                <n-select
                  :value="value as string"
                  :options="signVersionOptions"
                  :disabled="!signInfoState.canSelectVersion"
                  placeholder="请选择签名版本"
                  @update:value="setValue"
                />
              </AsyncFieldSection>
              <AsyncFieldSection
                v-else-if="selectedProtocolKey === 'lagrange' && fieldKey === 'signServerName'"
                :loading="signInfoState.mode === 'loading'"
                :message="signInfoState.mode === 'manual-fallback' ? '' : signInfoState.message"
                :error="fieldKey === 'signServerName' ? signInfoErrorMessage : ''"
                @retry="retrySignInfo"
              >
                <n-select
                  v-if="!signInfoState.showCustomServerInput"
                  :value="value as string"
                  :options="signServers"
                  :disabled="!signInfoState.canSelectServer"
                  placeholder="请选择签名服务"
                  @update:value="setValue"
                />
                <n-input
                  v-else
                  :value="value as string"
                  placeholder="请输入自定义签名地址"
                  @update:value="setValue"
                />
              </AsyncFieldSection>
              <n-input
                v-else-if="item.input_type === 0"
                :value="value as string"
                :type="item.sensitive ? 'password' : 'text'"
                :placeholder="item.placeholder"
                show-password-on="mousedown"
                @update:value="setValue"
              />
            </template>
          </DynamicForm>
        </div>
      </n-space>

      <template #action>
        <n-button @click="dialogVisible = false">
          取消
        </n-button>
        <n-button v-if="wizardStep > 1" @click="goPrev">
          上一步
        </n-button>
        <n-button
          v-if="wizardStep < 4"
          type="primary"
          :disabled="!wizardCanNext"
          @click="goNext"
        >
          下一步
        </n-button>
        <n-button
          v-if="wizardStep === 4"
          type="primary"
          :loading="createMutation.isPending.value"
          :disabled="!canSubmit"
          @click="submit"
        >
          添加
        </n-button>
      </template>
    </n-modal>
```

注意：需要在 `<script setup>` 的 import 区域更新类型导入，把 `ProtocolDefinition` 保留，并确保从生成的 SDK 中也能拿到 `PlatformTreeNode` 和 `MethodTreeNode`。

- [ ] **Step 10: 更新 import 类型**

在 connect.vue 的 import 块中，确保有：

```ts
import {
  // ...existing imports...
  type ProtocolDefinition,
  type PlatformTreeNode,
  type MethodTreeNode,
} from '@/api';
```

由于这些类型是重新生成的，具体名称可能为 `ProtocolDefinition`、`PlatformTreeNode`、`MethodTreeNode`（ PascalCase 对应 Go 结构体名）。若生成器命名不同，以实际生成为准。

- [ ] **Step 11: 添加 wizard 样式**

在 `<style scoped>` 末尾追加：

```css
.wizard-dialog {
  max-width: 640px;
}

.wizard-step-panel {
  min-height: 200px;
}

.option-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.option-card {
  cursor: pointer;
  transition: all 0.2s ease;
}

.option-card:hover {
  border-color: #1d4ed8;
}

.option-card--selected {
  border-color: #1d4ed8;
  background-color: #eff6ff;
}

.option-card--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.option-card-title {
  font-weight: 600;
  font-size: 1rem;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.option-card-desc {
  font-size: 0.85rem;
  color: #6b7280;
  line-height: 1.4;
}

.mt-2 {
  margin-top: 8px;
}
```

- [ ] **Step 12: 前端 type-check**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui/ui && pnpm run type-check`
Expected: 无 TypeScript 错误

---

### Task 6: 端到端验证

- [ ] **Step 1: 编译后端**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui && go build ./...`
Expected: 编译成功

- [ ] **Step 2: 运行后端测试**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui && go test ./api/v2/imconnection/... -v`
Expected: 全部 PASS

- [ ] **Step 3: 前端构建**

Run: `cd /home/pinenut/GolandProjects/sealdice-core-newui/ui && pnpm run build`
Expected: 构建成功

---

### Task 7: 提交

- [ ] **Step 1: 提交所有改动**

```bash
cd /home/pinenut/GolandProjects/sealdice-core-newui
git add -A
git commit -m "feat: 账号添加向导改为平台→方式→协议→填信息四步流程

- 后端 /protocols API 返回树形结构 PlatformTreeNode[]
- 每个平台/方式/协议增加 description 字段
- 前端使用 n-steps 实现四步向导
- 保留编辑/列表/二维码功能不变"
```

---

## 自我审查

**Spec coverage:**
- ✅ 后端树形 API — Task 1-3
- ✅ 废弃协议保留并标记 — Task 5 Step 9（`protocol.deprecated` tag）
- ✅ 四层 Step 向导（平台→方式→协议→填信息）— Task 5
- ✅ 每层有介绍框 — Task 5 Step 9（`option-card-desc`）
- ✅ 手动"下一步"推进 — Task 5 Step 6（`goNext`/`goPrev`）
- ✅ 非 QQ 平台也显示方式步骤（默认）— Task 2 树形定义（所有平台都有 methods）
- ✅ 动态表单复用 — Task 5 Step 9（Step 4 直接使用 `<DynamicForm>`）
- ✅ 编辑流程不变 — 未修改 editModal 和 edit mutations
- ✅ 旧前端不兼容 — 无需处理

**Placeholder scan:**
- 无 TBD、TODO、"add appropriate error handling" 等占位符。每个步骤都包含完整代码。

**Type consistency:**
- `PlatformTreeNode`、`MethodTreeNode`、`ProtocolDefinition` 在后端（Go）和前端（TS 生成类型）中名称一致。
- `wizardCanNext`、`goNext`、`goPrev`、`resetWizard` 命名在 Task 5 各步骤中保持一致。

**One gap identified and fixed:**
- 原设计未明确提及 `protocolBy` 平面映射仍需保留用于创建/编辑；已在 Task 2 Step 2 中补充 `newService` 遍历树生成 `protocolBy` 的逻辑。
