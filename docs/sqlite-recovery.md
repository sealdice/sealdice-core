# SQLite 数据库损坏恢复说明

本文按当前代码中的实际库名和迁移逻辑整理，适用于海豹默认 SQLite 数据目录 `data/default/` 下的这三个文件：

- `data.db`
- `data-logs.db`
- `data-censor.db`

> 请注意，升级之前务必先修复数据库再进行升级，海豹不会检查你的数据库的完整性。

注意：

1. 全程保持海豹关闭，不要在海豹运行时复制或替换数据库。
2. 不要直接在 `data/default/` 里对原文件做 `.recover`，先复制出来再操作。
3. 复制数据库时，要把对应的 `-wal`、`-shm` 文件一起复制出来；没有就跳过。
4. 三种数据库请分别处理，不要混用命令，也不要共用中间文件名。
5. V146 / V150 的结构差异只影响 `data.db`；`data-logs.db` 和 `data-censor.db` 这里不需要按 V146 / V150 分两套方案。

## 先复制出待修复文件

把损坏的数据库从海豹的 `data/default/` 目录复制出来，放到和 `sqlite3.exe` 同一个目录。复制时要带上对应的 WAL、SHM 文件。

例如要修 `data.db`，除了 `data.db` 本体，也要一并处理：

```text
data.db
data.db-wal
data.db-shm
```

`data-logs.db` 和 `data-censor.db` 也一样，分别对应：

```text
data-logs.db
data-logs.db-wal
data-logs.db-shm
```

```text
data-censor.db
data-censor.db-wal
data-censor.db-shm
```

## 1. 修复 `data.db`

在 `sqlite3.exe` 所在目录打开命令行，执行：

### 导出 `data.db`

```shell
sqlite3.exe data.db
.output recover-data.sql
.recover
.exit
```

### 恢复到新的 `data.db`

```shell
sqlite3.exe fixed-data.db
.read recover-data.sql
.exit
```

### 按版本检查并清理 `fixed-data.db`

`data.db` 的结构会随版本变化。先看表名，再决定用哪组清理语句。

#### 旧结构：V146 / v1.4.6 一类

如果 `.tables` 里还能看到 `attrs_user`、`attrs_group`、`attrs_group_user`，说明这是旧结构。

先抽查：

```sql
.tables
SELECT id FROM attrs_user LIMIT 20;
SELECT id FROM attrs_group LIMIT 20;
SELECT id FROM attrs_group_user LIMIT 20;
SELECT id FROM group_info LIMIT 20;
SELECT id FROM group_player_info LIMIT 20;
SELECT id FROM ban_info LIMIT 20;
```

确认结果基本正常后，再清理恢复过程中产生的坏行。下面这组语句合并了当前代码里仍在处理的几类脏数据：

```sql
delete from attrs_group where id is null;
delete from attrs_user where id is null;
delete from attrs_group_user where id is null;
delete from group_info where id is null;
delete from ban_info where id is null;
delete from group_player_info where id is null;

delete from group_info
where not (
    (created_at is null or cast(created_at as integer) > 0)
    and (updated_at is null or cast(updated_at as integer) > 0)
    and data is not null
);

delete from ban_info
where data is null or data = '' or length(data) = 0;

.exit
```

#### 新结构：V150 / v1.5.0 及以上

如果 `.tables` 里已经有统一的 `attrs` 表，就不要再使用旧的 `attrs_user` / `attrs_group` / `attrs_group_user` 清理语句。

先抽查：

```sql
.tables
SELECT id FROM attrs LIMIT 20;
SELECT id FROM group_info LIMIT 20;
SELECT id FROM ban_info LIMIT 20;
SELECT id FROM group_player_info LIMIT 20;
```

确认结果基本正常后，执行：

```sql
delete from attrs where id is null;
delete from group_info where id is null;
delete from ban_info where id is null;
delete from group_player_info where id is null;

delete from attrs
where data is null or data = '' or length(data) = 0;

delete from group_info
where not (
    (created_at is null or cast(created_at as integer) > 0)
    and (updated_at is null or cast(updated_at as integer) > 0)
    and data is not null
);

delete from ban_info
where data is null or data = '' or length(data) = 0;

.exit
```

处理完后，`fixed-data.db` 就是修复后的 `data.db`。

## 2. 修复 `data-logs.db`

