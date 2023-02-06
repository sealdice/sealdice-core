package model

import (
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func BanItemDel(db *sqlitex.Pool, id string) error {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()
	err := sqlitex.ExecuteTransient(conn, `delete from ban_info where id=$id`, &sqlitex.ExecOptions{
		Named: map[string]interface{}{
			"$id": id,
		},
	})

	return err
}

func BanItemSave(db *sqlitex.Pool, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	err := sqlitex.ExecuteTransient(conn, `replace into ban_info (id, updated_at, ban_updated_at, data) values ($id, $updated_at, $ban_updated_at, $data)`, &sqlitex.ExecOptions{
		// $id, $updated_at, $ban_updated_at, $data
		Named: map[string]interface{}{
			"$id":             id,
			"$updated_at":     updatedAt,
			"$ban_updated_at": banUpdatedAt,
			"$data":           data,
		},
	})

	return err
}

func BanItemList(db *sqlitex.Pool, callback func(id string, banUpdatedAt int64, data []byte)) error {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	err := sqlitex.ExecuteTransient(conn, `select id, ban_updated_at, data from ban_info order by ban_updated_at desc`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			callback(stmt.ColumnText(0), stmt.ColumnInt64(1), []byte(stmt.ColumnText(2)))
			return nil
		},
	})

	return err
}
