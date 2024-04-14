package model

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
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

func AttrGroupUserGetAll(db *sqlx.DB, groupID string, userID string) []byte {
	return attrGetAllBase(db, "attrs_group_user", fmt.Sprintf("%s-%s", groupID, userID))
}

func AttrGroupUserSave(db *sqlx.DB, groupID string, userID string, data []byte) {
	attrSave(db, "attrs_group_user", fmt.Sprintf("%s-%s", groupID, userID), data)
}

func AttrGroupGetAll(db *sqlx.DB, groupID string) []byte {
	return attrGetAllBase(db, "attrs_group", groupID)
}

func AttrGroupSave(db *sqlx.DB, groupID string, data []byte) {
	attrSave(db, "attrs_group", groupID, data)
}

func AttrUserGetAll(db *sqlx.DB, userID string) []byte {
	return attrGetAllBase(db, "attrs_user", userID)
}

func AttrUserSave(db *sqlx.DB, userID string, data []byte) {
	attrSave(db, "attrs_user", userID, data)
}
