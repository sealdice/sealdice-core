package model

import (
	"go.etcd.io/bbolt"
	"path/filepath"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func BoltDBInit(path string) *bbolt.DB {
	db, err := bbolt.Open(path, 0644, nil)
	if err != nil {
		panic(err)
	}

	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("attrs_group"))     // 组属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_user"))       // 用户属性
		_, err = tx.CreateBucketIfNotExists([]byte("attrs_group_user")) // 组_用户_属性
		return err
	})

	return db
}

func _SQLiteDBInit(path string, poolSize int) (*sqlitex.Pool, error) {
	flags := sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL
	return sqlitex.Open(path, flags, poolSize)
	//dbpool, err := sqlitex.Open(path, flags, poolSize)
	//return dbpool, err
}

func SQLiteDBInit(dataDir string) (dataDB *sqlitex.Pool, logsDB *sqlitex.Pool, err error) {
	dbDataPath, _ := filepath.Abs(filepath.Join(dataDir, "data.db"))
	dataDB, err = _SQLiteDBInit(dbDataPath, 3)
	if err != nil {
		return
	}

	dbDataLogsPath, _ := filepath.Abs(filepath.Join(dataDir, "data-logs.db"))
	logsDB, err = _SQLiteDBInit(dbDataLogsPath, 10)
	if err != nil {
		return
	}

	// data建表
	conn := dataDB.Get(nil)
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
	}
	for _, i := range texts {
		_ = sqlitex.ExecuteTransient(conn, i, nil)
	}
	dataDB.Put(conn)

	// logs建表
	conn = logsDB.Get(nil)
	texts = []string{
		`
create table if not exists logs
(
    id         INTEGER  primary key autoincrement,
    name       TEXT,
    group_id   TEXT,
    extra      TEXT,
    created_at INTEGER,
    updated_at INTEGER
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
		`
create table if not exists log_items
(
    id              INTEGER primary key autoincrement,
    log_id          INTEGER,
    group_id        TEXT,
    nickname        TEXT,
    im_userid       TEXT,
    time            INTEGER,
    message         INTEGER,
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
	}

	for _, i := range texts {
		_ = sqlitex.ExecuteTransient(conn, i, nil)
	}
	logsDB.Put(conn)

	return
}
