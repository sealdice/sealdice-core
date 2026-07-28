# 数据库升级迁移总览（migrate/v2）

本文档汇总 `migrate/v2` 下注册的全部数据库升级迁移，供审阅其行为是否符合预期。

## 升级框架工作原理

- **入口**：`migrate/v2/enter.go` 的 `InitUpgrader(operator)` 创建 `upgrade.Manager`，依次 `Register` 所有迁移，然后 `ApplyAll()`。
- **排序**：`ApplyAll` 按 **迁移 ID 的字符串字典序升序** 逐个应用。因此 ID 前缀的数字决定了执行顺序（`001_` < `002_` < … < `010_`）。
- **幂等 / 去重**：每个迁移应用前先问 `Store.IsApplied(id)`；`GormStore`（`data.db` 的 `upgrade_records` 表）记录迁移状态，再次启动会跳过。
- **失败处理**：任意迁移返回错误时，`ApplyAll` 立即中止，并把错误向上抛（“因无法忽略的错误，升级 X 失败”）。已成功的迁移不会被回滚，下次启动会从失败的那个继续。
- **记录**：无论成功失败，都会写一条 `UpgradeRecord`（含时间、成功标志、日志）到 `data.db` 的 `upgrade_records` 表。
- **旧格式清理**：启动时删除旧版 `upgrade_metadata.json`（升级状态已迁入 `data.db`）。

> 字段约定：下文“触发条件”指迁移函数内部的“是否需要真正干活”判断；“幂等”指**即使 Store 没拦住、重复执行同一迁移函数**是否安全。

## 迁移清单

| ID | 版本 | 名称 | 一句话作用 |
|----|------|------|-----------|
| `001_V120Migration` | v1.2.0 | 配置与日志入库 | 把旧 `serve.yaml` 配置 + BoltDB 日志（`data.bdb`）迁入 SQLite |
| `002_V120LogMessageMigration` | v1.2.x→1.3.1 | log_items 类型修复 | 修复 log_items.message 被错误建成 INTEGER 的问题（仅 SQLite） |
| `003_V131ConfigUpdateMigration` | v1.3.1 | 弃用配置迁移 | 把 serve.yaml 中若干弃用项迁到 `text-template.yaml` |
| `004_V141ConfigUpdateMigration` | v1.4.1 | 配置项改名 | `customReplenishRate→personalReplenishRate`、`customBurst→personalBurst` |
| `005_V144RemoveOldHelpDocMigration` | v1.4.4 | 清理旧帮助文档 | 校验哈希后删除旧版“蜜瓜包-怪物之锤查询.json” |
| `006_V150UpgradeAttrsMigration` | v1.5.0 | attrs 表统一重构 | 合并 attrs_user/attrs_group/attrs_group_user → 统一 `attrs`，角色卡数据转 V2 格式 |
| `007_V150FixGroupInfoMigration` | v1.5.0 | GroupInfo 清洗 | 删除 group_info 中 created_at/updated_at 异常或 data 为空的坏行 |
| `007_V151GORMCleanMigration` | v1.5.1 | GORM 脏数据清理 | 删除 ban_info、attrs 中 data 为 NULL/空的行 |
| `008_V160LogIDZeroCleanMigration` | v1.6.0 | log_id=0 清理 | 删除 log_items.log_id=0 与 logs.id=0 的残留并重算 size |
| `009_V160LogRawMsgIDIndexMigration` | v1.6.0 | 日志复合索引 | 为 log_items 建 `(group_id, raw_msg_id, id)` 复合索引 |
| `010_V160LogSizeRepairMigration` | v1.6.0 | logs.size 兜底修复 | 补建缺失的 size 列并全量重算（兜底 V150 失误） |

> ⚠️ ID 冲突提醒：`007_` 前缀同时被 `V150FixGroupInfoMigration` 与 `V151GORMCleanMigration` 使用，靠后缀字典序保证 V150 先于 V151 执行。代码内多处 `TODO` 标注“需要合理的生成逻辑”，建议后续改为更稳健的编号方案。

---

## 各迁移详解

### 001 — V120Migration（配置与日志入库）

- **触发条件**：存在 `./data/default/data.bdb`（旧 BoltDB 日志库）。不存在则直接返回（视为新版本或已迁移）。
- **自检守卫**：若新版 `attrs` 表已存在（说明 V150 已执行过），跳过迁移，直接将 `data.bdb` 重命名为 `data.bdb.migrated`。这避免了"V120 重建旧表 → V150 重跑时重复生成角色卡"的风险。
- **行为**：
  1. 经 sqlx 读旧库，把 `group_info`、`group_player_info`、`attrs_*`、`ban_info` 等配置数据写入 SQLite；
  2. 把 BoltDB 中的历史日志迁到新的 `logs` / `log_items` 表；
  3. 将原 `serve.yaml` 备份为 `serve.yaml.old`。
  4. **迁移成功后**将 `data.bdb` 重命名为 `data.bdb.migrated`（保留备份）。
