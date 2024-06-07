package migrate

import (
	"database/sql"
	"errors"
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
		// if name == "测试3" {
		// 	fmt.Println(m2)
		// }
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if k == "$:cardName" {
				continue
			}

			// if name == "测试3" {
			//	fmt.Println(k, v)
			// }
			m2.Store(k, v.ConvertToV2())
		}

		var rawData []byte
		rawData, err = ds.NewDictVal(m2).V().ToJSON()

		// if name == "测试3" {
		// 	fmt.Println("!!!!!", string(rawData))
		// 	fmt.Println(ds.NewDictVal(m2).V().ToString())
		// }
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
			fmt.Println("数据库读取出错，退出转换")
			fmt.Println("ID解析失败: ", id)
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

		rawData, err := ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Printf("群-用户 %s 的数据无法转换\n", id)
			continue
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

		_, err = model.AttrsNewItem(db, item)
		if err != nil {
			countFailed += 1
		} else {
			items = append(items, item)
			count += 1
		}
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
			fmt.Println("数据库读取出错，退出转换")
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

		rawData, err := ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Printf("群 %s 的数据无法转换\n", id)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        id,
			Data:      rawData,
			AttrsType: model.AttrsTypeGroup,

			IsHidden: true,

			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}

		_, err = model.AttrsNewItem(db, item)
		if err != nil {
			countFailed += 1
		} else {
			items = append(items, item)
			count += 1
		}
	}

	return count, countFailed, nil
}

// 全局个人数据转换、对应attrs_user和玩家人物卡
func attrsUserMigrate(db *sqlx.DB) (int, int, int, error) {
	rows, err := db.NamedQuery("select id, updated_at, data from attrs_user where length(data) < 9000000", map[string]any{})
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	count := 0
	countSheetsNum := 0
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
			fmt.Println("数据库读取出错，退出转换")
			return count, countSheetsNum, countFailed, err
		}

		mapData := make(map[string]*dice.VMValue)
		err = dice.JSONValueMapUnmarshal(data, &mapData)

		if err != nil {
			countFailed += 1
			continue
		}

		// fmt.Println("数据转换-用户:", ownerId)
		var newSheetsList []*model.AttributesItemModel
		var sheetNameBindByGroupId = map[string]string{}

		// 基础属性
		m := &ds.ValueMap{}
		for k, v := range mapData {
			if k == "$cardType" {
				continue
			}
			if k == "$:cardName" {
				continue
			}
			if strings.HasPrefix(k, "$:group-bind:") {
				// 这是绑卡信息，冒号后面的信息是GroupID，v是VMValue字符串
				// $:group-bind:群号  = 卡片名
				groupId := k[len("$:group-bind:"):]
				name, _ := v.ReadString()
				sheetNameBindByGroupId[groupId] = name
				// fmt.Println("绑卡关联:", groupId, name)
				continue
			}
			if strings.HasPrefix(k, "$ch:") {
				// 处理角色卡，这里 v 是 string
				var toNew *model.AttributesItemModel
				name := k[4:]

				toNew, err = convertToNew(name, ownerId, []byte(v.ToString()), updatedAt)
				if err != nil {
					fmt.Printf("用户 %s 的角色卡 %s 无法转换", ownerId, name)
					continue
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

		// 保存用户人物卡
		for _, i := range newSheetsList {
			_, err = model.AttrsNewItem(db, i)
			if err != nil {
				fmt.Printf("用户 %s 的角色卡 %s 无法写入数据库: %s\n", ownerId, i.Name, err.Error())
			}
		}

		countSheetsNum += len(newSheetsList)
		rawData, err := ds.NewDictVal(m).V().ToJSON()
		if err != nil {
			countFailed += 1
			fmt.Printf("用户 %s 的个人数据无法转换\n", ownerId)
			continue
		}

		item := &model.AttributesItemModel{
			Id:        ownerId,
			Data:      rawData,
			AttrsType: model.AttrsTypeUser,

			IsHidden:  true,
			CreatedAt: updatedAt,
			UpdatedAt: updatedAt,
		}

		_, err = model.AttrsNewItem(db, item)
		if err != nil {
			countFailed += 1
		} else {
			items = append(items, item)
			count += 1
		}
	}

	return count, countSheetsNum, countFailed, nil
}

func V150Upgrade() {
	dbDataPath, _ := filepath.Abs("./data/default/data.db")

	db, err := openDB(dbDataPath)
	if err != nil {
		fmt.Println("升级失败，无法打开数据库:", err)
		return
	}
	defer func() {
		_ = db.Close()
	}()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='attrs';").Scan(&name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// 表格不存在，继续执行
	case err != nil:
		fmt.Println("V150数据转换未知错误:", err.Error())
		return
	default:
		// 表格已经存在，说明转换完成，退出
		return
	}

	fmt.Println("1.5 数据迁移")

	sqls := []string{
		`CREATE TABLE IF NOT EXISTS endpoint_info (
user_id TEXT PRIMARY KEY,
cmd_num INTEGER,
cmd_last_time INTEGER,
online_time INTEGER,
updated_at INTEGER
);`,

		`
CREATE TABLE IF NOT EXISTS attrs (
    id TEXT PRIMARY KEY,
    data BYTEA,
    attrs_type TEXT,

	-- 坏，Get这个方法太严格了，所有的字段都要有默认值，不然无法反序列化
	binding_sheet_id TEXT default '',

    name TEXT default '',
    owner_id TEXT default '',
    sheet_type TEXT default '',
    is_hidden BOOLEAN default FALSE,

    created_at INTEGER default 0,
    updated_at INTEGER  default 0
);
`,
		`create index if not exists idx_attrs_binding_sheet_id on attrs (binding_sheet_id);`,
		`create index if not exists idx_attrs_owner_id_id on attrs (owner_id);`,
		`create index if not exists idx_attrs_attrs_type_id on attrs (attrs_type);`,
	}
	for _, i := range sqls {
		_, _ = db.Exec(i)
	}

	count, countSheetsNum, countFailed, err := attrsUserMigrate(db)
	fmt.Printf("数据卡转换 - 角色卡，成功人数%d 失败人数 %d 卡数 %d\n", count, countFailed, countSheetsNum)
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

	// 删档
	fmt.Println("删除旧版本数据")
	_, _ = db.Exec("drop table attrs_group")
	_, _ = db.Exec("drop table attrs_group_user")
	_, _ = db.Exec("drop table attrs_user")
	_, err = db.Exec("VACUUM;") // 收尾
	fmt.Println("V150 数据转换完成")
}