`data-logs.db` 是单独的日志库，主要是 `logs` 和 `log_items`，不要套用 `data.db` 的清理 SQL。

V146 对日志库没有单独的结构变化，这里的导出、恢复和清理流程不需要再区分 V146 / V150。

### 导出 `data-logs.db`

```shell
sqlite3.exe data-logs.db
.output recover-data-logs.sql
.recover
.exit
```

### 恢复到新的 `data-logs.db`

```shell
sqlite3.exe fixed-data-logs.db
.read recover-data-logs.sql
.exit
```

### 检查并按日志库结构清理 `fixed-data-logs.db`

先抽查：

```sql
.tables
PRAGMA table_info(logs);
PRAGMA table_info(log_items);
SELECT id FROM logs LIMIT 20;
SELECT log_id FROM log_items LIMIT 20;
```

如果发现恢复后出现了 `id = 0` 或 `log_id = 0` 的坏行，可以按当前代码里的日志修复逻辑清理：

```sql
delete from log_items where log_id = 0;
delete from logs where id = 0;
```

如果 `logs` 表里已经有 `size` 列，再执行一次重算：

```sql
update logs
set size = (
    select count(1)
    from log_items
    where log_items.log_id = logs.id
      and log_items.removed is null
);

.exit
```

如果 `PRAGMA table_info(logs);` 看不到 `size` 列，就先不要照着上面的 `update logs set size = ...` 执行；先保留 `fixed-data-logs.db`，再结合当前版本是否需要手工补列处理。

处理完后，`fixed-data-logs.db` 就是修复后的 `data-logs.db`。

## 3. 修复 `data-censor.db`

当前代码里 `data-censor.db` 主要使用 `censor_log` 表，不要套用 `data.db` 或 `data-logs.db` 的清理语句。

V146 对敏感词日志库也没有单独的结构变化，这里的流程同样不需要按 V146 / V150 区分。

### 导出 `data-censor.db`

```shell
sqlite3.exe data-censor.db
.output recover-data-censor.sql
.recover
.exit
```

### 恢复到新的 `data-censor.db`

```shell
sqlite3.exe fixed-data-censor.db
.read recover-data-censor.sql
.exit
```

### 检查 `fixed-data-censor.db`

先抽查：

```sql
.tables
PRAGMA table_info(censor_log);
SELECT id FROM censor_log LIMIT 20;
SELECT user_id, group_id, highest_level FROM censor_log LIMIT 20;
.exit
```

目前代码里没有像 `data.db`、`data-logs.db` 那样对 `data-censor.db` 定义一组通用的自动清理 SQL，所以这里不要机械套删。先抽查，如果确实能看出恢复出了明显垃圾行，再按实际内容有针对性处理。

处理完后，`fixed-data-censor.db` 就是修复后的 `data-censor.db`。

## 4. 复制回去前再做一次完整性检查

建议先检查修复结果：

```shell
sqlite3.exe fixed-data.db "PRAGMA integrity_check;"
sqlite3.exe fixed-data-logs.db "PRAGMA integrity_check;"
sqlite3.exe fixed-data-censor.db "PRAGMA integrity_check;"
```

只修其中一个库时，只检查对应那个 `fixed-*.db` 即可。

## 5. 复制回 `data/default/`

保持海豹关闭，把原路径中待替换数据库对应的旧 `-wal`、`-shm` 文件删掉，再把修好的库复制回去并改回原文件名：

- `fixed-data.db` 改回 `data.db`
- `fixed-data-logs.db` 改回 `data-logs.db`
- `fixed-data-censor.db` 改回 `data-censor.db`

替换完成后，再启动海豹。

## 一键修复脚本

下面的脚本只负责：

1. 从 `data/default/` 复制目标库及其 `-wal` / `-shm`
2. 生成唯一的 `.recover` 导出文件
3. 恢复出唯一的修复结果库
4. 自动执行当前已知的内容清理 SQL
5. 跑一次 `PRAGMA integrity_check;`

脚本支持直接修复数据内容，但它不会自动把修复结果覆盖回海豹目录。你仍然应该先检查生成的 `fixed-*.db`，确认无误后再手工替换。

其中：

