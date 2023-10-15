package model

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"time"
)

func attrGetAllBase(db *sqlx.DB, bucket string, key string) []byte {
	var buf []byte

	query := `SELECT updated_at, data FROM ` + bucket + ` WHERE id=:id`
	rows, err := db.NamedQuery(query, map[string]interface{}{"id": key})
	if err != nil {
		fmt.Println("Failed to execute query:", err)
		return buf
	}

	defer rows.Close()

	for rows.Next() {
		var updatedAt int64
		var data []byte

		err := rows.Scan(&updatedAt, &data)
		if err != nil {
			fmt.Println("Failed to scan row:", err)
			break
		}

		buf = data
	}

	return buf
}

func attrSave(db *sqlx.DB, bucket string, key string, data []byte) {
	if db == nil {
		return
	}

	nowTimestamp := time.Now().Unix()

	query := `REPLACE INTO ` + bucket + ` (id, updated_at, data) VALUES (:id, :updated_at, :data)`
	_, err := db.NamedExec(query, map[string]interface{}{
		"id":         key,
		"updated_at": nowTimestamp,
		"data":       data,
	})
	if err != nil {
		fmt.Println("Failed to execute query:", err)
	}
}

func AttrGroupUserGetAllBase(db *sqlx.DB, id string) []byte {
	return attrGetAllBase(db, "attrs_group_user", id)
}

func AttrGroupUserGetAll(db *sqlx.DB, groupId string, userId string) []byte {
	//return attrGetAllBase(db, "attrs_group_user", fmt.Sprintf("%s-%s", groupId, userId))
	return AttrGroupUserGetAllBase(db, fmt.Sprintf("%s-%s", groupId, userId))
}

func AttrGroupUserSave(db *sqlx.DB, groupId string, userId string, data []byte) {
	attrSave(db, "attrs_group_user", fmt.Sprintf("%s-%s", groupId, userId), data)
}

func AttrGroupGetAll(db *sqlx.DB, groupId string) []byte {
	return attrGetAllBase(db, "attrs_group", groupId)
}

func AttrGroupSave(db *sqlx.DB, groupId string, data []byte) {
	attrSave(db, "attrs_group", groupId, data)
}

func AttrUserGetAll(db *sqlx.DB, userId string) []byte {
	return attrGetAllBase(db, "attrs_user", userId)
}

func AttrUserSave(db *sqlx.DB, userId string, data []byte) {
	attrSave(db, "attrs_user", userId, data)
}
