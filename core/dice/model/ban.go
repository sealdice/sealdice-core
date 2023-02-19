package model

import (
	"github.com/jmoiron/sqlx"
)

func BanItemDel(db *sqlx.DB, id string) error {
	_, err := db.Exec("delete from ban_info where id=$1", id)
	return err
}

func BanItemSave(db *sqlx.DB, id string, updatedAt int64, banUpdatedAt int64, data []byte) error {
	_, err := db.NamedExec("replace into ban_info (id, updated_at, ban_updated_at, data) values (:id, :updated_at, :ban_updated_at, :data)",
		map[string]interface{}{
			"id":             id,
			"updated_at":     updatedAt,
			"ban_updated_at": banUpdatedAt,
			"data":           data,
		})
	return err
}

func BanItemList(db *sqlx.DB, callback func(id string, banUpdatedAt int64, data []byte)) error {
	var items []struct {
		ID           string `db:"id"`
		BanUpdatedAt int64  `db:"ban_updated_at"`
		Data         []byte `db:"data"`
	}
	if err := db.Select(&items, "SELECT id, ban_updated_at, data FROM ban_info ORDER BY ban_updated_at DESC"); err != nil {
		return err
	}
	for _, item := range items {
		callback(item.ID, item.BanUpdatedAt, item.Data)
	}
	return nil
}