- **幂等**：自检守卫 + `data.bdb` 重命名双重保证。`data.bdb` 不存在或已重命名为 `.migrated` 时直接跳过；`attrs` 表存在时也跳过（避免旧表被重建后与 V150 冲突）。
- **失败**：返回错误 → 中断整个升级。`data.bdb` 不会在此路径下被重命名，下次启动仍可重试。

### 002 — V120LogMessageMigration（log_items 类型修复）

- **触发条件**：仅 SQLite；且 `log_items` 建表语句匹配 `message\s+INTEGER,`（说明字段类型错了）。
- **行为**：重建 log_items 表，把 message 列改回 TEXT，迁移数据后删除旧表，并执行 `VACUUM`。
- **幂等**：是（通过建表语句正则判断，已修复则跳过）。
- **失败**：⚠️ 失败时**记录日志后返回 nil**（错误被故意吞掉），不会中断升级。

### 003 — V131ConfigUpdateMigration（弃用配置 → 自定义文案）

- **触发条件**：`serve.yaml` 存在且含待迁移的弃用键（骰主帮助信息、使用协议、骰子状态附加文本、抽牌列表文本）。
- **行为**：把这些值迁到 `text-template.yaml`，并备份原自定义文案文件。
- **幂等**：大致是（键不存在/已迁移则跳过）。
- **失败**：⚠️ 失败时**返回 nil**（非致命）。

### 004 — V141ConfigUpdateMigration（配置项改名）

- **触发条件**：`serve.yaml` 存在。
- **行为**：原地重命名两个字段：`customReplenishRate→personalReplenishRate`、`customBurst→personalBurst`。
- **幂等**：是（旧字段不存在则不改动）。
- **失败**：⚠️ 失败时**返回 nil**（非致命）。

### 005 — V144RemoveOldHelpDocMigration（清理旧帮助文档）

- **触发条件**：存在旧文件 `data/helpdoc/COC/蜜瓜包-怪物之锤查询.json`。
- **行为**：当且仅当旧文件、新文件 `怪物之锤查询.json` 的 SHA256 都与预期一致时，删除旧文件；任何校验不通过都跳过（不删）。
- **幂等**：是（旧文件不存在直接跳过）。
- **失败**：包装层会向上传错误（与 003/004 不同），但底层函数实际所有分支都返回 nil，故不会真正中断。

### 006 — V150UpgradeAttrsMigration（attrs 统一重构）★ 重点

- **触发条件**：始终执行（内部按表是否存在分别处理）。
- **行为**：
  1. **建表初始化** `dataDBInit` / `logDBInit` / `censorDBInit`：
     - SQLite 走“期望列清单”严格比对；列结构不符则**重建表**（建临时表→批量复制→改名），把表结构纠正到目标 schema。同时处理“前置测试版 150”残留的异常 attrs 表。
     - MySQL / PG 走 GORM `AutoMigrate`。
  2. **数据迁移**（事务内）：依次迁移 `attrs_user`（含 `$ch:*` 角色卡、`$:group-bind:*` 绑卡关系）、`attrs_group_user`、`attrs_group` 到统一 `attrs` 表；角色卡数据从 VMValue v1 转成 dicescript v2；维护群内绑卡关系。
  3. **清理**：删除旧表 `attrs_user` / `attrs_group` / `attrs_group_user`。
  4. **logs.size 计算**（`calculateLogSize`）：按 log_id 统计每条日志的条目数回填 size。
- **幂等**：是（旧表不存在则跳过对应迁移；建表走 IF NOT EXISTS / HasTable 判断）。
- **失败**：事务内出错会回滚该事务并返回错误 → 中断升级。
- **⚠️ 已知历史失误（正是 010 要兜底的）**：在某些历史版本中，本迁移的“建 size 列 / 计算 size”逻辑尚未就位就被应用并记为已完成，导致部分库的 `logs` 表**根本没有 size 列**。由于 Store 已记录 V150 完成、不会重跑，需要靠新的 010 迁移补救。

> size 语义说明：V150 的 `calculateLogSize` 统计的是**全部** log_items（不区分 removed）。这与后续 008/010 的“仅统计 removed IS NULL”口径存在差异（见末尾“size 语义”一节）。

### 007 — V150FixGroupInfoMigration（GroupInfo 清洗）

- **触发条件**：始终执行。
- **行为**：删除 `group_info` 中满足“created_at/updated_at 异常（≤0 或 NULL）”或“data 为空”的行。
- **幂等**：是（再跑删除 0 行）。
- **失败**：返回错误 → 中断升级。

### 007 — V151GORMCleanMigration（GORM 脏数据清理）

- **触发条件**：始终执行。
- **行为**：删除 `ban_info`、`attrs` 中 `data IS NULL OR data = ''` 的行。
- **幂等**：是。
- **失败**：返回错误 → 中断升级。
- **⚠️ 实现瑕疵**：`Pluck` 取待删 ID 时未检查错误（静默忽略），但实际删除操作 `Delete` 的错误会被返回。可后续加固。

