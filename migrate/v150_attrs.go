package migrate

import (
	"fmt"

	"sealdice-core/dice"
	"sealdice-core/dice/model"

	"github.com/jmoiron/sqlx"

	ds "github.com/sealdice/dicescript"
)

func attrsGroupUserMigrate(db *sqlx.DB) (int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_group_user", map[string]any{})
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	gVarCache := map[string]map[string]*dice.VMValue{}
	getGlobalVarByUid := func(id string) map[string]*dice.VMValue {
		if v, ok := gVarCache[id]; ok {
			return v
		}
		data := model.AttrUserGetAll(db, id)
		mapData := make(map[string]*dice.VMValue)
		_ = dice.JSONValueMapUnmarshal(data, &mapData)
		gVarCache[id] = mapData
		return mapData
	}

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

		// TODO
		// $:cardBindMark":{"typeId":0,"value":1,"expiredTime":0}
		// 归属从id拿
		// 卡片名字: $ch:xxxx  ctx.LoadPlayerGlobalVars
		// 绑定: ctx.ChBindCurGet

		// groupIdPart
		_, userIdPart, ok := dice.UnpackGroupUserId(id)
		if !ok {
			countFailed += 1
			continue
		}

		// 绑定的卡似乎都存在attrs_user里面
		gVars := getGlobalVarByUid(userIdPart)
		fmt.Println(gVars)
		//for k, v := range gVars {
		//	if strings.HasPrefix(k, "$ch:") {
		//		name := k[4:]
		//	}
		//}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			m.Store(k, v.ConvertToV2())
		}

		rawData, err := ds.VMValueNewDict(m).V().ToJSON()
		if err != nil {
			return count, err
		}

		item := &model.AttributesItemModel{
			Id:   id,
			Data: rawData,

			//BindingSheetId: "",

			// 这些是角色卡专用的
			Nickname:  "",
			OwnerId:   "",
			SheetType: cardType,
			IsHidden:  false,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}
		items = append(items, item)
		count += 1
	}

	return count, nil
}
