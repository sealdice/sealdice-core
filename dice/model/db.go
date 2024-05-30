package model

import (
	"fmt"
	"path/filepath"

	"github.com/jmoiron/sqlx"
)

func DBCheck(dataDir string) {
	checkDB := func(db *sqlx.DB) bool {
		rows, err := db.Query("PRAGMA integrity_check") //nolint:execinquery
		if err != nil {
			return false
		}
		var ok bool
		for rows.Next() {
			var s string
			if errR := rows.Scan(&s); errR != nil {
				ok = false
				break
			}
			fmt.Println(s)
			if s == "ok" {
				ok = true
			}
		}

		if errR := rows.Err(); errR != nil {
			ok = false
		}
		return ok
	}

	var ok1, ok2, ok3 bool
	var dataDB *sqlx.DB
	var logsDB *sqlx.DB
	var censorDB *sqlx.DB
	var err error

	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = _SQLiteDBInit(dbDataPath, false)
	if err != nil {
		fmt.Println("数据库 data.db 无法打开")
	} else {
		ok1 = checkDB(dataDB)
		dataDB.Close()
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, false)
	if err != nil {
		fmt.Println("数据库 data-logs.db 无法打开")
	} else {
		ok2 = checkDB(logsDB)
		logsDB.Close()
	}

	dbDataCensorPath, _ := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	censorDB, err = _SQLiteDBInit(dbDataCensorPath, false)
	if err != nil {
		fmt.Println("数据库 data-censor.db 无法打开")
	} else {
		ok3 = checkDB(censorDB)
		censorDB.Close()
	}

	fmt.Println("数据库检查结果：")
	fmt.Println("data.db:", ok1)
	fmt.Println("data-logs.db:", ok2)
	fmt.Println("data-censor.db:", ok3)
}

func SQLiteDBInit(dataDir string) (dataDB *sqlx.DB, logsDB *sqlx.DB, err error) {
	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = _SQLiteDBInit(dbDataPath, true)
	if err != nil {
		return
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, true)
	if err != nil {
		return
	}

	// data建表
	texts := []string{
		`
create table if not exists group_player_info
(
    id                     INTEGER
        primary key autoincrement,
    group_id               TEXT,
    user_id                TEXT,
    name                   TEXT,
    created_at             INTEGER,
    updated_at             INTEGER,
    last_command_time      INTEGER,
    auto_set_name_template TEXT,
    dice_side_num          TEXT
);`,
		`create index if not exists idx_group_player_info_group_id on group_player_info (group_id);`,
		`create index if not exists idx_group_player_info_user_id on group_player_info (user_id);`,
		`create unique index if not exists idx_group_player_info_group_user on group_player_info (group_id, user_id);`,
		`
create table if not exists group_info
(
    id         TEXT primary key,
    created_at INTEGER,
    updated_at INTEGER,
    data       BLOB
);`,

		`
create table if not exists attrs_group
(
    id         TEXT primary key,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_attrs_group_updated_at on attrs_group (updated_at);`,
		`create table if not exists attrs_group_user
(
    id         TEXT primary key,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_attrs_group_user_updated_at on attrs_group_user (updated_at);`,
		`create table if not exists attrs_user
(
    id         TEXT primary key,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_attrs_user_updated_at on attrs_user (updated_at);`,

		`
create table if not exists ban_info
(
    id         TEXT primary key,
    ban_updated_at INTEGER,
    updated_at INTEGER,
    data       BLOB
);`,
		`create index if not exists idx_ban_info_updated_at on ban_info (updated_at);`,
		`create index if not exists idx_ban_info_ban_updated_at on ban_info (ban_updated_at);`,

		`CREATE TABLE IF NOT EXISTS endpoint_info (
user_id TEXT PRIMARY KEY,
cmd_num INTEGER,
cmd_last_time INTEGER,
online_time INTEGER,
updated_at INTEGER
);`,
	}
	for _, i := range texts {
		_, _ = dataDB.Exec(i)
	}

	// logs建表
	texts = []string{
		`
create table if not exists logs
(
    id         INTEGER  primary key autoincrement,
    name       TEXT,
    group_id   TEXT,
    extra      TEXT,
    created_at INTEGER,
    updated_at INTEGER,
    upload_url TEXT,
    upload_time INTEGER
);`,
		`
create index if not exists idx_logs_group
    on logs (group_id);`,
		`
create index if not exists idx_logs_update_at
    on logs (updated_at);`,
		`
create unique index if not exists idx_log_group_id_name
    on logs (group_id, name);`,
		// 如果log_items有更改，需同步检查migrate/convert_logs.go
		`
create table if not exists log_items
(
    id              INTEGER primary key autoincrement,
    log_id          INTEGER,
    group_id        TEXT,
    nickname        TEXT,
    im_userid       TEXT,
    time            INTEGER,
    message         TEXT,
    is_dice         INTEGER,
    command_id      INTEGER,
    command_info    TEXT,
    raw_msg_id      TEXT,
    user_uniform_id TEXT,
    removed         INTEGER,
    parent_id       INTEGER
);`,
		`
create index if not exists idx_log_items_group_id
    on log_items (log_id);`,
		`
create index if not exists idx_log_items_log_id
    on log_items (log_id);`,

		`alter table logs add upload_url text;`, // 测试版特供
		`alter table logs add upload_time integer;`,
	}

	for _, i := range texts {
		_, _ = logsDB.Exec(i)
	}

	return
}

func SQLiteCensorDBInit(dataDir string) (censorDB *sqlx.DB, err error) {
	path, err := filepath.Abs(filepath.Join(dataDir, "data-censor.db"))
	if err != nil {
		return
	}
	censorDB, err = _SQLiteDBInit(path, true)
	if err != nil {
		return
	}

	texts := []string{`
CREATE TABLE IF NOT EXISTS censor_log
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    msg_type 		TEXT,
    user_id         TEXT,
    group_id 		TEXT,
    content         TEXT,
    sensitive_words TEXT,
    highest_level   INTEGER,
    created_at 		INTEGER,
    clear_mark		BOOLEAN
);
`,
		`
CREATE INDEX IF NOT EXISTS idx_censor_log_user_id
    ON censor_log (user_id);
`,
		`
CREATE INDEX IF NOT EXISTS idx_censor_log_level
    ON censor_log (highest_level);
`,

		`
CREATE TABLE IF NOT EXISTS attrs (
    id TEXT PRIMARY KEY,
    data BYTEA,

	-- 坏，Get这个方法太严格了，所有的字段都要有默认值，不然无法反序列化
	binding_sheet_id TEXT default '',

    nickname TEXT default '',
    owner_id TEXT default '',
    sheet_type TEXT default '',
    is_hidden BOOLEAN default FALSE,

    created_at INTEGER default 0,
    updated_at INTEGER  default 0
);
`,
		`create index if not exists idx_attrs_binding_sheet_id on ban_info (binding_sheet_id);`,
		`create index if not exists idx_attrs_owner_id_id on ban_info (owner_id);`,
	}

	for _, i := range texts {
		_, _ = censorDB.Exec(i)
	}
	return
}