### 008 — V160LogIDZeroCleanMigration（log_id=0 清理）

- **触发条件**：存在 `logs.id=0` 或 `log_items.log_id=0` 的记录；否则直接返回（无操作）。
- **行为**：
  1. 删除 `log_items.log_id=0` 的孤儿条目；
  2. 删除 `logs.id=0` 的伪日志；
  3. **重算 size**：`UPDATE logs SET size = (该日志下 removed IS NULL 的条目数)`（仅 id>0）。
- **幂等**：是。
- **失败**：返回错误 → 中断升级。
- **本轮加固**：重算前增加 `HasColumn(logs, size)` 判断——若 size 列不存在（V150 失误遗留），则**跳过重算**（不报错），改由 010 负责“建列+重算”。此前缺少此判断时，一旦“无 size 列 + 存在 log_id=0 数据”同时出现，本迁移会因 `UPDATE … SET size …` 列缺失而报错、阻塞后续迁移。

### 009 — V160LogRawMsgIDIndexMigration（日志复合索引）

- **触发条件**：存在 log_items 表。
- **行为**：创建复合索引 `idx_log_delete_by_id (group_id, raw_msg_id, id)`，用于日志消息回查/撤回删除。MySQL 使用长度 20 的前缀索引（与现有前缀索引风格一致）。
- **幂等**：是（`HasIndex` 判断，已存在则跳过）。
- **失败**：返回错误 → 中断升级。

### 010 — V160LogSizeRepairMigration（logs.size 兜底修复）★ 本轮新增

- **触发条件**：存在 `logs` 表；否则跳过。
- **行为**：
  1. **检测 size 列**：`HasColumn(logs, size)` 为假时，用 GORM `AddColumn` 补建（兼容 SQLite/MySQL/PG 的标识符引用）。
  2. **全量重算**：`UPDATE logs SET size = (SELECT COUNT(1) FROM log_items WHERE log_items.log_id = logs.id AND log_items.removed IS NULL)`。前置条件：`log_items` 表必须存在（与 `logs` 由 V120 一同创建）。若 `logs` 存在而 `log_items` 缺失，视为数据库状态异常，返回错误中断升级。
- **幂等**：是（列存在则只重算；重算是覆盖式，重复执行结果一致）。
- **失败**：返回错误 → 中断升级。
- **设计说明**：用裸 `db.Exec` 而非 `db.Model().Update()`，以绕开 GORM “无 WHERE 的批量更新”保护——这里确实需要更新全部行；相关子查询与 008 重算口径完全一致，三种数据库均支持。

---

## size 语义（请重点审阅）

`logs.size` 表示“该日志的条目数”，但历史上存在两种口径：

| 来源 | 口径 |
|------|------|
| V150 `calculateLogSize`（006 内） | 统计**全部** log_items（不区分 removed） |
| 运行期 `LogAppend`(+1) / `LogMarkDelete`(−1) | 实质追踪**未删除**条目（撤回是硬删除 + size−1） |
| 008 / 010 的重算 | 统计 `removed IS NULL` 的**可见**条目 |

**本次统一到“仅统计可见条目（removed IS NULL）”**，与运行期不变式、以及最新的 008 重算一致。V150 旧实现的“全量统计”属历史差异，保留在 006 内未改动（改动已应用的迁移有风险），由 010 的全量重算在 V160 阶段把口径纠正过来。

如希望 010 改为“统计全部（含 removed）”，把两条重算 SQL 中的 `AND log_items.removed IS NULL` 去掉即可——请告知偏好。

## 测试覆盖

测试位于 `migrate/v2/*_test.go`（包内测试，共享 `migrate_helper_test.go`），以及 `utils/upgrader/store/gorm_store_test.go`：

- `gorm_store_test.go`：GormStore 的"建表→SaveRecord→IsApplied→LoadRecords 往返"、失败记录语义、幂等 Save、空 Logs。
- `v120_test.go`：V120 自检守卫（`attrs` 存在 → 跳过迁移 + 重命名 `data.bdb`）。
- `v160_log_size_test.go`：010 的“补建缺失列+重算”、“已有列重算”、“无 logs 表无操作”。
- `v160_logid0_test.go`：008 的“清理+重算”、“size 列缺失时不报错”、“无数据无操作”。
- `full_flow_test.go`：用 `testdata/full_setup_logs.sql` + `testdata/full_setup_data.sql` 造假库，跑完整 V120→V010 链路并断言结果，再跑第二次验证幂等。

测试仅覆盖 SQLite（本仓库无 MySQL/PG 的 CI 实例）；迁移代码本身对三种数据库都做了分支处理。

> 测试数据约定：`testdata/problem.sql` 是真实“问题库”结构基线（保留不动）；`testdata/full_setup_*.sql` 是本轮新增的、含旧 attrs 表与坏数据的完整流程 fixture。