- `data.db` 会自动识别旧结构 `attrs_user/attrs_group/attrs_group_user` 和新结构 `attrs`，并选择对应的清理 SQL。
- `data-logs.db` 会自动清理 `id = 0` / `log_id = 0` 的坏行；如果存在 `size` 列，还会自动重算 `logs.size`。
- `data-censor.db` 当前没有通用的自动内容清理规则，所以脚本只做恢复和完整性检查。

### Bash

```bash
#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '%s\n' "$1"
}

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 /path/to/data/default data.db|data-logs.db|data-censor.db"
  exit 1
fi

SRC_DIR="$1"
DB_NAME="$2"

log "[STEP 1/7] Validating arguments and tool path"

case "$DB_NAME" in
  data.db)
    SQL_OUT="recover-data.sql"
    FIXED_DB="fixed-data.db"
    ;;
  data-logs.db)
    SQL_OUT="recover-data-logs.sql"
    FIXED_DB="fixed-data-logs.db"
    ;;
  data-censor.db)
    SQL_OUT="recover-data-censor.sql"
    FIXED_DB="fixed-data-censor.db"
    ;;
  *)
    echo "Unsupported database: $DB_NAME"
    exit 1
    ;;
esac

BASE_DIR="$(pwd)"
if [ -f "$BASE_DIR/sqlite3.exe" ]; then
  SQLITE="$BASE_DIR/sqlite3.exe"
else
  SQLITE="$(command -v sqlite3)"
fi

if [ ! -f "$SRC_DIR/$DB_NAME" ]; then
  echo "Source database not found: $SRC_DIR/$DB_NAME"
  exit 1
fi

log "[STEP 2/7] Preparing workspace"
WORK_DIR="$BASE_DIR/recover-${DB_NAME%.db}"
mkdir -p "$WORK_DIR"

log "[STEP 3/7] Copying database and sidecar files"
cp "$SRC_DIR/$DB_NAME" "$WORK_DIR/$DB_NAME"
for sidecar in "-wal" "-shm"; do
  if [ -f "$SRC_DIR/$DB_NAME$sidecar" ]; then
    cp "$SRC_DIR/$DB_NAME$sidecar" "$WORK_DIR/$DB_NAME$sidecar"
    log "[INFO] Copied $DB_NAME$sidecar"
  else
    log "[INFO] Sidecar not found, skipped: $DB_NAME$sidecar"
  fi
done

cd "$WORK_DIR"

log "[STEP 4/7] Exporting recover SQL"
"$SQLITE" "$DB_NAME" <<EOF
.output $SQL_OUT
.recover
.exit
EOF

log "[STEP 5/7] Rebuilding recovered database"
"$SQLITE" "$FIXED_DB" <<EOF
.read $SQL_OUT
.exit
EOF

cleanup_data_db() {
  local schema
  schema="$("$SQLITE" "$FIXED_DB" "SELECT CASE WHEN EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='attrs') THEN 'v150+' WHEN EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='attrs_user') THEN 'v146' ELSE 'unknown' END;")"
  log "[INFO] data.db schema detected: $schema"

  case "$schema" in
    v146)
      "$SQLITE" "$FIXED_DB" <<'EOF'
delete from attrs_group where id is null;
delete from attrs_user where id is null;
delete from attrs_group_user where id is null;
delete from group_info where id is null;
delete from ban_info where id is null;
delete from group_player_info where id is null;
delete from group_info
where not (
    (created_at is null or cast(created_at as integer) > 0)
    and (updated_at is null or cast(updated_at as integer) > 0)
    and data is not null
);
delete from ban_info
where data is null or data = '' or length(data) = 0;
EOF
      log "[INFO] Applied V146 data.db cleanup SQL"
      ;;
    v150+)
      "$SQLITE" "$FIXED_DB" <<'EOF'
delete from attrs where id is null;
delete from group_info where id is null;
delete from ban_info where id is null;
delete from group_player_info where id is null;
delete from attrs
where data is null or data = '' or length(data) = 0;
delete from group_info
where not (
    (created_at is null or cast(created_at as integer) > 0)
    and (updated_at is null or cast(updated_at as integer) > 0)
    and data is not null
);
delete from ban_info
where data is null or data = '' or length(data) = 0;
EOF
      log "[INFO] Applied V150+ data.db cleanup SQL"
      ;;
    *)
      log "[WARN] Unknown data.db schema. Automatic content cleanup skipped."
      ;;
  esac
}

cleanup_logs_db() {
  local has_size

  "$SQLITE" "$FIXED_DB" <<'EOF'
delete from log_items where log_id = 0;
delete from logs where id = 0;
EOF
  log "[INFO] Applied log_id=0 / id=0 cleanup SQL"

  has_size="$("$SQLITE" "$FIXED_DB" "SELECT CASE WHEN EXISTS(SELECT 1 FROM pragma_table_info('logs') WHERE name = 'size') THEN 1 ELSE 0 END;")"
  if [ "$has_size" = "1" ]; then
    "$SQLITE" "$FIXED_DB" <<'EOF'
update logs
set size = (
    select count(1)
    from log_items
    where log_items.log_id = logs.id
      and log_items.removed is null
);
EOF
    log "[INFO] Recalculated logs.size"
  else
    log "[WARN] logs.size column not found. Skipped size recalculation."
  fi
}

cleanup_censor_db() {
  log "[INFO] No generic content cleanup is defined for data-censor.db"
}

log "[STEP 6/7] Applying automatic content cleanup"
case "$DB_NAME" in
  data.db)
    cleanup_data_db
    ;;
  data-logs.db)
    cleanup_logs_db
    ;;
  data-censor.db)
    cleanup_censor_db
    ;;
esac

log "[STEP 7/7] Running integrity check"
"$SQLITE" "$FIXED_DB" "PRAGMA integrity_check;"

log "[DONE] Generated files:"
log "  $WORK_DIR/$SQL_OUT"
log "  $WORK_DIR/$FIXED_DB"
```

