package model

import (
	"encoding/binary"
	"fmt"
	"time"
	"zombiezen.com/go/sqlite/sqlitex"
)

func attrGetAllBase(db *sqlitex.Pool, bucket string, key string) []byte {
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	stmt := conn.Prep(`select updated_at, data from ` + bucket + ` where id=$id`)
	defer stmt.Finalize()
	stmt.SetText("$id", key)

	var buf []byte
	for {
		if hasRow, err := stmt.Step(); err != nil {
			break // error
		} else if !hasRow {
			break
		}
		buf = []byte(stmt.ColumnText(1))
	}

	return buf
}

func attrSave(db *sqlitex.Pool, bucket string, key string, data []byte) {
	if db == nil {
		return
	}
	conn := db.Get(nil)
	defer func() { db.Put(conn) }()

	stmt := conn.Prep(`replace into ` + bucket + ` (id, updated_at, data) VALUES ($id, $updated_at, $data)`)
	defer stmt.Finalize()

	// $id, $updated_at, $data
	now := time.Now()
	nowTimestamp := now.Unix()

	stmt.SetText("$id", key)
	stmt.SetInt64("$updated_at", nowTimestamp)
	stmt.SetBytes("$data", data)

	for {
		if hasRow, err := stmt.Step(); err != nil {
			break // error
		} else if !hasRow {
			break
		}
	}
}

func AttrGroupUserGetAll(db *sqlitex.Pool, groupId string, userId string) []byte {
	return attrGetAllBase(db, "attrs_group_user", fmt.Sprintf("%s-%s", groupId, userId))
}

func AttrGroupUserSave(db *sqlitex.Pool, groupId string, userId string, data []byte) {
	attrSave(db, "attrs_group_user", fmt.Sprintf("%s-%s", groupId, userId), data)
}

func AttrGroupGetAll(db *sqlitex.Pool, groupId string) []byte {
	return attrGetAllBase(db, "attrs_group", groupId)
}

func AttrGroupSave(db *sqlitex.Pool, groupId string, data []byte) {
	attrSave(db, "attrs_group", groupId, data)
}

func AttrUserGetAll(db *sqlitex.Pool, userId string) []byte {
	return attrGetAllBase(db, "attrs_user", userId)
}

func AttrUserSave(db *sqlitex.Pool, userId string, data []byte) {
	attrSave(db, "attrs_user", userId, data)
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
