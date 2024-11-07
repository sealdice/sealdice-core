package model

import (
	log "sealdice-core/utils/kratos"

	"github.com/jmoiron/sqlx"
)

// 废弃代码先不改

func attrGetAllBase(db *sqlx.DB, bucket string, key string) []byte {
	var buf []byte

	query := `SELECT updated_at, data FROM ` + bucket + ` WHERE id=:id`
	rows, err := db.NamedQuery(query, map[string]interface{}{"id": key})
	if err != nil {
		log.Errorf("Failed to execute query: %v", err)
		return buf
	}

	defer rows.Close()

	for rows.Next() {
		var updatedAt int64
		var data []byte

		err := rows.Scan(&updatedAt, &data)
		if err != nil {
			log.Errorf("Failed to scan row: %v", err)
			break
		}

		buf = data
	}

	return buf
}

func AttrUserGetAll(db *sqlx.DB, userID string) []byte {
	return attrGetAllBase(db, "attrs_user", userID)
}
