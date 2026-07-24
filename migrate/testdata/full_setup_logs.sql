-- full_setup_logs.sql
-- 用于“完整升级流程”测试：灌入 data-logs.db。
-- 特征：
--   1. logs 表为旧版结构（无 size 列），模拟 V150 历史升级失误的遗留；
--   2. 含 id=0 的伪日志与 log_id=0 的孤儿条目，供 V160 清理迁移处理；
--   3. 含一条 removed=1 的条目，验证 size 只统计可见条目。

DROP TABLE IF EXISTS logs;
CREATE TABLE logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT,
  group_id TEXT,
  extra TEXT,
  created_at INTEGER,
  updated_at INTEGER,
  upload_url TEXT,
  upload_time INTEGER
);

DROP TABLE IF EXISTS log_items;
CREATE TABLE log_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  log_id INTEGER,
  group_id TEXT,
  nickname TEXT,
  im_userid TEXT,
  time INTEGER,
  message TEXT,
  is_dice INTEGER,
  command_id INTEGER,
  command_info TEXT,
  raw_msg_id TEXT,
  user_uniform_id TEXT,
  removed INTEGER,
  parent_id INTEGER
);

-- 日志 1：3 条可见 + 1 条 removed（size 期望为 3）
-- 日志 2：1 条可见（size 期望为 1）
-- 日志 0：伪日志，将被 V160 清理
INSERT INTO logs (id, name, group_id, created_at, updated_at) VALUES
  (1, 'log-one', 'QQ-Group:1', 1700000000, 1700000100),
  (2, 'log-two', 'QQ-Group:1', 1700000200, 1700000300),
  (0, NULL, NULL, 0, 0);

INSERT INTO log_items (log_id, group_id, nickname, message, time, is_dice, removed) VALUES
  (1, 'QQ-Group:1', 'alice', 'hello', 1700000001, 0, NULL),
  (1, 'QQ-Group:1', 'bob', 'world', 1700000002, 0, NULL),
  (1, 'QQ-Group:1', 'alice', 'removed-line', 1700000003, 1, 1),
  (1, 'QQ-Group:1', 'bob', 'again', 1700000004, 0, NULL),
  (2, 'QQ-Group:1', 'carol', 'hi', 1700000201, 0, NULL),
  (0, 'QQ-Group:1', 'orphan', 'ghost', 0, 0, NULL);
