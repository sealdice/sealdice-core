package migrate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"

	"sealdice-core/dice"
	"sealdice-core/dice/model"
	"sealdice-core/utils"

	ds "github.com/sealdice/dicescript"
)

func convertToNew(name string, ownerId string, data []byte, updatedAt int64) (*model.AttributesItemModel, error) {
	var err error
	mapData := make(map[string]*dice.VMValue)

	if err = dice.JSONValueMapUnmarshal(data, &mapData); err == nil {
		var cardType string
		if val, ok := mapData["$cardType"]; ok {
			cardType, _ = val.ReadString()
		}

		// 卡片名字: $ch:xxxx  ctx.LoadPlayerGlobalVars
		// 归属: ownerId
		// 当前绑定: ctx.ChBindCur: 卡片角色名: $:ch-bind-data:name

		m2 := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			m2.Store(k, v.ConvertToV2())
		}

		rawData, err := ds.VMValueNewDict(m2).V().ToJSON()
		if err != nil {
			return nil, err
		}

		item := &model.AttributesItemModel{
			Id:        utils.NewID(),
			Data:      rawData,
			AttrsType: model.AttrsTypeCharacter,

			// 这些是角色卡专用的
			Name:      name,
			OwnerId:   ownerId,
			SheetType: cardType,
			IsHidden:  false,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}

		return item, nil
	}

	return nil, err
}

var uVarCache = map[string]map[string]*dice.VMValue{}

func getUserVarsByUid(db *sqlx.DB, id string) map[string]*dice.VMValue {
	if v, ok := uVarCache[id]; ok {
		return v
	}
	data := model.AttrUserGetAll(db, id)
	mapData := make(map[string]*dice.VMValue)
	_ = dice.JSONValueMapUnmarshal(data, &mapData)
	uVarCache[id] = mapData
	return mapData
}

var sheetIdBindByGroupId = map[string]string{}

// 群组个人数据转换
func attrsGroupUserMigrate(db *sqlx.DB) (int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_group_user", map[string]any{})
	if err != nil {
		return 0, 0, err
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
			return count, countFailed, err
		}

		// groupIdPart
		groupIdPart, userIdPart, ok := dice.UnpackGroupUserId(id)
		if !ok {
			countFailed += 1
			fmt.Println("ID解析失败: ", string(id))
			continue
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Println("解析失败: ", string(data))
			continue
		}

		var cardType string
		if val, ok := mapData["$cardType"]; ok {
			cardType, _ = val.ReadString()
		}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if k == "$:cardBindMark" {
				// 绑卡标记 直接跳过
				continue
			}
			m.Store(k, v.ConvertToV2())
		}

		rawData, err := ds.VMValueNewDict(m).V().ToJSON()
		if err != nil {
			return count, countFailed, err
		}

		item := &model.AttributesItemModel{
			Id:        id,
			Data:      rawData,
			AttrsType: model.AttrsTypeGroupUser,

			// 当前组内绑定的卡
			BindingSheetId: sheetIdBindByGroupId[groupIdPart],

			// 这些是角色卡专用的
			Name:      "", // 群内默认卡，无名字，还是说以后弄成和nn的名字一致？
			OwnerId:   userIdPart,
			SheetType: cardType,
			IsHidden:  true,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}
		items = append(items, item)
		count += 1
	}

	return count, countFailed, nil
}

// 群数据转换
func attrsGroupMigrate(db *sqlx.DB) (int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from main.attrs_group", map[string]any{})
	if err != nil {
		return 0, 0, err
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
			return count, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			fmt.Println("解析失败: ", string(data))
			continue
		}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			m.Store(k, v.ConvertToV2())
		}

		rawData, err := ds.VMValueNewDict(m).V().ToJSON()
		if err != nil {
			return count, countFailed, err
		}

		item := &model.AttributesItemModel{
			Id:        id,
			Data:      rawData,
			AttrsType: model.AttrsTypeGroup,

			IsHidden: true,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}
		items = append(items, item)
		count += 1
	}

	return count, countFailed, nil
}

// 全局个人数据转换
func attrsUserMigrate(db *sqlx.DB) (int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_user where length(data) < 9000000", map[string]any{})
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	count := 0
	countFailed := 0
	var items []*model.AttributesItemModel
	for rows.Next() {
		var ownerId string
		var updatedAt int64
		var data []byte

		err := rows.Scan(
			&ownerId,
			&updatedAt,
			&data,
		)

		if err != nil {
			return count, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			continue
		}

		fmt.Println("数据转换-用户:", ownerId)
		var newSheetsList []*model.AttributesItemModel
		var sheetNameBindByGroupId = map[string]string{}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if strings.HasPrefix(k, "$:group-bind:") {
				// 这是绑卡信息，冒号后面的信息是GroupID，v是VMValue字符串
				// $:group-bind:群号  = 卡片名
				groupId := k[len("$:group-bind:"):]
				name, _ := v.ReadString()
				sheetNameBindByGroupId[groupId] = name
				fmt.Println("绑卡关联:", groupId, name)
				continue
			}
			if strings.HasPrefix(k, "$ch:") {
				// 处理角色卡，这里 v 是 string
				name := k[4:]

				toNew, err := convertToNew(name, ownerId, []byte(v.ToString()), updatedAt)
				if err != nil {
					return count, countFailed, err
				}
				newSheetsList = append(newSheetsList, toNew)
				continue
			}
			m.Store(k, v.ConvertToV2())
		}

		for _, i := range newSheetsList {
			// 一次性，双循环罢
			for groupId, j := range sheetNameBindByGroupId {
				if j == i.Name {
					// 这个东西等下群卡片迁移的时候使用，因此顺序不要错
					sheetIdBindByGroupId[groupId] = i.Id
				}
			}
		}

		rawData, err := ds.VMValueNewDict(m).V().ToJSON()
		if err != nil {
			return count, countFailed, err
		}

		item := &model.AttributesItemModel{
			Id:        ownerId,
			Data:      rawData,
			AttrsType: model.AttrsTypeUser,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}
		items = append(items, item)
		count += 1
	}

	return count, countFailed, nil
}

func V150Upgrade() {
	fmt.Println("1.5 数据转换迁移测试(不进行数据库写入)")
	dbDataPath, _ := filepath.Abs("./data/default/data.db")

	db, err := openDB(dbDataPath)
	if err != nil {
		fmt.Println("升级失败，无法打开数据库:", err)
		return
	}

	count, countFailed, err := attrsUserMigrate(db)
	fmt.Printf("数据卡转换 - 角色卡，成功%d 失败 %d\n", count, countFailed)
	if err != nil {
		fmt.Println("异常", err.Error())
	}

	count, countFailed, err = attrsGroupUserMigrate(db)
	fmt.Printf("数据卡转换 - 群组个人数据，成功%d 失败 %d\n", count, countFailed)
	if err != nil {
		fmt.Println("异常", err.Error())
	}

	count, countFailed, err = attrsGroupMigrate(db)
	fmt.Printf("数据卡转换 - 群数据，成功%d 失败 %d\n", count, countFailed)
	if err != nil {
		fmt.Println("异常", err.Error())
	}
}
