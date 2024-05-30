package migrate

import (
	"github.com/jmoiron/sqlx"

	"sealdice-core/dice"
	"sealdice-core/dice/model"
)

func attrs_migrate(db *sqlx.DB) (int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_group_user", map[string]any{})
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	countFailed := 0
	var items []*model.AttributesItemModel
	for rows.Next() {
		var id string
		var updatedAt int64
		var data []byte

		err := rows.Scan(
			&id,
			&updatedAt,
			&data,
		)

		if err != nil {
			return count, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			continue
		}

		var cardType string
		if val, ok := mapData["$cardType"]; ok {
			cardType, _ = val.ReadString()
		}

		// ... TODO: 类型转换

		item := &model.AttributesItemModel{
			Id:        id,
			Data:      data,
			SheetType: cardType,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}
		items = append(items, item)
		count += 1
	}

	return count, nil
}
