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
2. **`InactivatedExtSet StringSet`** - 当前已关闭的扩展名称集合（新增）

移除了 `ExtDisabledByUser map[string]bool`，其功能由 `InactivatedExtSet` 替代。
移除了 `ExtActiveListSnapshot []string`，Wrapper 机制已使其不再必要。

**设计原则**：
- `ActivatedExtList` 存储完整的扩展对象，便于直接使用
- `InactivatedExtSet` 使用集合（`StringSet`，内部为 `map[string]struct{}`）存储扩展名称，高效查询，节省内存
- 列表和集合清晰记录扩展的激活/关闭状态，无需复杂推断
- Wrapper 机制确保热重载时扩展顺序保持不变

### 核心逻辑

新的判断逻辑非常简单直观：

```go
// 判断是否是新扩展
if 扩展既不在ActivatedExtList，也不在InactivatedExtSet {
    // 这是新扩展，根据 AutoActive 选项决定是否激活
    if ext.AutoActive || (ext.DefaultSetting != nil && ext.DefaultSetting.AutoActive) {
        激活它，加入 ActivatedExtList
    } else {
        加入 InactivatedExtSet
    }
} else if 扩展在InactivatedExtSet中 {
    // 用户曾经关闭过，保持关闭状态
    不激活
} else {
    // 扩展在 ActivatedExtList 中，保持激活状态
}
```

## 实现细节

### 新增辅助方法

```go
// IsExtInactivated 检查扩展是否在关闭集合中
func (group *GroupInfo) IsExtInactivated(name string) bool

// RemoveFromInactivated 从关闭集合中移除扩展
func (group *GroupInfo) RemoveFromInactivated(name string)

// AddToInactivated 添加扩展名称到关闭集合
func (group *GroupInfo) AddToInactivated(name string)
```

### 修改的方法

#### 1. ExtActive

当用户手动激活一个扩展时，会自动从关闭集合中移除：

```go
func (group *GroupInfo) ExtActive(ei *ExtInfo) {
    // 从关闭集合中移除
    group.RemoveFromInactivated(ei.Name)

    // 添加到激活列表
    // ...
}
```

#### 2. ExtInactive / ExtInactiveByName

当用户手动关闭一个扩展时，会自动加入关闭集合：

```go
func (group *GroupInfo) ExtInactive(ei *ExtInfo) *ExtInfo {
    // 从激活列表中移除
    // ...

    // 加入关闭集合
    group.AddToInactivated(ei.Name)
}
```

### 移除的方法

以下方法已被移除，不再需要：

- `IsUserDisabled(name string) bool`
- `SetUserDisabled(name string)`
- `ClearUserDisabledFlag(name string)`
- `ExtActiveBySnapshotOrder(ei *ExtInfo, isFirstTimeLoad bool)` - Wrapper 机制使快照不再必要
- `ExtActiveBatchBySnapshotOrder(extInfos []*ExtInfo, isFirstTimeLoad map[string]bool)` - 同上
- `ensureSnapshotFromActivated()` - 快照机制已移除

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
- Wrapper 机制确保热重载后扩展顺序保持不变

## 迁移指南

### 数据迁移

旧的群组数据中可能存在 `ExtDisabledByUser` 字段。新机制会自动处理：

1. 如果群组数据中存在旧的 `ExtDisabledByUser` 字段，它会被忽略
2. 新机制会初始化 `InactivatedExtSet` 为空集合
3. 当扩展首次加载时，根据 AutoActive 自动决定是否激活

### API 变更

如果你的代码中使用了以下方法，需要更新：

| 旧方法 | 新方法 | 说明 |
|--------|--------|------|
| `IsUserDisabled(name)` | `IsExtInactivated(name)` | 检查扩展是否被标记为手动关闭 |
| `SetUserDisabled(name)` | `AddToInactivated(name)` | 将扩展添加到关闭集合 |
| `ClearUserDisabledFlag(name)` | `RemoveFromInactivated(name)` | 从关闭集合中移除扩展 |

**注意**：所有方法的参数都是 `string`（扩展名称），不需要传递 `*ExtInfo` 对象。

## 测试

所有现有的单元测试均已通过，确保新机制与旧机制行为一致。

```bash
go test ./dice/...
# ok  	sealdice-core/dice	0.927s
# ok  	sealdice-core/dice/censor	0.775s
```

## 总结

新的扩展激活机制通过引入 `InactivatedExtSet`，简化了扩展状态管理，使代码更加清晰易懂。核心思想是：**用一个激活列表（`ActivatedExtList`）和一个关闭集合（`InactivatedExtSet`）来明确记录扩展的状态，而不是通过复杂的检查逻辑来推断**。

使用 `StringSet`（基于 `map[string]struct{}`）作为关闭集合的优势：
- 高效的 O(1) 查询性能
- 自动去重，避免重复记录
- 内存占用小（只存储键，值为空结构体）
- 支持 YAML 序列化为有序的字符串数组，提高可读性
