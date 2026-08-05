/*
 Navicat Premium Data Transfer

 Source Server         : data-logs (3)
 Source Server Type    : SQLite
 Source Server Version : 3035005 (3.35.5)
 Source Schema         : main

 Target Server Type    : SQLite
 Target Server Version : 3035005 (3.35.5)
 File Encoding         : 65001

 Date: 17/07/2026 18:46:45
*/

PRAGMA foreign_keys = false;

-- ----------------------------
-- Table structure for log_items
-- ----------------------------
DROP TABLE IF EXISTS "log_items";
CREATE TABLE "log_items" (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "log_id" INTEGER,
  "group_id" TEXT,
  "nickname" TEXT,
  "im_userid" TEXT,
  "time" INTEGER,
  "message" TEXT,
  "is_dice" INTEGER,
  "command_id" INTEGER,
  "command_info" TEXT,
  "raw_msg_id" TEXT,
  "user_uniform_id" TEXT,
  "removed" INTEGER,
  "parent_id" INTEGER
);

-- ----------------------------
-- Table structure for logs
-- ----------------------------
DROP TABLE IF EXISTS "logs";
CREATE TABLE "logs" (
  "id" INTEGER PRIMARY KEY AUTOINCREMENT,
  "name" TEXT,
  "group_id" TEXT,
  "extra" TEXT,
  "created_at" INTEGER,
  "updated_at" INTEGER,
  "upload_url" TEXT,
  "upload_time" INTEGER
);

-- ----------------------------
-- Table structure for lost_and_found
-- ----------------------------
DROP TABLE IF EXISTS "lost_and_found";
CREATE TABLE "lost_and_found" (
  "rootpgno" INTEGER,
  "pgno" INTEGER,
  "nfield" INTEGER,
  "id" INTEGER,
  "c0",
  "c1"
);

-- ----------------------------
-- Table structure for sqlite_sequence
-- ----------------------------
DROP TABLE IF EXISTS "sqlite_sequence";
CREATE TABLE "sqlite_sequence" (
  "name",
  "seq"
);

-- ----------------------------
-- Auto increment value for log_items
-- ----------------------------
UPDATE "sqlite_sequence" SET seq = 4176811 WHERE name = 'log_items';

-- ----------------------------
-- Indexes structure for table log_items
-- ----------------------------
CREATE INDEX "idx_log_items_group_id"
ON "log_items" (
  "log_id" ASC
);
CREATE INDEX "idx_log_items_log_id"
ON "log_items" (
  "log_id" ASC
);

-- ----------------------------
-- Auto increment value for logs
-- ----------------------------
UPDATE "sqlite_sequence" SET seq = 5301 WHERE name = 'logs';

-- ----------------------------
-- Indexes structure for table logs
-- ----------------------------
CREATE UNIQUE INDEX "idx_log_group_id_name"
ON "logs" (
  "group_id" ASC,
  "name" ASC
);
CREATE INDEX "idx_logs_group"
ON "logs" (
  "group_id" ASC
);
CREATE INDEX "idx_logs_update_at"
ON "logs" (
  "updated_at" ASC
);

PRAGMA foreign_keys = true;
