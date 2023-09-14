package migrate

import (
	"fmt"
	"os"
	"regexp"
)

var sqls = []string{
	`ALTER TABLE log_items RENAME TO log_items_old;`,
	`DROP INDEX IF EXISTS idx_log_items_group_id;`,
	`DROP INDEX IF EXISTS idx_log_items_log_id;`,
	`CREATE TABLE IF NOT EXISTS log_items
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
);
CREATE INDEX IF NOT EXISTS idx_log_items_group_id
    ON log_items (log_id);
CREATE INDEX IF NOT EXISTS idx_log_items_log_id
    ON log_items (log_id);
`,
	`INSERT INTO log_items SELECT * FROM log_items_old;`,
	`DROP TABLE log_items_old;`,
}

func LogItemFixDatatype() error {
	if _, err := os.Stat("./data/default/data-logs.db"); err != nil {
		return nil
	}

	db, err := openDB("./data/default/data-logs.db")
	if err != nil {
		return err
	}
	defer db.Close()

	var log_item_sql []string
	err = db.Select(&log_item_sql, "SELECT sql FROM sqlite_master WHERE type='table' AND name='log_items';")
	if err != nil || len(log_item_sql) != 1 {
		return err
	}

	// message字段不是INTEGER类型，说明已经修复过了
	if !regexp.MustCompile(`message\s+INTEGER,`).MatchString(log_item_sql[0]) {
		return nil
	}

	fmt.Println("开始修复log_items表message字段类型")
	fmt.Println("【不要关闭海豹程序！】")

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, sql := range sqls {
		_, err = tx.Exec(sql)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return tx.Rollback()
	}

	_, _ = db.Exec(`vacuum;`)

	fmt.Println("修复log_items表message字段类型成功")
	fmt.Println("您现在可以关闭海豹程序了")
	return nil
}
