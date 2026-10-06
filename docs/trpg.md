# `trpg` 通用跑团扩展

## 定位与兼容性

`trpg` 是默认激活的内置扩展，集中提供通用 `.st`、`.sn` 和 `.team` 指令。其中 `.st` 拥有独立于 `coc7.st` 和 `dnd5e.st` 的事务与 Hook 实现。新规则模板可以这样选择它：

```yaml
set:
  relatedExt:
    - trpg
    - my-rule
```

指令提供者继续遵循现有的 `relatedExt` 优先级：

- 仍声明 `relatedExt: [coc7, ...]` 的模板继续使用 `coc7.st`；
- DND5E 模板继续使用 `dnd5e.st`；
- 声明 `relatedExt: [trpg, ...]` 的模板优先使用 `trpg.st`；
- `trpg.st` 未注册任何 hook 时，属性解析和数据变更行为与通用 `coc7.st` 保持一致；回复使用独立的 `TRPG:*` 文本模板。

`.sn` 与 `.team` 的既有实现和数据结构完整迁移至 `trpg`。由于 `trpg` 默认激活，常规使用方式保持不变。系统不会额外检测多个 `.st` 提供者之间的冲突。

## 注册

第三方 JS 扩展通过 `seal.trpg.st.registerHooks` 注册一组回调。一个扩展重复注册时，新注册内容覆盖旧内容；卸载或重载扩展后不会继续调用旧运行时中的回调。

```javascript
let ext = seal.ext.find("my-rule");
if (!ext) {
  ext = seal.ext.new("my-rule", "Author", "1.0.0");
  seal.ext.register(ext);
}

seal.trpg.st.registerHooks(ext, {
  systems: ["my-rule"],

  afterEvaluate(event, operation) {
    if (operation.name !== "生命值" || operation.proposedValue.type !== "int") {
      return;
    }

    const upper = 20;
    const lower = 0;
    const value = operation.proposedValue.intValue;
    operation.proposedValue = seal.trpg.st.newIntValue(
      Math.max(lower, Math.min(upper, value)),
    );
  },
});
```

`systems` 为空或省略时，对所有关联该扩展的规则模板生效。指定 `systems` 后只对名称匹配的模板生效。Hook 扩展还必须位于当前规则模板的 `relatedExt` 层级中并处于激活状态，普通已激活但与当前规则无关的扩展不会收到事件。

可以使用 `seal.trpg.st.clearHooks(ext)` 清除扩展注册的整组 hook。

## 执行阶段

一次 `.st` 调用按以下阶段执行：

1. `beforeCommand(event)`：解析子命令前调用；可以设置 `event.handled = true` 和 `event.reply` 完全接管调用。
2. `afterParse(event)`：生成 `event.operations` 后调用；可以调整顺序、删除操作或加入新操作。
3. `beforeApply(event, operation)`：每项操作求值前调用。
4. `afterEvaluate(event, operation)`：已经得到旧值和建议新值，但尚未提交该项时调用。
5. `beforeCommit(event)`：所有操作处理完成、整批提交前调用。
6. `afterCommit(event)`：提交成功后调用。此阶段发生的异常只记录日志，不再回滚已经提交的数据。

展示与导出分别调用 `onShow(renderEvent)` 和 `onExport(renderEvent)`。回调可以修改 `renderEvent.text`，或设置 `renderEvent.skip = true` 隐藏该项。

`set`、`mod`、`delete`、`clear` 和 `format` 是当前定义的操作种类。`set` 和 `mod` 操作会在 `afterEvaluate` 阶段提供 `oldValue` 与 `proposedValue`。

## 修改、拒绝与提交

常用字段如下：

- `operation.skip = true`：跳过当前操作，不拒绝同批的其他操作；
- `operation.rejectReason = "原因"`：拒绝整批操作；
- `event.rejectReason = "原因"`：拒绝整条命令；
- `operation.proposedValue = ...`：替换即将写入的值；
- `operation.appendText`：给属性增减回复追加文本；
- `event.replySuffix`：给成功回复追加文本。

只要在提交前的 hook 中抛出异常或设置拒绝原因，本次修改涉及的属性、卡片类型和简化录卡中的角色名都会恢复到命令执行前的状态，不会留下只写入一部分属性的结果。

Hook 应通过事件和操作对象修改本次 `.st` 事务。不要在 hook 内另外调用 `seal.vars.*Set` 修改人物卡：这类旁路写入不属于稳定的事务接口，尤其是在 `beforeCommand`、展示或导出阶段不受 `.st` 回滚保证约束。

## 值与操作构造器

`TrpgStValue` 对 JS 暴露以下字段：

- `type`：`int`、`float`、`string`、`null`、`computed` 或 `opaque`；
- `intValue`、`floatValue`、`stringValue`：对应基础类型的值；
- `expression`：计算属性表达式；
- `repr`：用于读取和展示的 DiceScript 表示。

替换 `operand` 或 `proposedValue` 时应使用构造器，避免手工构造不完整的值对象：

```javascript
seal.trpg.st.newIntValue(10);
seal.trpg.st.newFloatValue(1.5);
seal.trpg.st.newStringValue("text");
seal.trpg.st.newComputedValue("力量 / 2");
```

在 `afterParse` 中加入操作时，可以使用：

```javascript
afterParse(event) {
  const operation = seal.trpg.st.newOperation("set", "幸运");
  operation.operand = seal.trpg.st.newIntValue(50);
  event.operations = [operation].concat(event.operations);
}
```

`opaque` 值可以原样保留和读取，但不能由第三方自行构造。当前 API 不承诺直接修改 `repr` 会改变实际写入值。
