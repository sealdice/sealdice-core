package model

import "zombiezen.com/go/sqlite/sqlitex"

func Backup(db *sqlitex.Pool, path string) {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	_ = sqlitex.ExecuteTransient(conn, `vacuum into ?`, &sqlitex.ExecOptions{
		Args: []interface{}{path},
	})
}
