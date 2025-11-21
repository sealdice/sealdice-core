# 扩展激活机制重构说明

## 概述

本文档说明了新的扩展激活机制的设计和实现。新机制通过使用两个列表（已激活列表和已关闭列表）来简化扩展的激活判断逻辑，使代码更加清晰易懂。

## 背景

### 旧机制的问题

在旧的扩展激活机制中，存在以下数据结构：

1. `ActivatedExtList` - 已激活的扩展列表
2. `ExtActiveListSnapshot` - 用于保持插件顺序的快照（包含所有曾激活过的扩展名）
3. `ExtDisabledByUser` - 用户手动禁用的扩展 map

判断新扩展是否应该激活的逻辑非常复杂：
- 需要检查是否在 snapshot 中
- 需要检查是否在 DisabledByUser 中
- 需要检查是否首次加载
- 需要检查 AutoActive 选项

这导致代码难以理解和维护。

## 新机制设计

### 数据结构

新机制使用以下数据结构：

1. **`ActivatedExtList []*ExtInfo`** - 当前已激活的扩展列表
2. **`InactivatedExtList []string`** - 当前已关闭的扩展名称列表（新增）
3. **`ExtActiveListSnapshot []string`** - 用于维持扩展激活顺序（保留）

移除了 `ExtDisabledByUser map[string]bool`，其功能由 `InactivatedExtList` 替代。

**设计原则**：
- `ActivatedExtList` 存储完整的扩展对象，便于直接使用
- `InactivatedExtList` 只存储扩展名称，节省内存，与 snapshot 设计保持一致
- 两个列表清晰记录扩展的激活/关闭状态，无需复杂推断

### 核心逻辑

新的判断逻辑非常简单直观：

```go
// 判断是否是新扩展
if 扩展既不在ActivatedExtList，也不在InactivatedExtList {
    // 这是新扩展，根据 AutoActive 选项决定是否激活
    if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
        激活它，并加入 ActivatedExtList 和 ExtActiveListSnapshot
    } else {
        加入 InactivatedExtList
    }
} else if 扩展在InactivatedExtList中 {
    // 用户曾经关闭过，保持关闭状态
    不激活
} else {
    // 扩展在 ActivatedExtList 中，保持激活状态
}
```

## 实现细节

### 新增辅助方法

```go
// IsExtInActivatedList 检查扩展是否在激活列表中
func (group *GroupInfo) IsExtInActivatedList(name string) bool

// IsExtInInactivatedList 检查扩展是否在关闭列表中
func (group *GroupInfo) IsExtInInactivatedList(name string) bool

// IsExtKnown 检查扩展是否已知（在激活或关闭列表中）
func (group *GroupInfo) IsExtKnown(name string) bool

// RemoveFromInactivatedList 从关闭列表中移除扩展
func (group *GroupInfo) RemoveFromInactivatedList(name string)

// AddToInactivatedList 添加扩展名称到关闭列表
func (group *GroupInfo) AddToInactivatedList(name string)
```

### 修改的方法

#### 1. ExtActive

当用户手动激活一个扩展时，会自动从关闭列表中移除：

```go
func (group *GroupInfo) ExtActive(ei *ExtInfo) {
    // 从关闭列表中移除
    group.RemoveFromInactivatedList(ei.Name)
    
    // 添加到激活列表
    // ...
}
```

#### 2. ExtInactive / ExtInactiveByName

当用户手动关闭一个扩展时，会自动加入关闭列表：

```go
func (group *GroupInfo) ExtInactive(ei *ExtInfo) *ExtInfo {
    // 从激活列表中移除
    // ...
    
    // 加入关闭列表
    group.AddToInactivated(ei.Name)
    
    // 从快照中移除
    // ...
}
```

#### 3. ExtActiveBySnapshotOrder

现在的实现只是对批量逻辑的薄封装，确保所有入口共享同一套排序/快照/连带激活规则：

```go
func (group *GroupInfo) ExtActiveBySnapshotOrder(ei *ExtInfo, isFirstTimeLoad bool) {
    if ei == nil {
        return
    }
    var firstLoad map[string]bool
    if isFirstTimeLoad {
        firstLoad = map[string]bool{ei.Name: true}
    }
    group.ExtActiveBatchBySnapshotOrder([]*ExtInfo{ei}, firstLoad)
}
```

#### 4. ExtActiveBatchBySnapshotOrder

批量逻辑会按以下顺序处理整批扩展：

1. 以 `ExtActiveListSnapshot` 为基准，按快照顺序取出现有扩展（忽略被用户禁用的项）。
2. 遍历待激活扩展：
   - 如果是首次加载且不在快照中，直接追加并写入快照；
   - 如果声明了 `AutoActive` 且不在快照中、也未被禁用，则自动恢复。
3. 对每个扩展执行 `ActiveWith` 链式激活，保证依赖扩展也被追加并更新快照。

这样无论是一次加载多个扩展还是单独开启，都会得到一致的行为。

### 移除的方法

以下方法已被移除，不再需要：

- `IsUserDisabled(name string) bool`
- `SetUserDisabled(name string)`
- `ClearUserDisabledFlag(name string)`

## 优势

### 1. 逻辑清晰

新机制的判断逻辑非常直观：
- 在激活列表 → 保持激活
- 在关闭列表 → 保持关闭
- 两者都不在 → 新扩展，根据 AutoActive 决定

### 2. 易于维护

通过两个明确的列表，可以轻松追踪扩展的状态，无需复杂的检查逻辑。

### 3. 数据一致性

使用集合（由 map 实现）保证了数据结构的一致性，避免了同步问题。

### 4. 功能完整

新机制完全保留了旧机制的所有功能：
- 支持扩展的自动激活
- 记住用户手动关闭的扩展
- 支持扩展的优先级排序
- 支持插件重载后恢复状态

## 迁移指南

### 数据迁移

旧的群组数据中可能存在 `ExtDisabledByUser` 字段。新机制会自动处理：

1. 如果群组数据中存在旧的 `ExtDisabledByUser` 字段，它会被忽略
2. 新机制会初始化 `InactivatedExtList` 为空列表
3. 当扩展首次加载时，根据 AutoActive 自动决定是否激活

### API 变更

如果你的代码中使用了以下方法，需要更新：

| 旧方法 | 新方法 | 参数变化 |
|--------|--------|----------|
| `IsUserDisabled(name)` | `IsExtInactivated(name)` | 无变化 |
| `SetUserDisabled(name)` | `AddToInactivated(name)` | 无变化 |
| `ClearUserDisabledFlag(name)` | `RemoveFromInactivated(name)` | 无变化 |

**重要**：`AddToInactivatedList` 的参数已从 `*ExtInfo` 改为 `string`（只需传扩展名称）

## 测试

所有现有的单元测试均已通过，确保新机制与旧机制行为一致。

```bash
go test ./dice/...
# ok  	sealdice-core/dice	0.927s
# ok  	sealdice-core/dice/censor	0.775s
```

## 总结

新的扩展激活机制通过引入 `InactivatedExtSet`，简化了扩展状态管理，使代码更加清晰易懂。核心思想是：**用一个明确的列表和一个集合来记录扩展的激活和关闭状态，而不是通过复杂的检查逻辑来推断**。