### Batch

```bat
@echo off
setlocal enabledelayedexpansion

if "%~2"=="" (
  echo Usage: %~nx0 ^<data-default-dir^> ^<data.db^|data-logs.db^|data-censor.db^>
  exit /b 1
)

set "SRC_DIR=%~1"
set "DB_NAME=%~2"

echo [STEP 1/7] Validating arguments and tool path

if /I "%DB_NAME%"=="data.db" (
  set "SQL_OUT=recover-data.sql"
  set "FIXED_DB=fixed-data.db"
) else if /I "%DB_NAME%"=="data-logs.db" (
  set "SQL_OUT=recover-data-logs.sql"
  set "FIXED_DB=fixed-data-logs.db"
) else if /I "%DB_NAME%"=="data-censor.db" (
  set "SQL_OUT=recover-data-censor.sql"
  set "FIXED_DB=fixed-data-censor.db"
) else (
  echo Unsupported database: %DB_NAME%
  exit /b 1
)

set "SQLITE=%~dp0sqlite3.exe"
if not exist "%SQLITE%" (
  echo sqlite3.exe was not found in the current directory
  exit /b 1
)

if not exist "%SRC_DIR%\%DB_NAME%" (
  echo Source database not found: %SRC_DIR%\%DB_NAME%
  exit /b 1
)

echo [STEP 2/7] Preparing workspace
set "WORK_DIR=%~dp0recover-%DB_NAME:.db=%"
if not exist "%WORK_DIR%" mkdir "%WORK_DIR%"

echo [STEP 3/7] Copying database and sidecar files
copy /y "%SRC_DIR%\%DB_NAME%" "%WORK_DIR%\%DB_NAME%" >nul || exit /b 1
if exist "%SRC_DIR%\%DB_NAME%-wal" (
  copy /y "%SRC_DIR%\%DB_NAME%-wal" "%WORK_DIR%\%DB_NAME%-wal" >nul
  echo [INFO] Copied %DB_NAME%-wal
) else (
  echo [INFO] Sidecar not found, skipped: %DB_NAME%-wal
)
if exist "%SRC_DIR%\%DB_NAME%-shm" (
  copy /y "%SRC_DIR%\%DB_NAME%-shm" "%WORK_DIR%\%DB_NAME%-shm" >nul
  echo [INFO] Copied %DB_NAME%-shm
) else (
  echo [INFO] Sidecar not found, skipped: %DB_NAME%-shm
)

pushd "%WORK_DIR%"

echo [STEP 4/7] Exporting recover SQL
(
  echo .output %SQL_OUT%
  echo .recover
  echo .exit
) > recover-commands.txt
"%SQLITE%" "%DB_NAME%" < recover-commands.txt

echo [STEP 5/7] Rebuilding recovered database
(
  echo .read %SQL_OUT%
  echo .exit
) > import-commands.txt
"%SQLITE%" "%FIXED_DB%" < import-commands.txt

echo [STEP 6/7] Applying automatic content cleanup
if /I "%DB_NAME%"=="data.db" (
  for /f "usebackq delims=" %%i in (`"%SQLITE%" "%FIXED_DB%" "SELECT CASE WHEN EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='attrs') THEN 'v150+' WHEN EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='attrs_user') THEN 'v146' ELSE 'unknown' END;"`) do set "DATA_SCHEMA=%%i"
  echo [INFO] data.db schema detected: !DATA_SCHEMA!

  if /I "!DATA_SCHEMA!"=="v146" (
    (
      echo delete from attrs_group where id is null;
      echo delete from attrs_user where id is null;
      echo delete from attrs_group_user where id is null;
      echo delete from group_info where id is null;
      echo delete from ban_info where id is null;
      echo delete from group_player_info where id is null;
      echo delete from group_info
      echo where not ^(
      echo     ^(created_at is null or cast^(created_at as integer^) ^> 0^)
      echo     and ^(updated_at is null or cast^(updated_at as integer^) ^> 0^)
      echo     and data is not null
      echo ^);
      echo delete from ban_info
      echo where data is null or data = '' or length^(data^) = 0;
    ) > cleanup.sql
    "%SQLITE%" "%FIXED_DB%" < cleanup.sql
    echo [INFO] Applied V146 data.db cleanup SQL
  ) else if /I "!DATA_SCHEMA!"=="v150+" (
    (
      echo delete from attrs where id is null;
      echo delete from group_info where id is null;
      echo delete from ban_info where id is null;
      echo delete from group_player_info where id is null;
      echo delete from attrs
      echo where data is null or data = '' or length^(data^) = 0;
      echo delete from group_info
      echo where not ^(
      echo     ^(created_at is null or cast^(created_at as integer^) ^> 0^)
      echo     and ^(updated_at is null or cast^(updated_at as integer^) ^> 0^)
      echo     and data is not null
      echo ^);
      echo delete from ban_info
      echo where data is null or data = '' or length^(data^) = 0;
    ) > cleanup.sql
    "%SQLITE%" "%FIXED_DB%" < cleanup.sql
    echo [INFO] Applied V150+ data.db cleanup SQL
  ) else (
    echo [WARN] Unknown data.db schema. Automatic content cleanup skipped.
  )
) else if /I "%DB_NAME%"=="data-logs.db" (
  (
    echo delete from log_items where log_id = 0;
    echo delete from logs where id = 0;
  ) > cleanup.sql
  "%SQLITE%" "%FIXED_DB%" < cleanup.sql
  echo [INFO] Applied log_id=0 / id=0 cleanup SQL

  for /f "usebackq delims=" %%i in (`"%SQLITE%" "%FIXED_DB%" "SELECT CASE WHEN EXISTS(SELECT 1 FROM pragma_table_info('logs') WHERE name = 'size') THEN 1 ELSE 0 END;"`) do set "HAS_SIZE=%%i"
  if "!HAS_SIZE!"=="1" (
    (
      echo update logs
      echo set size = ^(
      echo     select count^(1^)
      echo     from log_items
      echo     where log_items.log_id = logs.id
      echo       and log_items.removed is null
      echo ^);
    ) > recalc-size.sql
    "%SQLITE%" "%FIXED_DB%" < recalc-size.sql
    echo [INFO] Recalculated logs.size
  ) else (
    echo [WARN] logs.size column not found. Skipped size recalculation.
  )
) else if /I "%DB_NAME%"=="data-censor.db" (
  echo [INFO] No generic content cleanup is defined for data-censor.db
)

echo [STEP 7/7] Running integrity check
"%SQLITE%" "%FIXED_DB%" "PRAGMA integrity_check;"

echo [DONE] Generated files:
echo   %WORK_DIR%\%SQL_OUT%
echo   %WORK_DIR%\%FIXED_DB%

popd
endlocal
```

## 说明依据

这份说明按当前仓库代码整理，重点参考了：

- SQLite 三库文件名与完整性检查逻辑
- `data.db` 的 V150 / V151 清理逻辑
- `data-logs.db` 的 V160 清理与 `size` 重算逻辑
- `data-censor.db` 当前仅初始化 `censor_log` 表的事实
