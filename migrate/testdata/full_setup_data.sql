-- full_setup_data.sql
-- 用于“完整升级流程”测试：灌入 data.db。
-- 特征：
--   1. 存在旧版 attrs_user / attrs_group / attrs_group_user 表，供 V150 合并迁移到统一 attrs 表；
--   2. attrs_user 中含一行 data 为 NULL 的坏数据（V150 解析阶段会被跳过）；
--   3. ban_info 中含一行 data 为 NULL 的坏数据，供 V151 清理迁移删除。
--
-- 注：data 字段为旧版 VMValue 的 JSON 序列化形式，
--   形如 {"key":{"typeId":0,"value":42,"expiredTime":0}}。

DROP TABLE IF EXISTS attrs_user;
CREATE TABLE attrs_user (
  id TEXT PRIMARY KEY,
  updated_at INTEGER,
  data BLOB
);

DROP TABLE IF EXISTS attrs_group;
CREATE TABLE attrs_group (
  id TEXT PRIMARY KEY,
  updated_at INTEGER,
  data BLOB
);

DROP TABLE IF EXISTS attrs_group_user;
CREATE TABLE attrs_group_user (
  id TEXT PRIMARY KEY,
  updated_at INTEGER,
  data BLOB
);

DROP TABLE IF EXISTS ban_info;
CREATE TABLE ban_info (
  id TEXT PRIMARY KEY,
  data BLOB
);

-- 用户 QQ:100：一个普通整型属性
INSERT INTO attrs_user (id, updated_at, data) VALUES
  ('QQ:100', 1700000000, '{"hp":{"typeId":0,"value":42,"expiredTime":0}}'),
  ('QQ:BAD', 1700000000, NULL);

-- 群 QQ-Group:1
INSERT INTO attrs_group (id, updated_at, data) VALUES
  ('QQ-Group:1', 1700000000, '{"count":{"typeId":0,"value":7,"expiredTime":0}}');

-- 群内用户：id 形如 {GroupID}-{UserID}
INSERT INTO attrs_group_user (id, updated_at, data) VALUES
  ('QQ-Group:1-QQ:100', 1700000000, '{"gp":{"typeId":0,"value":5,"expiredTime":0}}');

-- ban_info：一条正常 + 一条坏（data 为 NULL，V151 会删）
INSERT INTO ban_info (id, data) VALUES
  ('ban-1', '{"a":1}'),
  ('ban-bad', NULL);
